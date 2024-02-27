package bitcoin

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/b2network/b2-indexer/internal/types"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

var (
	ErrParsePkScript         = errors.New("parse pkscript err")
	ErrDecodeListenAddress   = errors.New("decode listen address err")
	ErrTargetConfirmations   = errors.New("target confirmation number was not reached")
	ErrParsePubKey           = errors.New("parse pubkey failed, not found pubkey or nonsupport ")
	ErrParsePkScriptNullData = errors.New("parse pkscript null data err")
)

const (
	// tx type
	TxTypeTransfer = "transfer" // btc transfer
	TxTypeWithdraw = "withdraw" // btc withdraw
)

// Indexer bitcoin indexer, parse and forward data
type Indexer struct {
	client              *rpcclient.Client // call bitcoin rpc client
	chainParams         *chaincfg.Params  // bitcoin network params, e.g. mainnet, testnet, etc.
	listenAddress       btcutil.Address   // need listened bitcoin address
	TargetConfirmations int64
	logger              log.Logger
}

// NewBitcoinIndexer new bitcoin indexer
func NewBitcoinIndexer(
	log log.Logger,
	client *rpcclient.Client,
	chainParams *chaincfg.Params,
	listenAddress string,
	targetConfirmations int64,
) (*Indexer, error) {
	// check listenAddress
	address, err := btcutil.DecodeAddress(listenAddress, chainParams)
	if err != nil {
		return nil, fmt.Errorf("%w:%s", ErrDecodeListenAddress, err.Error())
	}
	return &Indexer{
		logger:              log,
		client:              client,
		chainParams:         chainParams,
		listenAddress:       address,
		TargetConfirmations: targetConfirmations,
	}, nil
}

// ParseBlock parse block data by block height
// NOTE: Currently, only transfer transactions are supported.
func (b *Indexer) ParseBlock(height int64, txIndex int64) ([]*types.BitcoinTxParseResult, *wire.BlockHeader, error) {
	blockResult, blockVerboseResult, err := b.getBlockByHeight(height)
	if err != nil {
		return nil, nil, err
	}

	if blockVerboseResult.Confirmations < b.TargetConfirmations {
		return nil, nil, fmt.Errorf("%w, current confirmations:%d target confirmations: %d",
			ErrTargetConfirmations, blockVerboseResult.Confirmations, b.TargetConfirmations)
	}

	blockParsedResult := make([]*types.BitcoinTxParseResult, 0)
	for k, v := range blockResult.Transactions {
		if int64(k) < txIndex {
			continue
		}

		b.logger.Debugw("parse block", "k", k, "height", height, "txIndex", txIndex, "tx", v.TxHash().String())

		parseTxs, err := b.parseTx(v, k)
		if err != nil {
			return nil, nil, err
		}
		if parseTxs != nil {
			blockParsedResult = append(blockParsedResult, parseTxs)
		}
	}

	return blockParsedResult, &blockResult.Header, nil
}

// getBlockByHeight returns a raw block from the server given its height
func (b *Indexer) getBlockByHeight(height int64) (*wire.MsgBlock, *btcjson.GetBlockVerboseResult, error) {
	blockhash, err := b.client.GetBlockHash(height)
	if err != nil {
		return nil, nil, err
	}
	blockVerbose, err := b.client.GetBlockVerbose(blockhash)
	if err != nil {
		return nil, nil, err
	}
	msgBlock, err := b.client.GetBlock(blockhash)
	if err != nil {
		return nil, nil, err
	}
	return msgBlock, blockVerbose, nil
}

// parseTx parse transaction data
func (b *Indexer) parseTx(txResult *wire.MsgTx, index int) (*types.BitcoinTxParseResult, error) {
	listenAddress := false
	var totalValue int64
	tos := make([]types.BitcoinTo, 0)
	for _, v := range txResult.TxOut {
		pkAddress, err := b.parseAddress(v.PkScript)
		if err != nil {
			if errors.Is(err, ErrParsePkScript) {
				continue
			}
			// TODO: handle null data
			if errors.Is(err, ErrParsePkScriptNullData) {
				continue
			}
			return nil, err
		}
		parseTo := types.BitcoinTo{
			Address: pkAddress,
			Value:   v.Value,
		}
		tos = append(tos, parseTo)
		// if pk address eq dest listened address, after parse from address by vin prev tx
		if pkAddress == b.listenAddress.EncodeAddress() {
			listenAddress = true
			totalValue += v.Value
		}
	}
	if listenAddress {
		fromAddress, err := b.parseFromAddress(txResult)
		if err != nil {
			return nil, fmt.Errorf("vin parse err:%w", err)
		}

		// TODO: temp fix, if from is listened address, continue
		if len(fromAddress) == 0 {
			b.logger.Warnw("parse from address empty or nonsupport tx type",
				"txId", txResult.TxHash().String(),
				"listenAddress", b.listenAddress.EncodeAddress())
			return nil, nil
		}

		return &types.BitcoinTxParseResult{
			TxID:   txResult.TxHash().String(),
			TxType: TxTypeTransfer,
			Index:  int64(index),
			Value:  totalValue,
			From:   fromAddress,
			To:     b.listenAddress.EncodeAddress(),
			Tos:    tos,
		}, nil
	}
	return nil, nil
}

// parseFromAddress from vin parse from address
// return all possible values parsed from address
// TODO: at present, it is assumed that it is a single from, and multiple from needs to be tested later
func (b *Indexer) parseFromAddress(txResult *wire.MsgTx) (fromAddress []types.BitcoinFrom, err error) {
	for _, vin := range txResult.TxIn {
		// get prev tx hash
		prevTxID := vin.PreviousOutPoint.Hash
		vinResult, err := b.client.GetRawTransaction(&prevTxID)
		if err != nil {
			return nil, fmt.Errorf("vin get raw transaction err:%w", err)
		}
		if len(vinResult.MsgTx().TxOut) == 0 {
			return nil, fmt.Errorf("vin txOut is null")
		}
		vinPKScript := vinResult.MsgTx().TxOut[vin.PreviousOutPoint.Index].PkScript
		//  script to address
		vinPkAddress, err := b.parseAddress(vinPKScript)
		if err != nil {
			b.logger.Errorw("vin parse address", "error", err)
			if errors.Is(err, ErrParsePkScript) || errors.Is(err, ErrParsePkScriptNullData) {
				continue
			}
			return nil, err
		}
		// parse sign pubkey
		pubKey, err := b.parsePubKey(vin)
		if err != nil {
			if errors.Is(err, ErrParsePubKey) {
				b.logger.Warnw(ErrParsePubKey.Error(),
					"vin", vin,
					"txId", txResult.TxHash().String(),
				)
				continue
			}
			return nil, err
		}
		fromAddress = append(fromAddress, types.BitcoinFrom{
			Address: vinPkAddress,
			PubKey:  pubKey,
		})
	}
	return fromAddress, nil
}

// parseAddress from pkscript parse address
func (b *Indexer) ParseAddress(pkScript []byte) (string, error) {
	return b.parseAddress(pkScript)
}

// parseNullData from pkscript parse null data
//
//lint:ignore U1000 Ignore unused function temporarily for debugging
func (b *Indexer) parseNullData(pkScript []byte) (string, error) {
	pk, err := txscript.ParsePkScript(pkScript)
	if err != nil {
		return "", fmt.Errorf("%w:%s", ErrParsePkScript, err.Error())
	}
	if pk.Class() != txscript.NullDataTy {
		return "", fmt.Errorf("not null data type")
	}
	return pk.String(), nil
}

// parseAddress from pkscript parse address
func (b *Indexer) parseAddress(pkScript []byte) (string, error) {
	pk, err := txscript.ParsePkScript(pkScript)
	if err != nil {
		return "", fmt.Errorf("%w:%s", ErrParsePkScript, err.Error())
	}

	if pk.Class() == txscript.NullDataTy {
		return "", ErrParsePkScriptNullData
	}

	//  encodes the script into an address for the given chain.
	pkAddress, err := pk.Address(b.chainParams)
	if err != nil {
		return "", fmt.Errorf("PKScript to address err:%w", err)
	}
	return pkAddress.EncodeAddress(), nil
}

// LatestBlock get latest block height in the longest block chain.
func (b *Indexer) LatestBlock() (int64, error) {
	return b.client.GetBlockCount()
}

// BlockChainInfo get block chain info
func (b *Indexer) BlockChainInfo() (*btcjson.GetBlockChainInfoResult, error) {
	return b.client.GetBlockChainInfo()
}

func (b *Indexer) parsePubKey(txIn *wire.TxIn) (string, error) {
	if txIn.Witness != nil {
		// only P2WPKH support
		if len(txIn.Witness) == 2 {
			pubkey := txIn.Witness[1]
			return hex.EncodeToString(pubkey), nil
		}
	}
	return "", ErrParsePubKey
}
