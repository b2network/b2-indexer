package types

import "github.com/btcsuite/btcd/wire"

// BITCOINTxIndexer defines the interface of custom bitcoin tx indexer.
type BITCOINTxIndexer interface {
	// ParseBlock parse bitcoin block tx
	ParseBlock(int64, int64) ([]*BitcoinTxParseResult, *wire.BlockHeader, error)
	// LatestBlock get latest block height in the longest block chain.
	LatestBlock() (int64, error)
}

type BitcoinTxParseResult struct {
	// from is l2 user address, by parse bitcoin get the address
	From []string
	// to is listening address
	To string
	// value is from transfer amount
	Value int64
	// tx_id is the btc transaction id
	TxID string
	// tx_type is the type of the transaction, eg. "brc20_transfer","transfer"
	TxType string
	// index is the index of the transaction in the block
	Index int64
}
