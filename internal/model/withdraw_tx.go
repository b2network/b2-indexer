package model

type WithdrawTx struct {
	Base
	BtcTxID    string `json:"btc_tx_id" gorm:"type:varchar(256);not null;default:'';uniqueIndex;comment:bitcoin tx id"`
	B2TxHashes string `json:"btc_tx_hashes" gorm:"type:text;not null;default:'';comment:bitcoin tx hash list"`
	BtcTx      string `json:"btc_tx" gorm:"type:text;not null;default:'';comment:bitcoin tx"`
	BtcTxHash  string `json:"btc_txHash" gorm:"type:varchar(256);not null;default:'';comment:bitcoin tx hash"`
	Status     int    `json:"status" gorm:"type:smallint;default:1"`
}

type WithdrawTxColumns struct {
	BtcTxID    string
	BtcTx      string
	B2TxHashes string
	BtcTxHash  string
	Status     string
}

func (WithdrawTx) TableName() string {
	return "withdraw_tx"
}

func (WithdrawTx) Column() WithdrawTxColumns {
	return WithdrawTxColumns{
		BtcTxID:    "btc_tx_id",
		B2TxHashes: "b2_tx_Hashes",
		BtcTx:      "btc_tx",
		BtcTxHash:  "btc_tx_hash",
		Status:     "status",
	}
}
