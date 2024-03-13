package types

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
)

// BITCOINBridge defines the interface of custom bitcoin bridge.
type BITCOINBridge interface {
	// Deposit transfers amount to address
	Deposit(string, string, int64) (*types.Transaction, []byte, string, error)
	// Transfer amount to address
	Transfer(string, int64) (*types.Transaction, error)
	// WaitMined wait mined
	WaitMined(context.Context, *types.Transaction, []byte) (*types.Receipt, error)
}
