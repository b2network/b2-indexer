package model

import "time"

const (
	EspStatus        = 1
	EspStatusSuccess = 2 // success
)

type Eps struct {
	Base
	DepositID          int64     `json:"deposit_id" gorm:"index;comment:deposit_history id"`
	B2From             string    `json:"b2_from" gorm:"type:varchar(64);not null;default:'';index;commit:b2 from"`
	B2To               string    `json:"b2_to" gorm:"type:varchar(64);not null;default:'';index;commit:b2 to"`
	BtcValue           int64     `json:"btc_value" gorm:"default:0;comment:btc transfer value"`
	B2TxHash           string    `json:"b2_tx_hash" gorm:"type:varchar(66);not null;default:'';index;comment:b2 network tx hash"`
	B2TxTime           time.Time `json:"b2_tx_time" gorm:"type:timestamp;comment:btc tx time"`
	B2BlockNumber      int64     `json:"b2_block_number" gorm:"index;comment:b2 block number"`
	B2TransactionIndex int64     `json:"b2_transaction_index" gorm:"index;comment:b2 transaction index"`
	Status             int       `json:"status" gorm:"type:SMALLINT;default:1"`
}

type EpsColumns struct {
	ID                 string
	DepositID          string
	B2From             string
	B2To               string
	BtcValue           string
	B2TxHash           string
	B2TxTime           string
	B2BlockNumber      string
	B2TransactionIndex string
	Status             string
}

func (Eps) TableName() string {
	return "eps_history"
}

func (Eps) Column() EpsColumns {
	return EpsColumns{
		ID:                 "id",
		DepositID:          "deposit_id",
		B2From:             "b2_from",
		B2To:               "b2_to",
		BtcValue:           "btc_value",
		B2TxHash:           "b2_tx_hash",
		B2TxTime:           "b2_tx_time",
		B2BlockNumber:      "b2_block_number",
		B2TransactionIndex: "b2_transaction_index",
		Status:             "status",
	}
}
