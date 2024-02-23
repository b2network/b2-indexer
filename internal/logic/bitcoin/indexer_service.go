package bitcoin

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/internal/types"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/tendermint/tendermint/libs/service"
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
			bis.log.Infow("start parse block", "currentBlock", i, "currentTxIndex", currentTxIndex)
			txResults, blockHeader, err := bis.txIdxr.ParseBlock(i, currentTxIndex)
			if err != nil {
				if errors.Is(err, ErrTargetConfirmations) {
					bis.log.Warnw("parse block confirmations", "error", err.Error(), "currentBlock", i, "currentTxIndex", currentTxIndex)
					time.Sleep(NewBlockWaitTimeout)
				} else {
					bis.log.Errorw("parse block unknown err", "error", err.Error(), "currentBlock", i, "currentTxIndex", currentTxIndex)
				}
				if currentTxIndex == 0 {
					currentBlock = i - 1
				} else {
					currentBlock = i
					currentTxIndex--
				}

				break
			}
			if len(txResults) > 0 {
				currentBlock, currentTxIndex, err = bis.HandleResults(txResults, btcIndex, blockHeader.Timestamp, i)
				if err != nil {
					bis.log.Errorw("failed to handle results", "error", err,
						"currentBlock", currentBlock, "currentTxIndex", currentTxIndex, "latestBlock", latestBlock)
					rollback := true
					// not duplicated key, rollback index
					if pgErr, ok := err.(*pgconn.PgError); ok {
						// 23505 duplicate key value violates unique constraint , continue
						if pgErr.Code == "23505" {
							rollback = false
						}
					}

					if rollback {
						if currentTxIndex == 0 {
							currentBlock = i - 1
						} else {
							currentBlock = i
							currentTxIndex--
						}
						break
					}
				}
			}
			currentBlock = i
			currentTxIndex = 0
			btcIndex.BtcIndexBlock = currentBlock
			btcIndex.BtcIndexTx = currentTxIndex
			if err := bis.db.Save(&btcIndex).Error; err != nil {
				bis.log.Errorw("failed to save bitcoin index block", "error", err, "currentBlock", i,
					"currentTxIndex", currentTxIndex, "latestBlock", latestBlock)
				// rollback
				currentBlock = i - 1
				break
			}
			bis.log.Infow("bitcoin indexer parsed", "currentBlock", i,
				"currentTxIndex", currentTxIndex, "latestBlock", latestBlock)
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
			BtcFrom:        parseResult.From[0].Address,
			BtcFromPubKey:  parseResult.From[0].PubKey,
			BtcTo:          parseResult.To,
			BtcValue:       parseResult.Value,
			BtcFroms:       string(froms),
			B2TxStatus:     b2TxStatus,
			BtcBlockTime:   btcBlockTime,
			B2TxRetry:      0,
			B2NodeTxStatus: model.DepositB2NodeTxStatusPending,
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

func (bis *IndexerService) HandleResults(
	txResults []*types.BitcoinTxParseResult,
	btcIndex model.BtcIndex,
	btcBlockTime time.Time,
	currentBlock int64,
) (int64, int64, error) {
	for _, v := range txResults {
		// if from is listen address, skip
		if bis.ToInFroms(v.From, v.To) {
			bis.log.Infow("current transaction from is listen address", "currentBlock", currentBlock, "currentTxIndex", v.Index, "data", v)
			continue
		}

		btcIndex.BtcIndexBlock = currentBlock
		btcIndex.BtcIndexTx = v.Index
		// write db
		err := bis.SaveParsedResult(
			v,
			currentBlock,
			model.DepositB2TxStatusPending,
			btcBlockTime,
			btcIndex,
		)
		if err != nil {
			bis.log.Errorw("failed to save bitcoin index tx", "error", err,
				"data", v)
			return currentBlock, v.Index, err
		}
		bis.log.Infow("save bitcoin index tx success", "currentBlock", currentBlock, "currentTxIndex", v.Index, "data", v)
		time.Sleep(IndexTxTimeout)
	}
	return currentBlock, 0, nil
}

func (bis *IndexerService) ToInFroms(a []types.BitcoinFrom, s string) bool {
	for _, i := range a {
		if i.Address == s {
			return true
		}
	}
	return false
}
