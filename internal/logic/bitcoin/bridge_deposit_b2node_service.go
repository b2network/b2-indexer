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
	DepositB2NodeRetry             = 3
)

// BridgeDepositB2NodeService l1->b2-node
type BridgeDepositB2NodeService struct {
	service.BaseService

	// bridge types.BITCOINBridge
	b2node types.BITCOINBridgeB2Node

	db  *gorm.DB
	log log.Logger
}

// NewBridgeDepositB2NodeService returns a new service instance.
func NewBridgeDepositB2NodeService(
	// bridge types.BITCOINBridge,
	b2node types.BITCOINBridgeB2Node,
	db *gorm.DB,
	logger log.Logger,
) *BridgeDepositB2NodeService {
	is := &BridgeDepositB2NodeService{b2node: b2node, db: db, log: logger}
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
	// create deposit record
	err := bis.b2node.CreateDeposit(deposit.BtcTxHash, deposit.BtcFrom, deposit.BtcTo, uint64(deposit.BtcValue))
	if err != nil {
		bis.log.Errorw("b2node create deposit err", "error", err.Error(), "deposit", deposit)
		switch err {
		case bridgeTypes.ErrIndexExist:
			deposit.B2NodeTxStatus = model.DepositB2NodeTxStatusTxHashExist
			return nil
		default:
			deposit.B2NodeTxStatus = model.DepositB2NodeTxStatusFailed
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
