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
	B2NodeUpdateDepositServiceName = "B2NodeUpdateDepositService"
	B2NodeUpdateDepositWaitTimeout = 1 * time.Minute
	BatchDepositLimit              = 100
)

// UpdateDepositService update b2node deposit record
type UpdateDepositService struct {
	service.BaseService
	b2node types.B2NODEBridge
	db     *gorm.DB
	log    log.Logger
}

// NewListenDepositB2NodeService returns a new service instance.
func NewUpdateDepositService(
	b2node types.B2NODEBridge,
	db *gorm.DB,
	logger log.Logger,
) *UpdateDepositService {
	is := &UpdateDepositService{b2node: b2node, db: db, log: logger}
	is.BaseService = *service.NewBaseService(nil, B2NodeUpdateDepositServiceName, is)
	return is
}

func (bis *UpdateDepositService) OnStart() error {
	ticker := time.NewTicker(B2NodeUpdateDepositWaitTimeout)
	for {
		<-ticker.C
		ticker.Reset(B2NodeUpdateDepositWaitTimeout)

		var confirmDeposits []model.Deposit
		err := bis.db.
			Where(
				fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxStatus),
				model.DepositB2TxStatusSuccess,
			).Where(
			fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().B2NodeTxStatus),
			model.DepositB2NodeTxStatusRollupPending,
		).
			Limit(BatchDepositLimit).
			Find(&confirmDeposits).Error
		if err != nil {
			bis.log.Errorw("failed find tx from db", "error", err)
		}
		for _, deposit := range confirmDeposits {
			err := bis.UpdateDeposit(deposit)
			if err != nil {
				bis.log.Errorw("failed update deposit", "error", err)
			}
		}
	}
}

func (bis *UpdateDepositService) UpdateDeposit(deposit model.Deposit) error {
	b2NodeCheck := true
	// check b2node deposit
	b2NodeDeposit, err := bis.b2node.QueryDeposit(deposit.BtcTxHash)
	if err != nil {
		bis.log.Errorw("failed query b2node deposit",
			"error", err,
			"deposit", deposit,
			"b2node deposit", b2NodeDeposit,
		)
		b2NodeCheck = false
	}
	if b2NodeDeposit.Status != bridgeTypes.DepositStatus_DEPOSIT_STATUS_PENDING {
		bis.log.Errorw("b2node deposit status mismatch",
			"status", b2NodeDeposit.Status,
			"deposit", deposit,
			"b2node deposit", b2NodeDeposit,
		)
		b2NodeCheck = false
	}
	// check params
	if b2NodeDeposit.GetFrom() != deposit.BtcFrom ||
		b2NodeDeposit.GetTo() != deposit.BtcTo ||
		b2NodeDeposit.GetValue() != deposit.BtcValue {
		bis.log.Errorw("b2node deposit value mismatch",
			"status", b2NodeDeposit.Status,
			"deposit", deposit,
			"b2node deposit", b2NodeDeposit,
		)
		b2NodeCheck = false
	}
	if b2NodeCheck {
		// rollup succss, update b2node deposit
		err = bis.b2node.UpdateDeposit(deposit.BtcTxHash,
			bridgeTypes.DepositStatus_DEPOSIT_STATUS_COMPLETED,
			deposit.B2TxHash,
			deposit.BtcFromAAAddress,
		)
		if err != nil {
			bis.log.Errorw("b2node update deposit err",
				"error", err.Error(),
				"data", deposit,
			)
			return err
		}
		deposit.B2NodeTxStatus = model.DepositB2NodeTxStatusSuccess

		updateFields := map[string]interface{}{
			model.Deposit{}.Column().B2NodeTxStatus: deposit.B2NodeTxStatus,
		}
		err = bis.db.Model(&model.Deposit{}).
			Where("id = ?", deposit.ID).
			Where(
				fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().B2TxStatus),
				model.DepositB2TxStatusSuccess,
			).
			Updates(updateFields).Error
		if err != nil {
			return err
		}
	}

	return nil
}
