package model

import "github.com/btcsuite/btcd/wire"

const (
	BtcTxWithdrawPending = iota + 1
	BtcTxWithdrawSuccess
	BtcTxWithdrawCompleted
	BtcTxWithdrawSubmitTxMsg
	BtcTxWithdrawSignatureCompleted
	BtcTxWithdrawBroadcastSuccess
)

type Withdraw struct {
	Base
	BtcFrom       string `json:"btc_from" gorm:"type:varchar(256);not null;default:'';index"`
	BtcTo         string `json:"btc_to" gorm:"type:varchar(256);not null;default:'';index"`
	BtcValue      int64  `json:"btc_value" gorm:"type:bigint;default:0;comment:bitcoin transfer value"`
	B2BlockNumber uint64 `json:"b2_block_number" gorm:"type:bigint;comment:b2 block number"`
	B2BlockHash   string `json:"b2_block_hash" gorm:"type:varchar(256);comment:b2 block hash"`
	B2TxHash      string `json:"b2_tx_hash" gorm:"type:varchar(256);not null;default:'';uniqueIndex;comment:b2 network tx hash"`
	B2TxIndex     uint   `json:"b2_tx_index" gorm:"type:bigint;comment:b2 tx index"`
	B2LogIndex    uint   `json:"b2_log_index" gorm:"type:int;comment:b2 log index"`
	Status        int    `json:"status" gorm:"type:smallint;default:1"`
}

type Sign struct {
	TxInIndex int
	Sign      []byte
}

type UnisatResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type UtxoData struct {
	Cursor                int64  `json:"cursor"`
	Total                 int64  `json:"total"`
	TotalConfirmed        int64  `json:"totalConfirmed"`
	TotalUnconfirmed      int64  `json:"totalUnconfirmed"`
	TotalUnconfirmedSpend int64  `json:"totalUnconfirmedSpend"`
	Utxo                  []Utxo `json:"utxo"`
}

type Utxo struct {
	Txid         string        `json:"txid"`
	Vout         int64         `json:"vout"`
	Satoshi      int64         `json:"satoshi"`
	ScriptType   string        `json:"scriptType"`
	ScriptPk     string        `json:"scriptPk"`
	CodeType     int64         `json:"codeType"`
	Address      string        `json:"address"`
	Height       int64         `json:"height"`
	Idx          int64         `json:"idx"`
	IsOpInRBF    bool          `json:"isOpInRBF"`
	IsSpent      bool          `json:"isSpent"`
	Inscriptions []interface{} `json:"inscriptions"`
}

type UnspentOutput struct {
	Outpoint *wire.OutPoint
	Output   *wire.TxOut
}

type WithdrawColumns struct {
	BtcTxHash     string
	BtcTx         string
	BtcSignature  string
	BtcFrom       string
	BtcTo         string
	BtcValue      string
	B2TxHash      string
	B2BlockNumber string
	B2LogIndex    string
	Status        string
}

func (Withdraw) TableName() string {
	return "withdraw_history"
}

func (Withdraw) Column() WithdrawColumns {
	return WithdrawColumns{
		BtcTxHash:     "btc_tx_hash",
		BtcTx:         "btc_tx",
		BtcSignature:  "btc_signature",
		BtcFrom:       "btc_from",
		BtcTo:         "btc_to",
		BtcValue:      "btc_value",
		B2TxHash:      "b2_tx_hash",
		B2BlockNumber: "b2_block_number",
		B2LogIndex:    "b2_log_index",
		Status:        "status",
	}
}
