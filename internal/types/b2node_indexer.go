package types

import (
	bridgeTypes "github.com/evmos/ethermint/x/bridge/types"
)

// B2NODEBridge defines the interface of custom b2-node bridge.
type B2NODEBridge interface {
	// ParseBlockBridgeEvent parse b2node block
	ParseBlockBridgeEvent(height int64, index int64) ([]*B2NodeTxParseResult, error)
	// CreateDeposit b2node create deposit record
	CreateDeposit(hash string, from string, to string, value int64) error
	// QueryDeposit query deposit record
	QueryDeposit(hash string) (*bridgeTypes.Deposit, error)
	// LatestBlock get b2node latest block number
	LatestBlock() (int64, error)
}

type B2NodeTxParseResult struct {
	Height              int64
	BridgeModuleTxIndex int
	TxHash              string
	EventType           string
	Messages            string
	RawLog              string
	TxCode              int
	TxData              string
	BridgeEventID       string
}
