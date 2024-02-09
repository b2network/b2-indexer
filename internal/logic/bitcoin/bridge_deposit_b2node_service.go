package bitcoin

import (
	"fmt"
	"time"

	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/internal/types"
	"github.com/b2network/b2-indexer/pkg/log"
	bridgeTypes "github.com/evmos/ethermint/x/bridge/types"
	"github.com/tendermint/tendermint/libs/service"
	"gorm.io/gorm"
)

const (
	BridgeDepositB2NodeServiceName = "BitcoinBridgeDepositB2NodeService"
	BatchDepositB2NodeWaitTimeout  = 10 * time.Second
	BatchDepositB2NodeLimit        = 100
	B2NodeWaitMinedTimeout         = 2 * time.Hour
	HandleDepositB2NodeTimeout     = 500 * time.Millisecond
	DepositB2NodeRetry             = 10
	DepositB2NodeErrTimeout        = 10 * time.Minute
)

// BridgeDepositB2NodeService l1->b2-node
type BridgeDepositB2NodeService struct {
	service.BaseService

	b2node types.B2NODEBridge

	db  *gorm.DB
	log log.Logger
}

// NewBridgeDepositB2NodeService returns a new service instance.
func NewBridgeDepositB2NodeService(
	b2node types.B2NODEBridge,
	db *gorm.DB,
	logger log.Logger,
) *BridgeDepositB2NodeService {
	is := &BridgeDepositB2NodeService{b2node: b2node, db: db, log: logger}
	is.BaseService = *service.NewBaseService(nil, BridgeDepositB2NodeServiceName, is)
	return is
}

// OnStart
func (bis *BridgeDepositB2NodeService) OnStart() error {
	ticker := time.NewTicker(BatchDepositB2NodeWaitTimeout)
	for {
		<-ticker.C
		ticker.Reset(BatchDepositB2NodeWaitTimeout)
		var deposits []model.Deposit
		// find all b2node pending txs
		err := bis.db.
			Where(
				fmt.Sprintf("%s.%s IN (?)", model.Deposit{}.TableName(), model.Deposit{}.Column().B2NodeTxStatus),
				[]int{
					model.DepositB2NodeTxStatusPending,
				},
			).
			Limit(BatchDepositB2NodeLimit).
			Find(&deposits).Error
		if err != nil {
			bis.log.Errorw("failed find tx from db", "error", err)
		}

		bis.log.Infow("start handle b2node create deposit", "deposit batch num", len(deposits))
		for _, deposit := range deposits {
			err := bis.HandleDeposit(deposit)
			if err != nil {
				bis.log.Errorw("handle deposit failed", "error", err, "deposit", deposit)
			}
			time.Sleep(HandleDepositB2NodeTimeout)
		}
	}
}

func (bis *BridgeDepositB2NodeService) HandleDeposit(deposit model.Deposit) error {
	defer func() {
		if err := recover(); err != nil {
			bis.log.Errorw("panic err", err)
		}
	}()
	// create deposit record
	err := bis.b2node.CreateDeposit(deposit.BtcTxHash, deposit.BtcFrom, deposit.BtcTo, deposit.BtcValue)
	if err != nil {
		bis.log.Errorw("b2node create deposit err", "error", err.Error(), "deposit", deposit)
		switch err {
		case bridgeTypes.ErrIndexExist:
			deposit.B2NodeTxStatus = model.DepositB2NodeTxStatusTxHashExist
		default:
			deposit.B2NodeTxRetry++
			if deposit.B2NodeTxRetry >= DepositB2NodeRetry {
				deposit.B2NodeTxStatus = model.DepositB2TxStatusFailed
				bis.log.Errorw("invoke b2node deposit send tx retry exceed max",
					"error", err.Error(),
					"retryMax", DepositB2NodeRetry,
					"data", deposit)
			} else {
				deposit.B2NodeTxStatus = model.DepositB2NodeTxStatusPending
				bis.log.Errorw("invoke b2node deposit send tx retry",
					"error", err.Error(),
					"data", deposit)
			}
			time.Sleep(DepositB2NodeErrTimeout)
		}
	} else {
		deposit.B2NodeTxStatus = model.DepositB2NodeTxStatusRollupPending
	}
	// update db status
	updateFields := map[string]interface{}{
		model.Deposit{}.Column().B2NodeTxStatus: deposit.B2NodeTxStatus,
	}
	err = bis.db.Model(&model.Deposit{}).
		Where("id = ?", deposit.ID).
		Where(fmt.Sprintf("%s = %d", model.Deposit{}.Column().B2NodeTxStatus, model.DepositB2NodeTxStatusPending)).
		Updates(updateFields).Error
	if err != nil {
		return err
	}
	return nil
}
