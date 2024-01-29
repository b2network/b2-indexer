package model

import "time"

const (
	EspStatus        = 1
	EspStatusSuccess = 2 // success
)

type Eps struct {
	Base
	DepositId          int64     `json:"deposit_id" gorm:"index;comment:deposit_history id"`
	B2From             string    `json:"b2_from" gorm:"type:varchar(64);not null;default:'b2 from';index"`
	B2To               string    `json:"b2_to" gorm:"type:varchar(64);not null;default:'b2 to';index"`
	BtcValue           int64     `json:"btc_value" gorm:"default:0;comment:btc transfer value"`
	B2TxHash           string    `json:"b2_tx_hash" gorm:"type:varchar(66);not null;default:'';index;comment:b2 network tx hash"`
	B2TxTime           time.Time `json:"b2_tx_time" gorm:"type:timestamp;comment:btc tx time"`
	B2BlockNumber      int64     `json:"b2_block_number" gorm:"index;comment:b2 block number"`
	B2TransactionIndex int64     `json:"b2_transaction_index" gorm:"index;comment:b2 transaction index"`
	Status             int       `json:"status" gorm:"type:SMALLINT;default:1"`
}

func (Eps) TableName() string {
	return "eps_history"
}
