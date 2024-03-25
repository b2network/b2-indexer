package types

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
)

// BITCOINBridge defines the interface of custom bitcoin bridge.
type BITCOINBridge interface {
	// Deposit transfers amout to address
	Deposit(string, BitcoinFrom, int64, *types.Transaction, uint64, bool) (*types.Transaction, []byte, string, string, error)
	// Transfer amount to address
	Transfer(BitcoinFrom, int64, *types.Transaction, uint64, bool) (*types.Transaction, string, error)
	// WaitMined wait mined
	WaitMined(context.Context, *types.Transaction, []byte) (*types.Receipt, error)
	// TransactionReceipt
	TransactionReceipt(hash string) (*types.Receipt, error)
	// TransactionByHash
	TransactionByHash(hash string) (*types.Transaction, bool, error)
	//  EnableEoaTransfer
	EnableEoaTransfer() bool
	FromAddress() string
}
