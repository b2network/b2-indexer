package bitcoin

import (
	"context"
	"errors"
	"fmt"
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
	DepositErrTimeout        = 10 * time.Minute
	BatchDepositLimit        = 100
	WaitMinedTimeout         = 2 * time.Hour
	HandleDepositTimeout     = 1 * time.Second
	DepositRetry             = 10 // temp fix, Increase retry times
)

var (
	serverStopErr = errors.New("server stop")
)

// BridgeDepositService l1->l2
type BridgeDepositService struct {
	service.BaseService

	bridge          types.BITCOINBridge
	btcIndexer      types.BITCOINTxIndexer
	db              *gorm.DB
	log             log.Logger
	wg              sync.WaitGroup
	stopChan        chan struct{}
	deadlineCancel1 context.CancelFunc
	deadlineCancel2 context.CancelFunc
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
	go bis.UnconfirmedDeposit()
	// TODO: retry max err
	bis.stopChan = make(chan struct{})
	select {}
}

func (bis *BridgeDepositService) OnStop() {
	bis.log.Warnf("bridge deposit service stoping...")
	close(bis.stopChan)
	bis.wg.Wait()
	return
}

func (bis *BridgeDepositService) Deposit() {
	defer bis.wg.Done()
	ticker := time.NewTicker(BatchDepositWaitTimeout)
	for {
		select {
		case <-bis.stopChan:
			bis.log.Warnf("deposit stopping...")
			// TODO: close db, deadline handle
			return
		case <-ticker.C:
			// Query condition
			// 1. tx status is pending
			// 2. contract insufficient balance
			// 3. invoke contract from account insufficient balance
			// 4. callback status is success
			// 5. listener status is success
			var deposits []model.Deposit
			err := bis.db.
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
					if errors.Is(err, serverStopErr) {
						return
					}
				}
				select {
				case <-bis.stopChan:
					bis.log.Warnf("handle deposit stopping...")
					return
				default:
				}
				time.Sleep(HandleDepositTimeout)
			}

			// handle aa not found err
			// If there is no binding between the registered address and pubkey
			// an error will occur, which can be handled again next time
			var aaNotFoundDeposits []model.Deposit
			err = bis.db.
				Where(
					fmt.Sprintf("%s.%s IN (?)", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxStatus),
					[]int{
						model.DepositB2TxStatusAAAddressNotFound,
					},
				).
				Limit(BatchDepositLimit).
				Find(&aaNotFoundDeposits).Error
			if err != nil {
				bis.log.Errorw("failed find tx from db", "error", err)
			}

			bis.log.Infow("start handle aa not found deposit", "aa not found deposit batch num", len(aaNotFoundDeposits))
			for _, deposit := range aaNotFoundDeposits {
				err = bis.HandleDeposit(deposit, nil, deposit.B2TxNonce)
				if err != nil {
					bis.log.Errorw("handle aa not found deposit failed", "error", err, "deposit", deposit)
					if errors.Is(err, serverStopErr) {
						return
					}
				}
				select {
				case <-bis.stopChan:
					bis.log.Warnf("handle aa not found deposit stopping...")
					return
				default:
				}
				time.Sleep(HandleDepositTimeout)
			}
		}
	}
}

func (bis *BridgeDepositService) UnconfirmedDeposit() {
	defer bis.wg.Done()
	ticker := time.NewTicker(BatchDepositWaitTimeout)
	for {
		select {
		case <-bis.stopChan:
			bis.log.Warnf("unconfirmed deposit stopping...")
			time.Sleep(10 * time.Second)
			return
		case <-ticker.C:
			var deposits []model.Deposit
			err := bis.db.
				Where(
					fmt.Sprintf("%s.%s IN (?)", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxStatus),
					[]int{
						model.DepositB2TxStatusContextDeadlineExceeded,
						model.DepositB2TxStatusWaitMined,
						model.DepositB2TxStatusWaitMinedFailed,
					},
				).
				Limit(BatchDepositLimit).
				Order(fmt.Sprintf("%s.%s ASC", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxNonce)).
				Find(&deposits).Error
			if err != nil {
				bis.log.Errorw("failed find tx from db", "error", err)
			}

			bis.log.Infow("start handle unconfirmed deposit", "unconfirmed deposit batch num", len(deposits))
			for _, deposit := range deposits {
				err = bis.HandleUnconfirmedDeposit(deposit)
				if err != nil {
					bis.log.Errorw("handle unconfirmed failed", "error", err, "deposit", deposit)
				}
				select {
				case <-bis.stopChan:
					bis.log.Warnf("unconfirmed deposit stopping...")
					return
				default:
				}
				time.Sleep(HandleDepositTimeout)
			}
		}
	}
}

func (bis *BridgeDepositService) HandleDeposit(deposit model.Deposit, oldTx *ethTypes.Transaction, nonce uint64) error {
	defer func() {
		if err := recover(); err != nil {
			bis.log.Errorw("panic err", err)
		}
	}()
	// set init status
	deposit.B2EoaTxStatus = model.DepositB2EoaTxStatusPending

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
		case errors.Is(err, ErrBrdigeDepositTxHashExist):
			deposit.B2TxStatus = model.DepositB2TxStatusTxHashExist
			bis.log.Errorw("invoke deposit send tx hash exist",
				"error", err.Error(),
				"btcTxHash", deposit.BtcTxHash,
				"data", deposit)
		case errors.Is(err, ErrBrdigeDepositContractInsufficientBalance):
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
		// case errors.Is(err, errors.New("nonce too low")):

		// TODO: handle other err
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
			select {
			case <-bis.stopChan:
				return serverStopErr
			case <-time.After(DepositErrTimeout):
				return fmt.Errorf("retry handle deposit")
			}
		}
	} else {
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
		select {
		case <-bis.stopChan:
			err = bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(map[string]interface{}{
				model.Deposit{}.Column().B2TxStatus: model.DepositB2TxStatusContextDeadlineExceeded,
			}).Error
			if err != nil {
				return err
			}
			return serverStopErr
		default:
			b2txReceipt, err := bis.bridge.WaitMined(ctx1, b2Tx, nil)
			if err != nil {
				switch {
				case errors.Is(err, ErrBridgeWaitMinedStatus):
					deposit.B2TxStatus = model.DepositB2TxStatusWaitMinedStatusFailed
					bis.log.Errorw("invoke deposit wait mined err, status != 1",
						"btcTxHash", deposit.BtcTxHash,
						"b2txReceipt", b2txReceipt,
						"data", deposit)
					if bis.bridge.EnableEoaTransfer() {
						// try eoa transfer, only b2tx recepit status != 1
						// NOTE: eoa tx is temp handle, It will be removed in the future
						bis.log.Errorw("invoke deposit wait mined err try again by eoa transfer",
							"btcTxHash", deposit.BtcTxHash,
							"b2txReceipt", b2txReceipt,
							"data", deposit)
						deposit.B2EoaTxHash, deposit.B2EoaTxStatus, deposit.B2EoaTxNonce = bis.EoaTransfer(deposit)
					}
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
		}

	}

	updateFields := map[string]interface{}{
		model.Deposit{}.Column().B2TxHash:         deposit.B2TxHash,
		model.Deposit{}.Column().BtcFromAAAddress: deposit.BtcFromAAAddress,
		model.Deposit{}.Column().B2TxStatus:       deposit.B2TxStatus,
		model.Deposit{}.Column().B2TxRetry:        deposit.B2TxRetry,
		model.Deposit{}.Column().B2EoaTxHash:      deposit.B2EoaTxHash,
		model.Deposit{}.Column().B2EoaTxStatus:    deposit.B2EoaTxStatus,
		model.Deposit{}.Column().B2EoaTxNonce:     deposit.B2EoaTxNonce,
		model.Deposit{}.Column().B2TxNonce:        deposit.B2TxNonce,
	}
	err = bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(updateFields).Error
	if err != nil {
		return err
	}
	bis.log.Infow("handle deposit success", "btcTxHash", deposit.BtcTxHash, "deposit", deposit)
	return nil
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
			if bis.bridge.EnableEoaTransfer() {
				b2EoaTxHash, b2EoaTxStatus, b2EoaTxNonce := bis.EoaTransfer(deposit)
				updateFields[model.Deposit{}.Column().B2EoaTxHash] = b2EoaTxHash
				updateFields[model.Deposit{}.Column().B2EoaTxStatus] = b2EoaTxStatus
				updateFields[model.Deposit{}.Column().B2EoaTxNonce] = b2EoaTxNonce
			}
		}

		err = bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(updateFields).Error
		if err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		bis.log.Errorw("TransactionReceipt err", "error", err)
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
				return bis.HandleDeposit(deposit, tx, 0)
			}
		}
		return err
	}
	return nil
}

func (bis *BridgeDepositService) EoaTransfer(deposit model.Deposit) (string, int, uint64) {
	b2EoaTx, err := bis.bridge.Transfer(types.BitcoinFrom{
		Address: deposit.BtcFrom,
	}, deposit.BtcValue)
	if err != nil {
		deposit.B2EoaTxStatus = model.DepositB2EoaTxStatusFailed
		bis.log.Errorw("invoke eoa transfer tx unknown err",
			"error", err.Error(),
			"btcTxHash", deposit.BtcTxHash,
			"data", deposit)
	} else {
		deposit.B2EoaTxHash = b2EoaTx.Hash().String()
		deposit.B2EoaTxNonce = b2EoaTx.Nonce()
		// eoa wait mined
		ctx2, cancel2 := context.WithTimeout(context.Background(), WaitMinedTimeout)
		defer cancel2()
		_, err := bis.bridge.WaitMined(ctx2, b2EoaTx, nil)
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
	}
	return deposit.B2EoaTxHash, deposit.B2EoaTxStatus, deposit.B2EoaTxNonce
}
