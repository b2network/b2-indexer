package bitcoin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/internal/types"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/cometbft/cometbft/libs/service"
	"gorm.io/gorm"
)

const (
	BridgeDepositServiceName = "BitcoinBridgeDepositService"
	BatchDepositWaitTimeout  = 10 * time.Second
	DepositErrTimeout        = 10 * time.Minute
	BatchDepositLimit        = 100
	WaitMinedTimeout         = 2 * time.Hour
	HandleDepositTimeout     = 1 * time.Second
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
	ticker := time.NewTicker(BatchDepositWaitTimeout)
	for {
		<-ticker.C
		ticker.Reset(BatchDepositWaitTimeout)
		// Query condition
		// 1. tx status is pending
		// 2. contract insufficient balance
		// 3. invoke contract from account insufficient balance
		// 4. b2-node create deposit success
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
			err = bis.HandleDeposit(deposit)
			if err != nil {
				bis.log.Errorw("handle deposit failed", "error", err, "deposit", deposit)
			}

			time.Sleep(HandleDepositTimeout)
		}

		// tx wait confirm
		// The chain is stuck for unknown reasons, and the transaction status is stuck when the reply is processed
		var unconfirmDeposits []model.Deposit
		err = bis.db.
			Where(
				fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxStatus),
				model.DepositB2TxStatusWaitMined,
			).
			Limit(BatchDepositLimit).
			Find(&unconfirmDeposits).Error
		if err != nil {
			bis.log.Errorw("failed find tx from db", "error", err)
		}
		bis.log.Infow("start handle unconfirm deposit", "unconfirm deposit batch num", len(unconfirmDeposits))
		for _, deposit := range unconfirmDeposits {
			err := bis.HandleUnconfirmDeposit(deposit)
			if err != nil {
				bis.log.Errorw("handle unconfirm deposit failed", "error", err, "unconfirm deposits", deposit)
			}

			time.Sleep(HandleDepositTimeout)
		}
	}
}

func (bis *BridgeDepositService) HandleDeposit(deposit model.Deposit) error {
	defer func() {
		if err := recover(); err != nil {
			bis.log.Errorw("panic err", err)
		}
	}()
	// set init status
	deposit.B2EoaTxStatus = model.DepositB2EoaTxStatusPending

	// send deposit tx
	b2Tx, _, aaAddress, err := bis.bridge.Deposit(deposit.BtcTxHash, types.BitcoinFrom{
		Address: deposit.BtcFrom,
	}, deposit.BtcValue)
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
	}

	updateFields := map[string]interface{}{
		model.Deposit{}.Column().B2TxHash:         deposit.B2TxHash,
		model.Deposit{}.Column().BtcFromAAAddress: deposit.BtcFromAAAddress,
		model.Deposit{}.Column().B2TxStatus:       deposit.B2TxStatus,
		model.Deposit{}.Column().B2TxRetry:        deposit.B2TxRetry,
	}
	err = bis.db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(updateFields).Error
	if err != nil {
		return err
	}
	bis.log.Infow("handle deposit success", "btcTxHash", deposit.BtcTxHash, "deposit", deposit)
	return nil
}

func (bis *BridgeDepositService) HandleUnconfirmDeposit(deposit model.Deposit) error {
	defer func() {
		if err := recover(); err != nil {
			bis.log.Errorw("panic err", err)
		}
	}()
	// find tx
	b2txReceipt, err := bis.bridge.TransactionReceipt(deposit.B2TxHash)
	if err != nil {
		bis.log.Errorw("handle unconfirm deposit get b2tx receipt err",
			"error", err.Error(),
			"data", deposit)
		return err
	}
	deposit.B2TxStatus = model.DepositB2TxStatusSuccess
	if b2txReceipt.Status != 1 {
		deposit.B2TxStatus = model.DepositB2TxStatusWaitMinedStatusFailed
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

	updateFields := map[string]interface{}{
		model.Deposit{}.Column().B2TxStatus:    deposit.B2TxStatus,
		model.Deposit{}.Column().B2EoaTxHash:   deposit.B2EoaTxHash,
		model.Deposit{}.Column().B2EoaTxStatus: deposit.B2EoaTxStatus,
	}
	err = bis.db.Model(&model.Deposit{}).
		Where("id = ?", deposit.ID).
		Where(
			fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxStatus),
			model.DepositB2TxStatusWaitMined,
		).
		Updates(updateFields).Error
	if err != nil {
		return err
	}
	bis.log.Infow("handle unconfirm deposit success", "btcTxHash", deposit.BtcTxHash, "deposit", deposit)
	return nil
}
