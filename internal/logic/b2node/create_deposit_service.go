package b2node

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
	B2NodeCreateDepositServiceName = "B2NodeCreateDepositService"
	BatchDepositB2NodeWaitTimeout  = 10 * time.Second
	BatchDepositB2NodeLimit        = 100
	B2NodeWaitMinedTimeout         = 2 * time.Hour
	HandleDepositB2NodeTimeout     = 1 * time.Second
	DepositB2NodeErrTimeout        = 10 * time.Minute
)

// B2NodeCreateDepositService l1->b2-node
// B2node create deposit record
type B2NodeCreateDepositService struct { //nolint
	service.BaseService

	b2node types.B2NODEBridge

	db  *gorm.DB
	log log.Logger
}

// NewB2NodeCreateDepositService returns a new service instance.
func NewB2NodeCreateDepositService(
	b2node types.B2NODEBridge,
	db *gorm.DB,
	logger log.Logger,
) *B2NodeCreateDepositService {
	is := &B2NodeCreateDepositService{b2node: b2node, db: db, log: logger}
	is.BaseService = *service.NewBaseService(nil, B2NodeCreateDepositServiceName, is)
	return is
}

// OnStart
func (bis *B2NodeCreateDepositService) OnStart() error {
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

func (bis *B2NodeCreateDepositService) HandleDeposit(deposit model.Deposit) error {
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
			deposit.B2NodeTxStatus = model.DepositB2NodeTxStatusPending
			bis.log.Errorw("invoke b2node deposit send tx retry",
				"error", err.Error(),
				"data", deposit)
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
	bis.log.Infow("b2node create deposit success", "deposit", deposit)
	return nil
}
