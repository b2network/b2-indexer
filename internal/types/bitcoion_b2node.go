package types

import (
	bridgeTypes "github.com/evmos/ethermint/x/bridge/types"
)

// BITCOINBridgeB2Node defines the interface of custom bitcoin bridge.
type BITCOINBridgeB2Node interface {
	CreateDeposit(hash string, from string, to string, value uint64) error
	QueryDeposit(hash string) (*bridgeTypes.Deposit, error)
}
