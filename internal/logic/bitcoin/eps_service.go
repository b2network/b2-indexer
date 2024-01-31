package bitcoin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/b2network/b2-indexer/internal/config"
	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/go-resty/resty/v2"
	"gorm.io/gorm"
)

// EpsService eps service
type EpsService struct {
	EthRPCURL string
	config    config.EpsConfig
	log       log.Logger
	db        *gorm.DB
}

type DepositData struct {
	Caller      string
	ToAddress   string
	Amount      string
	Timestamp   string
	BlockNumber int64
	LogIndex    int
	TxHash      string
}

type EpsResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

type EthRequest struct {
	Jsonrpc string   `json:"jsonrpc"`
	Method  string   `json:"method"`
	Params  []string `json:"params"`
	ID      int      `json:"id"`
}

type EthResponse struct {
	Jsonrpc string                 `json:"jsonrpc"`
	ID      int                    `json:"id"`
	Result  EthTransactionResponse `json:"result"`
}

type EthTransactionResponse struct {
	BlockNumber      string `json:"blockNumber"`
	From             string `json:"from"`
	To               string `json:"to"`
	TransactionIndex string `json:"transactionIndex"`
	Value            string `json:"value"`
}

// NewEpsService new eps
func NewEpsService(
	bridgeCfg config.BridgeConfig,
	config config.EpsConfig,
	log log.Logger,
	db *gorm.DB,
) (*EpsService, error) {
	rpcURL, err := url.ParseRequestURI(bridgeCfg.EthRPCURL)
	if err != nil {
		return nil, err
	}
	return &EpsService{
		EthRPCURL: rpcURL.String(),
		config:    config,
		log:       log,
		db:        db,
	}, nil
}

func (e *EpsService) OnStart() error {
	if !e.db.Migrator().HasTable(&model.Eps{}) {
		err := e.db.AutoMigrate(&model.Eps{})
		if err != nil {
			e.log.Errorw("eps create table", "error", err.Error())
			return err
		}
	}
	go func() {
		for {
			time.Sleep(time.Minute)
			var depositList []model.Deposit
			err := e.db.Model(&model.Deposit{}).Where("%s = ?", model.Deposit{}.Column().B2EoaTxStatus, model.DepositB2EoaTxStatusSuccess).Find(&depositList).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				e.log.Errorw("eps get list err", "error", err)
				continue
			}
			for _, v := range depositList {
				var eps model.Eps
				err := e.db.Model(&model.Eps{}).Where("%s = ?", model.Eps{}.Column().DepositID, v.ID).First(&eps).Error
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// insert into eps table
						txData, err := e.GetTransactionByHash(v.B2EoaTxHash)
						if err != nil {
							e.log.Errorw("eps GetTransactionByHash err", "error", err)
							continue
						}
						blockNumber, err := strconv.ParseInt(txData.BlockNumber, 0, 0)
						if err != nil {
							e.log.Errorw("eps strconv blockNumber err", "error", err)
							continue
						}
						transactionIndex, err := strconv.ParseInt(txData.TransactionIndex, 0, 0)
						if err != nil {
							e.log.Errorw("eps strconv blockNumber err", "error", err)
							continue
						}
						esp := model.Eps{
							DepositID:          v.ID,
							B2From:             txData.From,
							B2To:               txData.To,
							BtcValue:           v.BtcValue,
							B2TxHash:           v.B2TxHash,
							B2TxTime:           v.UpdatedAt,
							B2BlockNumber:      blockNumber,
							B2TransactionIndex: transactionIndex,
						}
						err = e.db.Model(&model.Eps{}).Save(&esp).Error
						if err != nil {
							e.log.Errorw("insert into eps table err", "error", err)
							continue
						}
					} else {
						e.log.Errorw("eps get by deposit_id err", "error", err, "deposit_id", v.ID)
					}
				}
			}
		}
	}()

	for {
		time.Sleep(time.Minute)
		var epsList []model.Eps
		result := e.db.Model(&model.Eps{}).Where(fmt.Sprintf("%s = ?", model.Eps{}.Column().Status), model.EspStatus).Find(&epsList)
		if result.Error != nil {
			e.log.Errorw("eps get list err", "error", result.Error)
			continue
		}
		if result.RowsAffected <= 0 {
			continue
		}
		for _, v := range epsList {
			depositData := DepositData{
				Caller:      v.B2From,
				ToAddress:   v.B2To,
				Amount:      strconv.FormatInt(v.BtcValue, 10),
				Timestamp:   v.B2TxTime.Format(""),
				BlockNumber: v.B2BlockNumber,
				LogIndex:    0,
				TxHash:      v.B2TxHash,
			}
			err := e.Deposit(depositData)
			if err != nil {
				e.log.Errorw("eps deposit err", "error", err)
				continue
			}
			e.log.Infow("post bridge deposit success", "depositData", depositData)
			err = e.db.Model(&model.Eps{}).Where(fmt.Sprintf("%s = ?", model.Eps{}.Column().ID), v.ID).Update(model.Eps{}.Column().Status, model.EspStatusSuccess).Error
			if err != nil {
				e.log.Errorw("eps update eps table err", "error", err, "id", v.ID)
				continue
			}
			e.log.Infow("update deposit success", "id", v.ID, "tx_hash", v.B2TxHash)
		}
	}
}

func (e *EpsService) Deposit(data DepositData) error {
	body, err := json.Marshal(data)
	if err != nil {
		e.log.Errorw("eps deposit marshal err", "error", err)
		return err
	}
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", e.config.Authorization).
		SetBody(string(body)).
		Put(fmt.Sprintf("%s/bridge/deposit", e.config.URL))
	if err != nil {
		e.log.Errorw("eps deposit client url err", "error", err)
		return err
	}
	if resp.StatusCode() != 200 {
		return errors.New(resp.Status())
	}
	var respData EpsResponse
	err = json.Unmarshal(resp.Body(), &respData)
	if err != nil {
		e.log.Errorw("eps deposit unmarshal err", "error", err)
		return err
	}
	if respData.Code != 0 {
		e.log.Errorw("eps deposit failed", "resp", respData)
		return errors.New(respData.Msg)
	}
	return nil
}

func (e *EpsService) GetTransactionByHash(txHash string) (*EthTransactionResponse, error) {
	reqData := EthRequest{
		Jsonrpc: "2.0",
		Method:  "eth_getTransactionByHash",
		Params:  []string{txHash},
		ID:      1,
	}
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(reqData).
		Post(e.EthRPCURL)
	if err != nil {
		e.log.Errorw("eps post GetTransactionByHash  err", "error", err)
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New(resp.Status())
	}
	var respData EthResponse
	err = json.Unmarshal(resp.Body(), &respData)
	if err != nil {
		e.log.Errorw("eps GetTransactionByHash unmarshal err", "error", err)
		return nil, err
	}
	return &respData.Result, nil
}
