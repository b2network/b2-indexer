package bitcoin

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/internal/types"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/cometbft/cometbft/libs/service"
	"gorm.io/gorm"
)

const (
	ServiceName = "BitcoinIndexerService"

	NewBlockWaitTimeout = 60 * time.Second
)

// IndexerService indexes transactions for json-rpc service.
type IndexerService struct {
	service.BaseService

	txIdxr types.BITCOINTxIndexer
	bridge types.BITCOINBridge

	db  *gorm.DB
	log log.Logger
}

// NewIndexerService returns a new service instance.
func NewIndexerService(
	txIdxr types.BITCOINTxIndexer,
	bridge types.BITCOINBridge,
	db *gorm.DB,
	logger log.Logger,
) *IndexerService {
	is := &IndexerService{txIdxr: txIdxr, bridge: bridge, db: db, log: logger}
	is.BaseService = *service.NewBaseService(nil, ServiceName, is)
	return is
}

// OnStart
func (bis *IndexerService) OnStart() error {
	latestBlock, err := bis.txIdxr.LatestBlock()
	if err != nil {
		bis.log.Errorw("bitcoin indexer latestBlock", "error", err.Error())
		return err
	}

	var (
		currentBlock   int64 // index current block number
		currentTxIndex int64 // index current block tx index
	)
	// TODO: create db table
	if !bis.db.Migrator().HasTable(&model.Deposit{}) {
		err = bis.db.AutoMigrate(&model.Deposit{})
		if err != nil {
			bis.log.Errorw("bitcoin indexer create table", "error", err.Error())
			return err
		}
	}

	if !bis.db.Migrator().HasTable(&model.BtcIndex{}) {
		err = bis.db.AutoMigrate(&model.BtcIndex{})
		if err != nil {
			bis.log.Errorw("bitcoin indexer create table", "error", err.Error())
			return err
		}
	}

	var btcIndex model.BtcIndex
	if err := bis.db.First(&btcIndex, 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			btcIndex = model.BtcIndex{
				Base: model.Base{
					ID: 1,
				},
				BtcIndexBlock: latestBlock,
				BtcIndexTx:    0,
			}
			if err := bis.db.Create(&btcIndex).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	bis.log.Infow("bitcoin indexer load db", "data", btcIndex)

	// set default value
	currentBlock = btcIndex.BtcIndexBlock
	currentTxIndex = btcIndex.BtcIndexTx

	ticker := time.NewTicker(NewBlockWaitTimeout)
	for {
		bis.log.Infow("bitcoin indexer", "latestBlock",
			latestBlock, "currentBlock", currentBlock, "currentTxIndex", currentTxIndex)

		if latestBlock <= currentBlock {
			<-ticker.C
			ticker.Reset(NewBlockWaitTimeout)

			// update latest block
			latestBlock, err = bis.txIdxr.LatestBlock()
			if err != nil {
				bis.log.Errorw("bitcoin indexer latestBlock", "error", err.Error())
			}
			continue
		}

		// index > 0, start index from currentBlock currentTxIndex + 1
		// index == 0, start index from currentBlock + 1
		if currentTxIndex == 0 {
			currentBlock++
		} else {
			currentTxIndex++
		}

		for i := currentBlock; i <= latestBlock; i++ {
			txResults, blockHeader, err := bis.txIdxr.ParseBlock(i, currentTxIndex)
			if err != nil {
				bis.log.Errorw("bitcoin indexer parseblock", "error", err.Error(), "currentBlock", i, "currentTxIndex", currentTxIndex)
				continue
			}

			if len(txResults) > 0 {
				for _, v := range txResults {
					depositStatus := model.DepositStatusSuccess

					// if from is listen address, skip
					if v.From[0] == v.To {
						bis.log.Infow("bitcoin indexer current transaction from is listen address", "currentBlock", i, "currentTxIndex", v.Index, "data", v)
						continue
					}
					var transferResult string
					// TODO: may be wait long time
					depositResult, aaAddress, err := bis.bridge.Deposit(v.TxID, v.From[0], v.Value)
					if err != nil {
						bis.log.Errorw("bitcoin indexer invoke deposit unknown err try again by transfer", "error", err.Error(),
							"currentBlock", i, "currentTxIndex", v.Index, "data", v)
						// try transfer
						transferResult, err = bis.bridge.Transfer(v.From[0], v.Value)
						if err != nil {
							depositStatus = model.DepositStatusFailed
							bis.log.Errorw("bitcoin indexer invoke transfer unknown err", "error", err.Error(),
								"currentBlock", i, "currentTxIndex", v.Index, "data", v)
						}
					}
					btcIndex.BtcIndexBlock = i
					btcIndex.BtcIndexTx = v.Index
					// write db
					err = bis.db.Transaction(func(tx *gorm.DB) error {
						froms, err := json.Marshal(v.From)
						if err != nil {
							return err
						}
						deposit := model.Deposit{
							BtcBlockNumber: i,
							BtcTxIndex:     v.Index,
							BtcTxHash:      v.TxID,
							B2TxHash:       depositResult,
							From:           v.From[0],
							To:             v.To,
							Value:          v.Value,
							FromAAAddress:  aaAddress,
							Froms:          string(froms),
							Status:         depositStatus,
							BtcBlockTime:   blockHeader.Timestamp,
						}
						err = tx.Save(&deposit).Error
						if err != nil {
							bis.log.Errorw("failed to set deposit record", "error", err)
							return err
						}

						if err := tx.Save(&btcIndex).Error; err != nil {
							bis.log.Errorw("failed to set bitcoin index block", "error", err)
							return err
						}

						return nil
					})
					if err != nil {
						bis.log.Errorw("failed to set bitcoin index block", "error", err)
					}

					bis.log.Infow("bitcoin indexer invoke deposit", "deposit data", v, "depositResult", depositResult, "transferResult", transferResult)
				}
			}
			btcIndex.BtcIndexBlock = i
			btcIndex.BtcIndexTx = 0
			currentBlock = i
			currentTxIndex = 0

			if err := bis.db.Save(&btcIndex).Error; err != nil {
				bis.log.Errorw("failed to set bitcoin index block", "error", err)
			}

			bis.log.Infow("bitcoin indexer parsed", "currentBlock", i,
				"currentTxIndex", currentTxIndex, "latestBlock", latestBlock)
		}
	}
}
