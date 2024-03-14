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

// BridgeDepositService l1->l2
type BridgeDepositService struct {
	service.BaseService

	bridge types.BITCOINBridge

	db  *gorm.DB
	log log.Logger
}

// NewBridgeDepositService returns a new service instance.
func NewBridgeDepositService(
	bridge types.BITCOINBridge,
	db *gorm.DB,
	logger log.Logger,
) *BridgeDepositService {
	is := &BridgeDepositService{bridge: bridge, db: db, log: logger}
	is.BaseService = *service.NewBaseService(nil, BridgeDepositServiceName, is)
	return is
}

// OnStart
func (bis *BridgeDepositService) OnStart() error {
	var depositWg sync.WaitGroup
	depositWg.Add(2)
	go bis.Deposit(&depositWg)
	go bis.DeadlineExceededDeposit(&depositWg)
	depositWg.Wait()
	return nil
}

func (bis *BridgeDepositService) Deposit(wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(BatchDepositWaitTimeout)
	for {
		<-ticker.C
		ticker.Reset(BatchDepositWaitTimeout)
		// Query condition
		// 1. tx status is pending
		// 2. contract insufficient balance
		// 3. invoke contract from account insufficient balance
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
			Limit(BatchDepositLimit).
			Find(&deposits).Error
		if err != nil {
			bis.log.Errorw("failed find tx from db", "error", err)
		}

		bis.log.Infow("start handle deposit", "deposit batch num", len(deposits))
		for _, deposit := range deposits {
			err = bis.HandleDeposit(deposit, nil)
			if err != nil {
				bis.log.Errorw("handle deposit failed", "error", err, "deposit", deposit)
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
			err = bis.HandleDeposit(deposit, nil)
			if err != nil {
				bis.log.Errorw("handle aa not found deposit failed", "error", err, "deposit", deposit)
			}

			time.Sleep(HandleDepositTimeout)
		}
	}
}

func (bis *BridgeDepositService) DeadlineExceededDeposit(wg *sync.WaitGroup) {
	wg.Done()
	ticker := time.NewTicker(BatchDepositWaitTimeout)
	for {
		<-ticker.C
		ticker.Reset(BatchDepositWaitTimeout)
		var deposits []model.Deposit
		err := bis.db.
			Where(
				fmt.Sprintf("%s.%s IN (?)", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxStatus),
				[]int{
					model.DepositB2TxStatusContextDeadlineExceeded,
				},
			).
			Limit(BatchDepositLimit).
			Find(&deposits).Error
		if err != nil {
			bis.log.Errorw("failed find tx from db", "error", err)
		}

		bis.log.Infow("start handle deadline deposit", "deposit batch num", len(deposits))
		for _, deposit := range deposits {
			err = bis.HandleDeadlineDeposit(deposit)
			if err != nil {
				bis.log.Errorw("handle deadline deposit failed", "error", err, "deposit", deposit)
			}

			time.Sleep(HandleDepositTimeout)
		}
	}
}

func (bis *BridgeDepositService) HandleDeposit(deposit model.Deposit, oldTx *ethTypes.Transaction) error {
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

	// send deposit tx
	// TODO: wait tx mined
	b2Tx, _, aaAddress, err := bis.bridge.Deposit(deposit.BtcTxHash, types.BitcoinFrom{
		Address: deposit.BtcFrom,
	}, deposit.BtcValue, oldTx)
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
			// TODO: if nonce used
		default:
			deposit.B2TxRetry++
			deposit.B2TxStatus = model.DepositB2TxStatusPending
			bis.log.Errorw("invoke deposit send tx retry",
				"error", err.Error(),
				"btcTxHash", deposit.BtcTxHash,
				"data", deposit)
			// The call may not succeed due to network reasons. sleep wait for a while
			time.Sleep(DepositErrTimeout)
		}
	} else {
		deposit.B2TxStatus = model.DepositB2TxStatusWaitMined
		deposit.B2TxHash = b2Tx.Hash().String()
		deposit.BtcFromAAAddress = aaAddress

		bis.log.Infow("invoke deposit send tx success, wait confirm",
			"data", deposit)

		// wait tx mined, may be wait long time so set timeout ctx
		ctx1, cancel1 := context.WithTimeout(context.Background(), WaitMinedTimeout)
		defer cancel1()
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
		}
	}

	updateFields := map[string]interface{}{
		model.Deposit{}.Column().B2TxHash:         deposit.B2TxHash,
		model.Deposit{}.Column().BtcFromAAAddress: deposit.BtcFromAAAddress,
		model.Deposit{}.Column().B2TxStatus:       deposit.B2TxStatus,
		model.Deposit{}.Column().B2TxRetry:        deposit.B2TxRetry,
		model.Deposit{}.Column().B2EoaTxHash:      deposit.B2EoaTxHash,
		model.Deposit{}.Column().B2EoaTxStatus:    deposit.B2EoaTxStatus,
	}
	err = bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(updateFields).Error
	if err != nil {
		return err
	}
	bis.log.Infow("handle deposit success", "btcTxHash", deposit.BtcTxHash, "deposit", deposit)
	return nil
}

func (bis *BridgeDepositService) HandleDeadlineDeposit(deposit model.Deposit) error {
	tx, isPending, err := bis.bridge.TransactionByHash(deposit.B2TxHash)
	if err != nil {
		return err
	}

	if isPending {
		return bis.HandleDeposit(deposit, tx)
	}

	return nil
}
