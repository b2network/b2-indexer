package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	pb "github.com/b2network/b2-indexer/api/protobuf"
	"github.com/b2network/b2-indexer/api/protobuf/vo"
	"github.com/b2network/b2-indexer/internal/app/exceptions"
	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/pkg/log"
	sinohopeType "github.com/b2network/b2-indexer/pkg/sinohope/types"
	"github.com/b2network/b2-indexer/pkg/utils"
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
	listenAddress, err := GetListenAddress(ctx)
	if err != nil {
		logger.Errorf("GetListenAddress err:%v", err.Error())
		return ErrorTransactionNotify(exceptions.SystemError, "system error"), nil
	}
	logger.Infof("listen address config:%v", listenAddress)
	httpCfg, err := GetHTTPConfig(ctx)
	if err != nil {
		logger.Errorf("GetHttpConfig err:%v", err.Error())
		return ErrorTransactionNotify(exceptions.SystemError, "system error"), nil
	}
	// check white list
	isWhiteList := false
	whiteList := httpCfg.IPWhiteList
	whiteListIP := strings.Split(whiteList, ",")
	clientIP := utils.ClientIP(ctx, logger)
	for _, ip := range whiteListIP {
		if clientIP == strings.TrimSpace(ip) {
			isWhiteList = true
		}
	}
	logger.Infof("ip:%v  white list:%v", clientIP, whiteList)
	if !isWhiteList {
		logger.Errorf("ip:%v not in white list", clientIP)
		return ErrorTransactionNotify(exceptions.IPWhiteList, "ip limit"), nil
	}

	if req.RequestType != sinohopeType.RequestTypeRecharge {
		return ErrorTransactionNotify(exceptions.RequestTypeNonsupport, "request type nonsupport"), nil
	}
	detail, err := req.RequestDetail.MarshalJSON()
	if err != nil {
		return ErrorTransactionNotify(exceptions.SystemError, "system error"), nil
	}
	logger.Infof("request detail: %s", string(detail))
	requestDetail := sinohopeType.RequestDetail{}
	err = json.Unmarshal(detail, &requestDetail)
	if err != nil {
		logger.Errorf("request detail unmarshal err:%v", err.Error())
		return ErrorTransactionNotify(exceptions.RequestDetailUnmarshal, "request detail unmarshal err"), nil
	}
	if requestDetail.From == "" || requestDetail.To == "" || requestDetail.TxHash == "" {
		logger.Errorf("request detail empty")
		return ErrorTransactionNotify(exceptions.RequestDetailParameter, "request detail check err"), nil
	}
	if requestDetail.To != listenAddress {
		logger.Errorf("request detail to address not eq listen address")
		return ErrorTransactionNotify(exceptions.RequestDetailToMismatch, "request detail to mismatch"), nil
	}
	amount, err := strconv.ParseInt(requestDetail.Amount, 10, 64)
	if err != nil {
		return ErrorTransactionNotify(exceptions.RequestDetailAmount, "request detail amount "), nil
	}
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
				requestDetail.TxHash,
			).
			First(&deposit).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				deposit := model.Deposit{
					BtcTxHash:      requestDetail.TxHash,
					BtcFrom:        requestDetail.From,
					BtcTos:         string("{}"),
					BtcTo:          requestDetail.To,
					BtcValue:       amount,
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
			// update check fields
			if !strings.EqualFold(deposit.BtcFrom, requestDetail.From) {
				return errors.New("from address not match")
			}
			if !strings.EqualFold(deposit.BtcTo, requestDetail.To) {
				return errors.New("to address not match")
			}
			if deposit.BtcValue != amount {
				return errors.New("amount not match")
			}
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
		return ErrorTransactionNotify(exceptions.SystemError, "system error"), nil
	}
	return &vo.TransactionNotifyResponse{
		RequestId: req.RequestId,
		Code:      200,
	}, nil
}
