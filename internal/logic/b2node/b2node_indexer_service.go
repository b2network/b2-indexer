package b2node

import (
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
	B2NodeIndexerServiceName  = "B2NodeIndexerService"
	B2NodeNewBlockWaitTimeout = 1 * time.Second
	B2NodeIndexBlockTimeout   = 1 * time.Second
	B2NodeIndexTxTimeout      = 100 * time.Millisecond
)

// B2NodeIndexerService sync b2node tx event
type B2NodeIndexerService struct { //nolint
	service.BaseService
	b2node types.B2NODEBridge
	db     *gorm.DB
	log    log.Logger
}

// NewListenDepositB2NodeService returns a new service instance.
func NewB2NodeIndexerService(
	b2node types.B2NODEBridge,
	db *gorm.DB,
	logger log.Logger,
) *B2NodeIndexerService {
	is := &B2NodeIndexerService{b2node: b2node, db: db, log: logger}
	is.BaseService = *service.NewBaseService(nil, B2NodeIndexerServiceName, is)
	return is
}

func (bis *B2NodeIndexerService) OnStart() error {
	latestBlock, err := bis.b2node.LatestBlock()
	if err != nil {
		bis.log.Errorw("b2node indexer latestBlock", "error", err.Error())
		return err
	}

	var (
		currentBlock   int64 // index current block number
		currentTxIndex int64 // index current block tx index
	)
	if !bis.db.Migrator().HasTable(&model.B2Node{}) {
		err := bis.db.AutoMigrate(&model.B2Node{})
		if err != nil {
			bis.log.Errorw("b2node indexer create table", "error", err.Error())
			return err
		}
	}

	if !bis.db.Migrator().HasTable(&model.B2NodeIndex{}) {
		err := bis.db.AutoMigrate(&model.B2NodeIndex{})
		if err != nil {
			bis.log.Errorw("b2node indexer create table", "error", err.Error())
			return err
		}
	}

	var b2nodeIndex model.B2NodeIndex
	if err := bis.db.First(&b2nodeIndex, 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			b2nodeIndex = model.B2NodeIndex{
				Base: model.Base{
					ID: 1,
				},
				IndexBlock: latestBlock,
				IndexTx:    0,
			}
			if err := bis.db.Create(&b2nodeIndex).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	bis.log.Infow("b2node indexer load db", "data", b2nodeIndex)

	// set default value
	currentBlock = b2nodeIndex.IndexBlock
	currentTxIndex = b2nodeIndex.IndexTx

	ticker := time.NewTicker(B2NodeNewBlockWaitTimeout)
	for {
		bis.log.Infow("b2node indexer", "latestBlock",
			latestBlock, "currentBlock", currentBlock, "currentTxIndex", currentTxIndex)
		if latestBlock <= currentBlock {
			<-ticker.C
			ticker.Reset(B2NodeNewBlockWaitTimeout)

			// update latest block
			latestBlock, err = bis.b2node.LatestBlock()
			if err != nil {
				bis.log.Errorw("b2node indexer latestBlock", "error", err.Error())
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
			bis.log.Infow("handle", "latestBlock",
				latestBlock, "currentBlock", i, "currentTxIndex", currentTxIndex)
			txResults, err := bis.b2node.ParseBlockBridgeEvent(i, currentTxIndex)
			if err != nil {
				bis.log.Errorw("b2node indexer parse block", "error", err.Error())
				if currentTxIndex == 0 {
					currentBlock = i - 1
				} else {
					currentBlock = i
					currentTxIndex--
				}
				break
			}
			if len(txResults) > 0 {
				currentBlock, currentTxIndex, err = bis.HandleResults(txResults, b2nodeIndex, i)
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
			b2nodeIndex.IndexBlock = currentBlock
			b2nodeIndex.IndexTx = currentTxIndex
			if err := bis.db.Save(&b2nodeIndex).Error; err != nil {
				bis.log.Errorw("failed to save b2node index block", "error", err, "currentBlock", i,
					"currentTxIndex", currentTxIndex, "latestBlock", latestBlock)
				// rollback
				currentBlock = i - 1
				break
			}
			bis.log.Infow("b2node indexer parsed", "currentBlock", i,
				"currentTxIndex", currentTxIndex, "latestBlock", latestBlock)
			time.Sleep(B2NodeIndexBlockTimeout)
		}
	}
}

func (bis *B2NodeIndexerService) HandleResults(
	txResults []*types.B2NodeTxParseResult,
	b2NodeIndex model.B2NodeIndex,
	currentBlock int64,
) (int64, int64, error) {
	for _, v := range txResults {
		b2NodeIndex.IndexBlock = currentBlock
		b2NodeIndex.IndexTx = int64(v.BridgeModuleTxIndex)
		// write db
		err := bis.SaveParsedResult(
			v,
			currentBlock,
			b2NodeIndex,
		)
		if err != nil {
			bis.log.Errorw("failed to save b2node index tx", "error", err,
				"data", v)
			return currentBlock, int64(v.BridgeModuleTxIndex), err
		}
		bis.log.Infow("save b2node index tx success", "currentBlock", currentBlock, "currentTxIndex", v.BridgeModuleTxIndex, "data", v)
		time.Sleep(B2NodeIndexTxTimeout)
	}
	return currentBlock, 0, nil
}

func (bis *B2NodeIndexerService) SaveParsedResult(
	parseResult *types.B2NodeTxParseResult,
	_ int64,
	b2NodeIndex model.B2NodeIndex,
) error {
	err := bis.db.Transaction(func(tx *gorm.DB) error {
		b2Node := model.B2Node{
			Height:              parseResult.Height,
			BridgeModuleTxIndex: parseResult.BridgeModuleTxIndex,
			TxHash:              parseResult.TxHash,
			EventType:           parseResult.EventType,
			TxData:              parseResult.TxData,
			RawLog:              parseResult.RawLog,
			BridgeEventID:       parseResult.BridgeEventID,
			Messages:            parseResult.Messages,
			TxCode:              parseResult.TxCode,
		}
		err := tx.Save(&b2Node).Error
		if err != nil {
			bis.log.Errorw("failed to save b2node tx parsed result", "error", err)
			return err
		}

		if err := tx.Save(&b2NodeIndex).Error; err != nil {
			bis.log.Errorw("failed to save b2node tx index", "error", err)
			return err
		}

		return nil
	})
	return err
}
