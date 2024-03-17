package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	pb "github.com/b2network/b2-indexer/api/protobuf"
	"github.com/b2network/b2-indexer/api/protobuf/vo"
	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/pkg/log"
	sinohopeType "github.com/b2network/b2-indexer/pkg/sinohope/types"
	"gorm.io/gorm"
)

type notifyServer struct {
	pb.UnimplementedNotifyServiceServer
}

func newNotifyServer() *notifyServer {
	return &notifyServer{}
}

func (s *notifyServer) TransactionNotify(ctx context.Context, req *vo.TransactionNotifyRequest) (*vo.TransactionNotifyResponse, error) {
	logger := log.WithName("TransactionNotify")
	logger.Infow("request data:", "req", req)
	db, err := GetDBContext(ctx)
	if err != nil {
		return nil, err
	}

	btcTx := sinohopeType.DepositDetail{}
	err = json.Unmarshal([]byte(req.RequestDetail), &btcTx)
	if err != nil {
		return nil, err
	}

	var deposit model.Deposit
	err = db.
		Where(
			fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().BtcTxHash),
			btcTx.TxHash,
		).
		First(&deposit).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			amount, err := strconv.Atoi(btcTx.Amount)
			if err != nil {
				return nil, err
			}
			deposit := model.Deposit{
				// BtcBlockNumber: btcTx.BlockHash,
				// BtcTxIndex:     parseResult.Index,
				BtcTxHash:  btcTx.TxHash,
				BtcFrom:    btcTx.From,
				BtcTos:     string("{}"),
				BtcTo:      btcTx.To,
				BtcValue:   int64(amount),
				BtcFroms:   string("{}"),
				B2TxStatus: model.DepositB2TxStatusPending,
				// BtcBlockTime:  btcBlockTime,
				CallbackStatus: model.CallbackStatusSuccess,
			}
			err = db.Save(&deposit).Error
			if err != nil {
				logger.Errorw("failed to save tx result", "error", err)
				return nil, err
			}
		} else {
			logger.Errorw("failed find tx from db", "error", err)
			return nil, err
		}
	} else {
		updateFields := map[string]interface{}{
			model.Deposit{}.Column().CallbackStatus: model.CallbackStatusSuccess,
		}
		err = db.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(updateFields).Error
		if err != nil {
			return nil, err
		}
	}

	return &vo.TransactionNotifyResponse{
		RequestId: req.RequestId,
		Code:      200,
	}, nil
}
