package bitcoin

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/internal/types"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/b2network/b2-indexer/pkg/utils"
	"github.com/cometbft/cometbft/libs/service"
	"gorm.io/gorm"
)

const (
	ServiceName = "BitcoinIndexerService"

	NewBlockWaitTimeout = 60 * time.Second

	IndexTxTimeout    = 100 * time.Millisecond
	IndexBlockTimeout = 2 * time.Second
)

// IndexerService indexes transactions for json-rpc service.
type IndexerService struct {
	service.BaseService

	txIdxr types.BITCOINTxIndexer

	db  *gorm.DB
	log log.Logger
}

// NewIndexerService returns a new service instance.
func NewIndexerService(
	txIdxr types.BITCOINTxIndexer,
	// bridge types.BITCOINBridge,
	db *gorm.DB,
	logger log.Logger,
) *IndexerService {
	is := &IndexerService{txIdxr: txIdxr, db: db, log: logger}
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
				bis.log.Errorw("parseblock", "error", err.Error(), "currentBlock", i, "currentTxIndex", currentTxIndex)
				continue
			}
			if len(txResults) > 0 {
				for _, v := range txResults {
					// if from is listen address, skip
					if utils.StrInArray(v.From, v.To) {
						bis.log.Infow("current transaction from is listen address", "currentBlock", i, "currentTxIndex", v.Index, "data", v)
						continue
					}

					btcIndex.BtcIndexBlock = i
					btcIndex.BtcIndexTx = v.Index
					// write db
					err = bis.SaveParsedResult(
						v,
						i,
						model.DepositB2TxStatusPending,
						blockHeader.Timestamp,
						btcIndex,
					)
					if err != nil {
						bis.log.Errorw("failed to save bitcoin index tx", "error", err,
							"data", v)
					} else {
						bis.log.Infow("bitcoin indexer save bitcoin index tx success", "data", v)
					}

					time.Sleep(IndexTxTimeout)
				}
			}
			currentBlock = i
			currentTxIndex = 0
			btcIndex.BtcIndexBlock = currentBlock
			btcIndex.BtcIndexTx = currentTxIndex
			if err := bis.db.Save(&btcIndex).Error; err != nil {
				bis.log.Errorw("failed to save bitcoin index block", "error", err, "currentBlock", i,
					"currentTxIndex", currentTxIndex, "latestBlock", latestBlock)
			} else {
				bis.log.Infow("bitcoin indexer parsed", "currentBlock", i,
					"currentTxIndex", currentTxIndex, "latestBlock", latestBlock)
			}

			time.Sleep(IndexBlockTimeout)
		}
	}
}

// save index tx to db
func (bis *IndexerService) SaveParsedResult(
	parseResult *types.BitcoinTxParseResult,
	btcBlockNumber int64,
	b2TxStatus int,
	btcBlockTime time.Time,
	btcIndex model.BtcIndex,
) error {
	// write db
	err := bis.db.Transaction(func(tx *gorm.DB) error {
		froms, err := json.Marshal(parseResult.From)
		if err != nil {
			return err
		}
		deposit := model.Deposit{
			BtcBlockNumber: btcBlockNumber,
			BtcTxIndex:     parseResult.Index,
			BtcTxHash:      parseResult.TxID,
			BtcFrom:        parseResult.From[0],
			BtcTo:          parseResult.To,
			BtcValue:       parseResult.Value,
			BtcFroms:       string(froms),
			B2TxStatus:     b2TxStatus,
			BtcBlockTime:   btcBlockTime,
		}
		err = tx.Save(&deposit).Error
		if err != nil {
			bis.log.Errorw("failed to save tx parsed result", "error", err)
			return err
		}

		if err := tx.Save(&btcIndex).Error; err != nil {
			bis.log.Errorw("failed to save bitcoin tx index", "error", err)
			return err
		}

		return nil
	})
	return err
}
