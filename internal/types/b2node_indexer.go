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
	// UpdateDeposit b2node update deposit record
	UpdateDeposit(hash string, status bridgeTypes.DepositStatus, rollupTxHash string, fromAa string) error
	// LatestBlock get b2node latest block number
	LatestBlock() (int64, error)
	// CreateWithdraw b2node create withdraw record
	CreateWithdraw(txID string, txHashList []string, encodedData string) error
	// QueryWithdraw query withdraw record
	QueryWithdraw(txID string) (*bridgeTypes.Withdraw, error)
	// UpdateWithdraw b2node update withdraw status
	UpdateWithdraw(txID string, status bridgeTypes.WithdrawStatus) error
	// DeleteWithdraw b2node delete withdraw
	DeleteWithdraw(txID string) error
	// QueryWithdrawByTxHash queries a withdraw record by tx_hash.
	QueryWithdrawByTxHash(txHash string) (*bridgeTypes.RollupTx, error)
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
