package event

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

func DataToAddress(vLog types.Log, index int64) common.Address {
	start := 32 * index
	return common.BytesToAddress(vLog.Data[start : start+32])
}

func DataToInt64(vLog types.Log, index int64) int64 {
	start := 32 * index
	return big.NewInt(0).SetBytes(vLog.Data[start : start+32]).Int64()
}

func DataToBool(vLog types.Log, index int64) bool {
	start := 32 * index
	return big.NewInt(0).SetBytes(vLog.Data[start:start+32]).Int64() == 1
}

func DataToDecimal(vLog types.Log, index int64, exp int32) decimal.Decimal {
	start := 32 * index
	return decimal.NewFromBigInt(big.NewInt(0).SetBytes(vLog.Data[start:start+32]), exp)
}

func DataToHash(vLog types.Log, index int64) common.Hash {
	start := 32 * index
	return common.BytesToHash(vLog.Data[start : start+32])
}

func DataToAddressArray(vLog types.Log, index int64) []string {
	offset := big.NewInt(0).SetBytes(vLog.Data[32*index : 32*(index+1)]).Int64()
	length := big.NewInt(0).SetBytes(vLog.Data[offset : offset+32]).Int64()
	array := make([]string, 0)
	var i int64
	for i = 0; i < length; i++ {
		start := offset + 32*(i+1)
		one := common.BytesToAddress(vLog.Data[start : start+32])
		array = append(array, one.Hex())
	}
	return array
}

func DataToHashArray(vLog types.Log, index int64) []string {
	offset := big.NewInt(0).SetBytes(vLog.Data[32*index : 32*(index+1)]).Int64()
	length := big.NewInt(0).SetBytes(vLog.Data[offset : offset+32]).Int64()
	array := make([]string, 0)
	var i int64
	for i = 0; i < length; i++ {
		start := offset + 32*(i+1)
		one := common.BytesToHash(vLog.Data[start : start+32])
		array = append(array, one.Hex())
	}
	return array
}

func DataToInt64Array(vLog types.Log, index int64) []int64 {
	offset := big.NewInt(0).SetBytes(vLog.Data[32*index : 32*(index+1)]).Int64()
	length := big.NewInt(0).SetBytes(vLog.Data[offset : offset+32]).Int64()
	array := make([]int64, 0)
	var i int64
	for i = 0; i < length; i++ {
		start := offset + 32*(i+1)
		one := big.NewInt(0).SetBytes(vLog.Data[start : start+32]).Int64()
		array = append(array, one)
	}
	return array
}

func DataToDecimalArray(vLog types.Log, index int64, exp int32) []decimal.Decimal {
	offset := big.NewInt(0).SetBytes(vLog.Data[32*index : 32*(index+1)]).Int64()
	length := big.NewInt(0).SetBytes(vLog.Data[offset : offset+32]).Int64()
	array := make([]decimal.Decimal, 0)
	var i int64
	for i = 0; i < length; i++ {
		start := offset + 32*(i+1)
		one := decimal.NewFromBigInt(big.NewInt(0).SetBytes(vLog.Data[start:start+32]), exp)
		array = append(array, one)
	}
	return array
}

func TopicToAddress(vLog types.Log, index int64) common.Address {
	return common.BytesToAddress(vLog.Topics[index].Bytes())
}

func TopicToInt64(vLog types.Log, index int64) int64 {
	return big.NewInt(0).SetBytes(vLog.Topics[index].Bytes()).Int64()
}

func TopicToBool(vLog types.Log, index int64) bool {
	return big.NewInt(0).SetBytes(vLog.Topics[index].Bytes()).Int64() == 1
}

func DataToArrayOffsetAndLength(vLog types.Log, index int64) (int64, int64) {
	offset := big.NewInt(0).SetBytes(vLog.Data[32*index : 32*(index+1)]).Int64()
	length := big.NewInt(0).SetBytes(vLog.Data[offset : offset+32]).Int64()
	return offset, length
}

func TopicToDecimal(vLog types.Log, index int64, exp int32) decimal.Decimal {
	return decimal.NewFromBigInt(big.NewInt(0).SetBytes(vLog.Topics[index].Bytes()), exp)
}

func TopicToHash(vLog types.Log, index int64) common.Hash {
	return common.BytesToHash(vLog.Topics[index].Bytes())
}

func TopicToInt64Array(vLog types.Log, index int64) []int64 {
	offset := big.NewInt(0).SetBytes(vLog.Topics[index].Bytes()).Int64()
	length := big.NewInt(0).SetBytes(vLog.Data[offset : offset+32]).Int64()
	array := make([]int64, 0)
	var i int64
	for i = 0; i < length; i++ {
		start := offset + 32*(i+1)
		one := big.NewInt(0).SetBytes(vLog.Data[start : start+32]).Int64()
		array = append(array, one)
	}
	return array
}

func TopicToArrayOffsetAndLength(vLog types.Log, index int64) (int64, int64) {
	offset := big.NewInt(0).SetBytes(vLog.Topics[index].Bytes()).Int64()
	length := big.NewInt(0).SetBytes(vLog.Data[offset : offset+32]).Int64()
	return offset, length
}

func StartAndEndTokenIDToString(startTokenID, endTokenID int64) (string, int64) {
	str := ""
	var count int64
	for startTokenID <= endTokenID {
		if str == "" {
			str = fmt.Sprintf("%s%d", str, 0)
		} else {
			str = fmt.Sprintf("%s,%d", str, 0)
		}
		count++
		startTokenID++
	}
	return str, count
}

func TokenIDsToString(tokenIDs []int64) string {
	str := ""
	for _, tokenID := range tokenIDs {
		if str == "" {
			str = fmt.Sprintf("%s%d", str, tokenID)
		} else {
			str = fmt.Sprintf("%s,%d", str, tokenID)
		}
	}
	return str
}

func DataToString(vLog types.Log, index int64) string {
	start := 32 * index
	offset := big.NewInt(0).SetBytes(vLog.Data[start : start+32]).Int64()
	length := big.NewInt(0).SetBytes(vLog.Data[offset : offset+32]).Int64()
	return string(vLog.Data[offset+32 : offset+32+length])
}

func GetSender(rpc *ethclient.Client, blockHash, txHash common.Hash, txIndex uint) (*common.Address, error) {
	ctx := context.Background()
	tx, _, err := rpc.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}
	sender, err := rpc.TransactionSender(ctx, tx, blockHash, txIndex)
	if err != nil {
		return nil, err
	}
	return &sender, nil
}

func ToJSON(obj interface{}) (string, error) {
	jsonValue, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(jsonValue), nil
}
