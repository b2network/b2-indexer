package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	pb "github.com/b2network/b2-indexer/api/protobuf"
	"github.com/b2network/b2-indexer/api/protobuf/vo"
	"github.com/b2network/b2-indexer/internal/app/exceptions"
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

func ErrorTransactionNotify(code int64, message string) *vo.TransactionNotifyResponse {
	return &vo.TransactionNotifyResponse{
		Code:    code,
		Message: message,
	}
}

func (s *notifyServer) TransactionNotify(ctx context.Context, req *vo.TransactionNotifyRequest) (*vo.TransactionNotifyResponse, error) {
	logger := log.WithName("TransactionNotify")
	logger.Infow("request data:", "req", req)
	db, err := GetDBContext(ctx)
	if err != nil {
		logger.Errorf("GetDBContext err:%v", err.Error())
		return ErrorTransactionNotify(exceptions.SystemError, "system error"), nil
	}

	if req.RequestType != sinohopeType.RequestTypeRecharge {
		return ErrorTransactionNotify(exceptions.RequestTypeNonsupport, "request type nonsupport"), nil
	}
	detail, err := req.RequestDetail.MarshalJSON()
	if err != nil {
		return ErrorTransactionNotify(exceptions.SystemError, "system error"), nil
	}
	logger.Infof("request detail: %s", string(detail))
	var deposit model.Deposit
	var sinohope model.Sinohope
	err = db.Transaction(func(tx *gorm.DB) error {
		err = tx.
			Where(
				fmt.Sprintf("%s.%s = ?", model.Sinohope{}.TableName(), model.Sinohope{}.Column().RequestID),
				req.RequestId,
			).
			First(&sinohope).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				sinohope = model.Sinohope{
					RequestID:     req.RequestId,
					RequestType:   int(req.RequestType),
					RequestDetail: string(detail),
				}
				err = tx.Save(&sinohope).Error
				if err != nil {
					logger.Errorw("failed to save tx result", "error", err)
					return err
				}
			} else {
				logger.Errorw("failed find tx from db", "error", err)
				return err
			}
		}

		err = tx.
			Where(
				fmt.Sprintf("%s.%s = ?", model.Deposit{}.TableName(), model.Deposit{}.Column().BtcTxHash),
				req.RequestDetail.Fields["txHash"].GetStringValue(),
			).
			First(&deposit).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				amount, err := strconv.Atoi(req.RequestDetail.Fields["amount"].GetStringValue())
				if err != nil {
					return err
				}
				deposit := model.Deposit{
					BtcTxHash:      req.RequestDetail.Fields["txHash"].GetStringValue(),
					BtcFrom:        req.RequestDetail.Fields["from"].GetStringValue(),
					BtcTos:         string("{}"),
					BtcTo:          req.RequestDetail.Fields["to"].GetStringValue(),
					BtcValue:       int64(amount),
					BtcFroms:       string("{}"),
					B2TxStatus:     model.DepositB2TxStatusPending,
					CallbackStatus: model.CallbackStatusSuccess,
					ListenerStatus: model.ListenerStatusPending,
				}
				err = tx.Create(&deposit).Error
				if err != nil {
					logger.Errorw("failed to save tx result", "error", err)
					return err
				}
			} else {
				logger.Errorw("failed find tx from db", "error", err)
				return err
			}
		} else {
			updateFields := map[string]interface{}{
				model.Deposit{}.Column().CallbackStatus: model.CallbackStatusSuccess,
			}
			err = tx.Model(&model.Deposit{}).Where("id = ?", deposit.ID).Updates(updateFields).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logger.Errorw("save tx result err", "err", err.Error())
		return nil, err
	}
	return &vo.TransactionNotifyResponse{
		RequestId: req.RequestId,
		Code:      200,
	}, nil
}
