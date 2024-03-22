package bitcoin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/internal/types"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/ethereum/go-ethereum"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"gorm.io/gorm"
)

const (
	BridgeDepositServiceName = "BitcoinBridgeDepositService"
	BatchDepositWaitTimeout  = 10 * time.Second
	DepositErrTimeout        = 30 * time.Second
	BatchDepositLimit        = 100
	WaitMinedTimeout         = 2 * time.Minute
	HandleDepositTimeout     = 1 * time.Second
	DepositRetry             = 10 // temp fix, Increase retry times
)

var ErrServerStop = errors.New("server stop")

// BridgeDepositService l1->l2
type BridgeDepositService struct {
	service.BaseService

	bridge     types.BITCOINBridge
	btcIndexer types.BITCOINTxIndexer
	db         *gorm.DB
	log        log.Logger
	wg         sync.WaitGroup
	stopChan   chan struct{}
}

// NewBridgeDepositService returns a new service instance.
func NewBridgeDepositService(
	bridge types.BITCOINBridge,
	btcIndexer types.BITCOINTxIndexer,
	db *gorm.DB,
	logger log.Logger,
) *BridgeDepositService {
	is := &BridgeDepositService{
		bridge:     bridge,
		btcIndexer: btcIndexer,
		db:         db,
		log:        logger,
	}
	is.BaseService = *service.NewBaseService(nil, BridgeDepositServiceName, is)
	return is
}

// OnStart
func (bis *BridgeDepositService) OnStart() error {
	bis.wg.Add(2)
	go bis.Deposit()
	// TODO: check rollup indexer tx status
	go bis.CheckDeposit()
	bis.stopChan = make(chan struct{})
	select {}
}

func (bis *BridgeDepositService) OnStop() {
	bis.log.Warnf("bridge deposit service stoping...")
	close(bis.stopChan)
	bis.wg.Wait()
}

func (bis *BridgeDepositService) Deposit() {
	defer bis.wg.Done()
	ticker := time.NewTicker(BatchDepositWaitTimeout)
	for {
	DEPOSIT:
		select {
		case <-bis.stopChan:
			bis.log.Warnf("deposit stopping...")
			return
		case <-ticker.C:
			// Priority processing UnconfirmedDeposit
			err := bis.UnconfirmedDeposit()
			if err != nil {
				bis.log.Warnf("unconfirmed deposit err: %s", err)
				if errors.Is(err, ErrServerStop) {
					return
				}
				continue
			}

			if bis.bridge.EnableEoaTransfer() {
				err := bis.HandleEoaTransfer()
				if err != nil {
					bis.log.Warnf("HandleEoaTransfer err: %s", err)
					if errors.Is(err, ErrServerStop) {
						return
					}
					continue
				}
			}

			// Query condition
			// 1. tx status is pending
			// 2. contract insufficient balance
			// 3. invoke contract from account insufficient balance
			// 4. callback status is success
			// 5. listener status is success
			var deposits []model.Deposit
			err = bis.db.
				Where(
					fmt.Sprintf("%s.%s IN (?)", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxStatus),
					[]int{
						model.DepositB2TxStatusPending,
						model.DepositB2TxStatusInsufficientBalance,
						model.DepositB2TxStatusFromAccountGasInsufficient,
					},
				).
				Where(
					fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().CallbackStatus),
					model.CallbackStatusSuccess,
				).
				Where(
					fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().ListenerStatus),
					model.ListenerStatusSuccess,
				).
				Limit(BatchDepositLimit).
				Order(fmt.Sprintf("%s.%s ASC", model.Deposit{}.TableName(), model.Deposit{}.Column().BtcBlockNumber)).
				Order(fmt.Sprintf("%s.%s ASC", model.Deposit{}.TableName(), "id")).
				Find(&deposits).
				Error
			if err != nil {
				bis.log.Errorw("failed find tx from db", "error", err)
			}

			bis.log.Infow("start handle deposit", "deposit batch num", len(deposits))
			for _, deposit := range deposits {
				err = bis.HandleDeposit(deposit, nil, deposit.B2TxNonce)
				if err != nil {
					bis.log.Errorw("handle deposit failed", "error", err, "deposit", deposit)
					if errors.Is(err, ErrServerStop) {
						return
					}
					break DEPOSIT
				}
				timeoutTicker := time.NewTicker(HandleDepositTimeout)
				select {
				case <-bis.stopChan:
					bis.log.Warnf("handle deposit stopping...")
					return
				case <-timeoutTicker.C:
				}
			}

			// handle aa not found err
			// If there is no binding between the registered address and pubkey
			// an error will occur, which can be handled again next time
			var aaNotFoundDeposits []model.Deposit
			err = bis.db.
				Where(
					fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxStatus),
					model.DepositB2TxStatusAAAddressNotFound,
				).
				Limit(BatchDepositLimit).
				Order(fmt.Sprintf("%s.%s ASC", model.Deposit{}.TableName(), model.Deposit{}.Column().BtcBlockNumber)).
				Order(fmt.Sprintf("%s.%s ASC", model.Deposit{}.TableName(), "id")).
				Find(&aaNotFoundDeposits).Error
			if err != nil {
				bis.log.Errorw("failed find tx from db", "error", err)
			}

			bis.log.Infow("start handle aa not found deposit", "aa not found deposit batch num", len(aaNotFoundDeposits))
			for _, deposit := range aaNotFoundDeposits {
				err = bis.HandleDeposit(deposit, nil, deposit.B2TxNonce)
				if err != nil {
					bis.log.Errorw("handle aa not found deposit failed", "error", err, "deposit", deposit)
					if errors.Is(err, ErrServerStop) {
						return
					}
					break DEPOSIT
				}
				timeoutTicker := time.NewTicker(HandleDepositTimeout)
				select {
				case <-bis.stopChan:
					bis.log.Warnf("handle aa not found deposit stopping...")
					return
				case <-timeoutTicker.C:
				}
			}
		}
	}
}

func (bis *BridgeDepositService) UnconfirmedDeposit() error {
	var deposits []model.Deposit
	err := bis.db.
		Where(
			fmt.Sprintf("%s.%s IN (?)", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxStatus),
			[]int{
				model.DepositB2TxStatusContextDeadlineExceeded,
				model.DepositB2TxStatusWaitMined,
				model.DepositB2TxStatusWaitMinedFailed,
				model.DepositB2TxStatusIsPending,
			},
		).
		Order(fmt.Sprintf("%s.%s ASC", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxNonce)).
		Find(&deposits).Error
	if err != nil {
		bis.log.Errorw("failed find tx from db", "error", err)
		return err
	}

	bis.log.Infow("start handle unconfirmed deposit", "unconfirmed deposit batch num", len(deposits))
	for _, deposit := range deposits {
		err = bis.HandleUnconfirmedDeposit(deposit)
		if err != nil {
			bis.log.Errorw("handle unconfirmed failed", "error", err, "deposit", deposit)
			return err
		}

		timeoutTicker := time.NewTicker(HandleDepositTimeout)
		select {
		case <-bis.stopChan:
			bis.log.Warnf("unconfirmed deposit stopping...")
			return ErrServerStop
		case <-timeoutTicker.C:
		}
	}
	return nil
}

func (bis *BridgeDepositService) HandleDeposit(deposit model.Deposit, oldTx *ethTypes.Transaction, nonce uint64) error {
	defer func() {
		if err := recover(); err != nil {
			bis.log.Errorw("panic err", err)
		}
	}()

	if oldTx != nil {
		bis.log.Warnw("handle old deposit", "old tx:", oldTx)
	}

	// check Confirmations
	err := bis.btcIndexer.CheckConfirmations(deposit.BtcTxHash)
	if err != nil {
		bis.log.Errorw("check btc tx confirmations err", "tx hash:", deposit.B2TxHash, "err:", err)
		return err
	}

	// send deposit tx
	b2Tx, _, aaAddress, err := bis.bridge.Deposit(deposit.BtcTxHash, types.BitcoinFrom{
		Address: deposit.BtcFrom,
	}, deposit.BtcValue, oldTx, nonce)
	if err != nil {
		switch {
		case errors.Is(err, ErrBridgeDepositTxHashExist):
			deposit.B2TxStatus = model.DepositB2TxStatusTxHashExist
			bis.log.Errorw("invoke deposit send tx hash exist",
				"error", err.Error(),
				"btcTxHash", deposit.BtcTxHash,
				"data", deposit)
			if deposit.B2TxHash != "" {
				receipt, err := bis.bridge.TransactionReceipt(deposit.B2TxHash)
				if err != nil {
					return err
				}
				if receipt.Status == 1 {
					deposit.B2TxStatus = model.DepositB2TxStatusSuccess
				} else {
					deposit.B2TxStatus = model.DepositB2TxStatusWaitMinedStatusFailed
				}
			}
		case errors.Is(err, ErrBridgeDepositContractInsufficientBalance):
			deposit.B2TxStatus = model.DepositB2TxStatusInsufficientBalance
			bis.log.Errorw("invoke deposit send tx contract insufficient balance",
				"error", err.Error(),
				"btcTxHash", deposit.BtcTxHash,
				"data", deposit)
		case errors.Is(err, ErrBridgeFromGasInsufficient):
			deposit.B2TxStatus = model.DepositB2TxStatusFromAccountGasInsufficient
			bis.log.Errorw("invoke deposit send tx from account gas insufficient",
				"error", err.Error(),
				"btcTxHash", deposit.BtcTxHash,
				"data", deposit)
		case errors.Is(err, ErrAAAddressNotFound):
			deposit.B2TxStatus = model.DepositB2TxStatusAAAddressNotFound
			bis.log.Errorw("invoke deposit send tx aa address not found",
				"error", err.Error(),
				"btcTxHash", deposit.BtcTxHash,
				"data", deposit)
		case errors.Is(err, errors.New("already known")):

			if deposit.B2TxHash != "" {
				deposit.B2TxStatus = model.DepositB2TxStatusIsPending
				bis.log.Errorw("invoke deposit send tx already known",
					"error", err.Error(),
					"btcTxHash", deposit.BtcTxHash,
					"data", deposit)
			}

		default:
			deposit.B2TxRetry++
			deposit.B2TxStatus = model.DepositB2TxStatusPending
			bis.log.Errorw("invoke deposit send tx retry",
				"error", err.Error(),
				"btcTxHash", deposit.BtcTxHash,
				"data", deposit)
			// The call may not succeed due to network reasons. sleep wait for a while
			err = bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(map[string]interface{}{
				model.Deposit{}.Column().B2TxStatus: deposit.B2TxStatus,
				model.Deposit{}.Column().B2TxRetry:  deposit.B2TxRetry,
			}).Error
			if err != nil {
				return err
			}
			tryTicker := time.NewTicker(DepositErrTimeout)
			select {
			case <-bis.stopChan:
				return ErrServerStop
			case <-tryTicker.C:
				return fmt.Errorf("retry handle deposit")
			}
		}
		dbErr := bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(map[string]interface{}{
			model.Deposit{}.Column().B2TxStatus: deposit.B2TxStatus,
		}).Error
		if dbErr != nil {
			return dbErr
		}
		return err
	}
	deposit.B2TxStatus = model.DepositB2TxStatusWaitMined
	deposit.B2TxHash = b2Tx.Hash().String()
	deposit.BtcFromAAAddress = aaAddress
	deposit.B2TxNonce = b2Tx.Nonce()
	updateFields := map[string]interface{}{
		model.Deposit{}.Column().B2TxHash:         deposit.B2TxHash,
		model.Deposit{}.Column().BtcFromAAAddress: deposit.BtcFromAAAddress,
		model.Deposit{}.Column().B2TxStatus:       deposit.B2TxStatus,
		model.Deposit{}.Column().B2TxNonce:        deposit.B2TxNonce,
	}
	err = bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(updateFields).Error
	if err != nil {
		return err
	}

	bis.log.Infow("invoke deposit send tx success, wait confirm",
		"data", deposit)

	// wait tx mined, may be wait long time so set timeout ctx
	ctx1, cancel1 := context.WithTimeout(context.Background(), WaitMinedTimeout)
	defer cancel1()
	errCh := make(chan error)
	go func(ctx1 context.Context, b2Tx *ethTypes.Transaction, deposit model.Deposit) {
		err := bis.WaitMined(ctx1, b2Tx, deposit)
		if err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}(ctx1, b2Tx, deposit)
	waitMinedTicker := time.NewTicker(WaitMinedTimeout + (10 * time.Second))
	for {
		bis.log.Warn("wait mined")
		select {
		case err = <-errCh:
			if err != nil {
				bis.log.Errorw("wait tx mined err", "error", err)
			}
			return err
		case <-bis.stopChan:
			bis.log.Errorw("wait tx mined stop chan", "error", err)
			cancel1()
			return ErrServerStop
		case <-waitMinedTicker.C:
			bis.log.Errorw("wait tx mined timeout", "error", err)
		}
	}
}

// HandleUnconfirmedDeposit
// 1. tx mined, update status
// 2. tx not mined, isPending, need reset gasprice
// 3. tx not mined, tx not mempool, need retry send tx
func (bis *BridgeDepositService) HandleUnconfirmedDeposit(deposit model.Deposit) error {
	txReceipt, err := bis.bridge.TransactionReceipt(deposit.B2TxHash)
	if err == nil {
		// case 1
		updateFields := map[string]interface{}{}
		if txReceipt.Status == 1 {
			updateFields[model.Deposit{}.Column().B2TxStatus] = model.DepositB2TxStatusSuccess
		} else {
			updateFields[model.Deposit{}.Column().B2TxStatus] = model.DepositB2TxStatusWaitMinedStatusFailed
		}

		dbErr := bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(updateFields).Error
		if err != nil {
			return dbErr
		}
		return err
	}
	if err != nil {
		bis.log.Errorw("TransactionReceipt err", "error", err, "data", deposit)
		if errors.Is(err, ethereum.NotFound) {
			bis.log.Errorf("TransactionReceipt not found")
			// tx in mempool, isPending
			tx, isPending, err := bis.bridge.TransactionByHash(deposit.B2TxHash)
			if err != nil {
				if errors.Is(err, ethereum.NotFound) {
					// case 3
					bis.log.Errorf("TransactionByHash not found, try send tx by nonce")
					return bis.HandleDeposit(deposit, nil, deposit.B2TxNonce)
				}
				return err
			}
			if isPending {
				// case 2
				bis.log.Warnw("tx is pending retry", "old", tx, "deposit", deposit)
				return bis.HandleDeposit(deposit, tx, 0)
			}
		}
		return err
	}
	return nil
}

func (bis *BridgeDepositService) HandleEoaTransfer() error {
	var deposits []model.Deposit
	err := bis.db.
		Where(
			fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxStatus),
			model.DepositB2TxStatusWaitMinedStatusFailed,
		).
		Where(
			fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().B2EoaTxStatus),
			model.DepositB2EoaTxStatusPending,
		).
		Where(
			fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().B2EoaTxNonce),
			0,
		).
		Order(fmt.Sprintf("%s.%s ASC", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxNonce)).
		Find(&deposits).Error
	if err != nil {
		bis.log.Errorw("failed find tx from db", "error", err)
		return err
	}

	bis.log.Infow("start handle eoaTransfer deposit", "eoaTransfer deposit batch num", len(deposits))
	for _, deposit := range deposits {
		err = bis.EoaTransfer(deposit)
		if err != nil {
			bis.log.Errorw("handle eoaTransfer failed", "error", err, "deposit", deposit)
			return err
		}

		timeoutTicker := time.NewTicker(HandleDepositTimeout)
		select {
		case <-bis.stopChan:
			bis.log.Warnf("eoaTransfer deposit stopping...")
			return ErrServerStop
		case <-timeoutTicker.C:
		}
	}
	return nil
}

func (bis *BridgeDepositService) EoaTransfer(deposit model.Deposit) error {
	b2EoaTx, err := bis.bridge.Transfer(types.BitcoinFrom{
		Address: deposit.BtcFrom,
	}, deposit.BtcValue)
	if err != nil {
		bis.log.Errorw("invoke eoa transfer tx unknown err",
			"error", err.Error(),
			"btcTxHash", deposit.BtcTxHash,
			"data", deposit)

		dbErr := bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(map[string]interface{}{
			model.Deposit{}.Column().B2EoaTxStatus: model.DepositB2EoaTxStatusFailed,
		}).Error
		if dbErr != nil {
			return dbErr
		}
		return err
	}
	err = bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(map[string]interface{}{
		model.Deposit{}.Column().B2EoaTxHash:   b2EoaTx.Hash().String(),
		model.Deposit{}.Column().B2EoaTxNonce:  b2EoaTx.Nonce(),
		model.Deposit{}.Column().B2EoaTxStatus: model.DepositB2EoaTxStatusWaitMinedFailed,
	}).Error
	if err != nil {
		return err
	}
	// eoa wait mined
	ctx2, cancel2 := context.WithTimeout(context.Background(), WaitMinedTimeout)
	defer cancel2()
	_, err = bis.bridge.WaitMined(ctx2, b2EoaTx, nil)
	if err != nil {
		deposit.B2EoaTxStatus = model.DepositB2EoaTxStatusWaitMinedFailed
		bis.log.Errorw("invoke eoa transfer wait mined err",
			"error", err.Error(),
			"btcTxHash", deposit.BtcTxHash,
			"data", deposit)

		if errors.Is(err, context.DeadlineExceeded) {
			deposit.B2EoaTxStatus = model.DepositB2EoaTxStatusContextDeadlineExceeded
			bis.log.Error("invoke eoa transfer wait mined context deadline exceeded")
		}
	} else {
		deposit.B2EoaTxStatus = model.DepositB2EoaTxStatusSuccess
		bis.log.Infow("invoke eoa transfer success",
			"btcTxHash", deposit.BtcTxHash,
			"data", deposit)
	}
	err = bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(map[string]interface{}{
		model.Deposit{}.Column().B2EoaTxStatus: deposit.B2EoaTxStatus,
	}).Error
	if err != nil {
		return err
	}
	return nil
}

func (bis *BridgeDepositService) WaitMined(ctx1 context.Context, b2Tx *ethTypes.Transaction, deposit model.Deposit) error {
	b2txReceipt, err := bis.bridge.WaitMined(ctx1, b2Tx, nil)
	if err != nil {
		switch {
		case errors.Is(err, ErrBridgeWaitMinedStatus):
			deposit.B2TxStatus = model.DepositB2TxStatusWaitMinedStatusFailed
			bis.log.Errorw("invoke deposit wait mined err, status != 1",
				"btcTxHash", deposit.BtcTxHash,
				"b2txReceipt", b2txReceipt,
				"data", deposit)
		case errors.Is(err, context.DeadlineExceeded):
			// handle ctx deadline timeout
			// Indicates that the chain is unavailable at this time
			// This particular error needs to be recorded and handled manually
			deposit.B2TxStatus = model.DepositB2TxStatusContextDeadlineExceeded
			bis.log.Errorw("invoke deposit wait mined context deadline exceeded",
				"error", err.Error(),
				"btcTxHash", deposit.BtcTxHash,
				"data", deposit)
		default:
			deposit.B2TxStatus = model.DepositB2TxStatusWaitMinedFailed
			bis.log.Errorw("invoke deposit wait mined unknown err",
				"error", err.Error(),
				"btcTxHash", deposit.BtcTxHash,
				"data", deposit)
		}
	} else {
		deposit.B2TxStatus = model.DepositB2TxStatusSuccess
	}
	err = bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(map[string]interface{}{
		model.Deposit{}.Column().B2TxStatus: deposit.B2TxStatus,
	}).Error
	if err != nil {
		return err
	}
	if deposit.B2TxStatus == model.DepositB2TxStatusSuccess {
		bis.log.Infow("handle deposit success", "btcTxHash", deposit.BtcTxHash, "deposit", deposit)
	} else {
		bis.log.Errorw("handle deposit failed", "btcTxHash", deposit.BtcTxHash, "deposit", deposit)
		return fmt.Errorf("wait mined err b2_tx_status: %v", deposit.B2TxStatus)
	}
	return nil
}

func (bis *BridgeDepositService) CheckDeposit() {
	defer bis.wg.Done()
	ticker := time.NewTicker(BatchDepositWaitTimeout)
	for {
		select {
		case <-bis.stopChan:
			bis.log.Warnf("check deposit stopping...")
			return
		case <-ticker.C:
			var deposits []model.Deposit
			err := bis.db.
				Where(
					fmt.Sprintf("%s.%s IN (?)", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxStatus),
					[]int{
						model.DepositB2TxStatusSuccess,
						model.DepositB2TxStatusTxHashExist,
					},
				).
				Where(
					fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxCheck),
					model.B2CheckStatusPending,
				).
				Limit(BatchDepositLimit).
				Order(fmt.Sprintf("%s.%s ASC", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxNonce)).
				Find(&deposits).
				Error
			if err != nil {
				bis.log.Errorw("failed find tx from db", "error", err)
				continue
			}

			for _, deposit := range deposits {
				timeoutTicker := time.NewTicker(2 * time.Minute)
				select {
				case <-bis.stopChan:
					bis.log.Warnf("handle deposit stopping...")
					return
				case <-timeoutTicker.C:
				}
				var rollupDeposit model.RollupDeposit
				if deposit.B2TxStatus == model.DepositB2TxStatusSuccess {
					err = bis.db.
						Where(
							fmt.Sprintf("%s.%s = ?", model.RollupDeposit{}.TableName(), model.Deposit{}.Column().BtcTxHash),
							deposit.BtcTxHash,
						).
						First(&rollupDeposit).Error
					if err != nil {
						bis.log.Errorw("find rollup deposit error", "err", err, "deposit", deposit)
						continue
					}
					if strings.EqualFold(deposit.BtcFromAAAddress, rollupDeposit.BtcFromAAAddress) &&
						deposit.BtcValue == rollupDeposit.BtcValue {
						deposit.B2TxCheck = model.B2CheckStatusSuccess
					} else {
						deposit.B2TxCheck = model.B2CheckStatusFailed
					}
					err = bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(map[string]interface{}{
						model.Deposit{}.Column().B2TxCheck: deposit.B2TxCheck,
					}).Error
					if err != nil {
						bis.log.Errorw("update deposit error", "err", err)
					}
				}
			}
		}
	}
}
