package bitcoin

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type WithdrawEvent struct {
	FromAddress common.Address
	ToAddress   string
	Amount      *big.Int
}

type DepositEvent struct {
	Sender    common.Address
	ToAddress common.Address
	Amount    *big.Int
}

func TopicToHash(vLog ethtypes.Log, index int64) common.Hash {
	return common.BytesToHash(vLog.Topics[index].Bytes())
}

func TopicToAddress(vLog ethtypes.Log, index int64) common.Address {
	return common.BytesToAddress(vLog.Topics[index].Bytes())
}

func DataToBigInt(vLog ethtypes.Log, index int64) *big.Int {
	start := 32 * index
	return big.NewInt(0).SetBytes(vLog.Data[start : start+32])
}

func DataToString(vLog ethtypes.Log, index int64) string {
	start := 32 * index
	offset := big.NewInt(0).SetBytes(vLog.Data[start : start+32]).Int64()
	length := big.NewInt(0).SetBytes(vLog.Data[offset : offset+32]).Int64()
	return string(vLog.Data[offset+32 : offset+32+length])
}
