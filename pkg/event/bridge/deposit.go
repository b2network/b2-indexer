package bridge

import (
	"encoding/json"

	"github.com/b2network/b2-indexer/pkg/event"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
)

var (
	DepositName = "depositV3"
	DepositHash = crypto.Keccak256([]byte("DepositEvent(address,address,uint256,byte32)"))
)

type Deposit struct {
	Caller    string          `json:"caller"`
	ToAddress string          `json:"to_address"`
	Amount    decimal.Decimal `json:"amount"`
	TxHash    common.Hash     `json:"tx_hash"`
}

func (*Deposit) Name() string {
	return DepositName
}

func (*Deposit) EventHash() common.Hash {
	return common.BytesToHash(DepositHash)
}

func (t *Deposit) ToObj(data string) error {
	err := json.Unmarshal([]byte(data), &t)
	if err != nil {
		return err
	}
	return nil
}

func (*Deposit) Data(log types.Log) (string, error) {
	transfer := &Deposit{
		Caller:    event.TopicToAddress(log, 1).Hex(),
		ToAddress: event.TopicToAddress(log, 2).Hex(),
		Amount:    event.DataToDecimal(log, 0, 0),
		TxHash:    event.DataToHash(log, 1),
	}
	data, err := event.ToJSON(transfer)
	if err != nil {
		return "", err
	}
	return data, nil
}
