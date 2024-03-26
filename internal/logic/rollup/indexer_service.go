package rollup

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/b2network/b2-indexer/pkg/event"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"

	"github.com/b2network/b2-indexer/internal/config"
	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/cometbft/cometbft/libs/service"
	"gorm.io/gorm"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	IndexerServiceName = "RollupIndexerService"

	WaitHandleTime = 10
)

// IndexerService indexes transactions for json-rpc service.
type IndexerService struct {
	service.BaseService

	ethCli *ethclient.Client
	config *config.BitconConfig
	db     *gorm.DB
	log    log.Logger
}

// NewIndexerService returns a new service instance.
func NewIndexerService(
	ethCli *ethclient.Client,
	config *config.BitconConfig,
	db *gorm.DB,
	log log.Logger,
) *IndexerService {
	is := &IndexerService{ethCli: ethCli, config: config, db: db, log: log}
	is.BaseService = *service.NewBaseService(nil, IndexerServiceName, is)
	return is
}

// OnStart implements service.Service by subscribing for new blocks
// and indexing them by events.
func (bis *IndexerService) OnStart() error {
	defer func() {
		if r := recover(); r != nil {
			bis.log.Errorw("BridgeWithdrawService panic", "error", r)
		}
	}()
	if !bis.db.Migrator().HasTable(&model.RollupIndex{}) {
		err := bis.db.AutoMigrate(&model.RollupIndex{})
		if err != nil {
			bis.log.Errorw("IndexerService create WithdrawIndex table", "error", err.Error())
			return err
		}
	}
	for {
		// listen server scan blocks
		time.Sleep(time.Duration(WaitHandleTime) * time.Second)
		var currentBlock uint64 // index current block number
		var currentTxIndex uint // index current block tx index
		var currentLogIndex uint
		var rollupIndex model.RollupIndex
		if err := bis.db.First(&rollupIndex, 1).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				latestBlock, err := bis.ethCli.BlockNumber(context.Background())
				if err != nil {
					bis.log.Errorw("IndexerService headerByNumber is failed:", "error", err)
					continue
				}
				rollupIndex = model.RollupIndex{
					Base: model.Base{
						ID: 1,
					},
					B2IndexBlock: latestBlock,
					B2IndexTx:    0,
					B2LogIndex:   0,
				}
				if err := bis.db.Create(&rollupIndex).Error; err != nil {
					bis.log.Errorw("IndexerService create rollupIndex is failed:", "error", err)
					continue
				}
			} else {
				bis.log.Errorw("IndexerService get first rollupIndex is failed:", "error", err)
				continue
			}
		}
		currentBlock = rollupIndex.B2IndexBlock
		currentTxIndex = rollupIndex.B2IndexTx
		currentLogIndex = rollupIndex.B2LogIndex
		addresses := []common.Address{
			common.HexToAddress(bis.config.Bridge.ContractAddress),
		}
		topics := [][]common.Hash{
			{
				common.HexToHash(bis.config.Bridge.Deposit),
				common.HexToHash(bis.config.Bridge.Withdraw),
			},
		}

		latestBlock, err := bis.ethCli.BlockNumber(context.Background())
		if err != nil {
			bis.log.Errorw("IndexerService HeaderByNumber is failed:", "error", err)
			continue
		}
		bis.log.Infow("IndexerService ethClient height", "height", latestBlock, "currentBlock", currentBlock)
		if latestBlock == currentBlock {
			continue
		}
		for i := currentBlock; i <= latestBlock; i++ {
			bis.log.Infow("IndexerService get log height:", "height", i)
			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(0).SetUint64(i),
				ToBlock:   big.NewInt(0).SetUint64(i),
				Topics:    topics,
				Addresses: addresses,
			}
			logs, err := bis.ethCli.FilterLogs(context.Background(), query)
			if err != nil {
				bis.log.Errorw("IndexerService failed to fetch block", "height", i, "error", err)
				continue
			}

			for _, vlog := range logs {
				if currentBlock == vlog.BlockNumber && currentTxIndex == vlog.TxIndex && currentLogIndex == vlog.Index {
					continue
				}
				eventHash := common.BytesToHash(vlog.Topics[0].Bytes())
				// if eventHash == common.HexToHash(bis.config.Bridge.Withdraw) {
				// 	err = handelWithdrawEvent(vlog, bis.db, bis.config.IndexerListenAddress)
				// 	if err != nil {
				// 		bis.log.Errorw("IndexerService handelWithdrawEvent err: ", "error", err)
				// 		continue
				// 	}
				// }
				if eventHash == common.HexToHash(bis.config.Bridge.Deposit) {
					bis.log.Warnw("vlog", "vlog", vlog)
					err = handelDepositEvent(vlog, bis.db)
					if err != nil {
						bis.log.Errorw("IndexerService handelDepositEvent err: ", "error", err)
						continue
					}
				}
				currentTxIndex = vlog.TxIndex
				currentLogIndex = vlog.Index
			}
			currentBlock = i
			rollupIndex.B2IndexBlock = currentBlock
			rollupIndex.B2IndexTx = currentTxIndex
			rollupIndex.B2LogIndex = currentLogIndex
			if err := bis.db.Save(&rollupIndex).Error; err != nil {
				bis.log.Errorw("failed to save b2 index block", "error", err, "currentBlock", i,
					"currentTxIndex", currentTxIndex, "latestBlock", latestBlock)
				continue
			}
		}
	}
}

// handelWithdrawEvent
//
//lint:ignore U1000 Ignore unused function temporarily for debugging
func handelWithdrawEvent(vlog ethtypes.Log, db *gorm.DB, listenAddress string) error {
	amount := DataToBigInt(vlog, 1)
	destAddrStr := DataToString(vlog, 0)
	withdrawData := model.Withdraw{
		BtcFrom:       listenAddress,
		BtcTo:         destAddrStr,
		BtcValue:      amount.Int64(),
		B2BlockNumber: vlog.BlockNumber,
		B2BlockHash:   vlog.BlockHash.String(),
		B2TxHash:      vlog.TxHash.String(),
		B2TxIndex:     vlog.TxIndex,
		B2LogIndex:    vlog.Index,
	}
	if err := db.Create(&withdrawData).Error; err != nil {
		return err
	}
	return nil
}

func handelDepositEvent(vlog ethtypes.Log, db *gorm.DB) error {
	Caller := event.TopicToAddress(vlog, 1).Hex()
	ToAddress := event.TopicToAddress(vlog, 2).Hex()
	Amount := event.DataToDecimal(vlog, 0, 0)
	TxHash := event.DataToHash(vlog, 1)

	log.Errorw("deposit event ", "Caller", Caller, "ToAddress", ToAddress, "Amount", Amount.String(), "TxHash", TxHash.String())

	depositData := model.RollupDeposit{
		BtcTxHash:        remove0xPrefix(TxHash.String()),
		BtcFromAAAddress: ToAddress,
		BtcValue:         Amount.Div(decimal.NewFromInt(10000000000)).BigInt().Int64(),
		B2TxFrom:         Caller,
		B2BlockNumber:    vlog.BlockNumber,
		B2BlockHash:      vlog.BlockHash.String(),
		B2TxHash:         vlog.TxHash.String(),
		B2TxIndex:        vlog.TxIndex,
		B2LogIndex:       vlog.Index,
	}
	if err := db.Create(&depositData).Error; err != nil {
		return err
	}
	return nil
}

func remove0xPrefix(input string) string {
	if len(input) > 2 && input[:2] == "0x" {
		return input[2:]
	}
	return input
}
