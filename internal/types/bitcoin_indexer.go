package types

// BITCOINTxIndexer defines the interface of custom bitcoin tx indexer.
type BITCOINTxIndexer interface {
	// ParseBlock parse bitcoin block tx
	ParseBlock(int64, int64) ([]*BitcoinTxParseResult, error)
	// LatestBlock get latest block height in the longest block chain.
	LatestBlock() (int64, error)
}
