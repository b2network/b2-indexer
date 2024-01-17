package model

import "time"

const (
	BtcTxTypeTransfer = 0 // transfer

	DepositStatusPending = 0 // pending
	DepositStatusSuccess = 1 // success
	DepositStatusFailed  = 2 // failed
)

type Deposit struct {
	Base
	BtcBlockNumber int64  `json:"btc_block_number" gorm:"index;comment:bitcoin block number"`
	BtcTxIndex     int64  `json:"btc_tx_index" gorm:"comment:bitcoin tx index"`
	BtcTxHash      string `json:"btc_tx_hash" gorm:"type:varchar(64);not null;uniqueIndex;comment:bitcoin tx hash"`
	B2TxHash       string `json:"b2_tx_hash" gorm:"type:varchar(66);not null;index;comment:b2 network tx hash"`
	// B2L1TxHash     string    `json:"b2_l1_tx_hash" gorm:"type:varchar(66);not null;index;comment:b2 l1 network tx hash"`
	BtcTxType     int       `json:"btc_tx_type" gorm:"type:SMALLINT;comment:btc tx type"`
	Froms         string    `json:"froms" gorm:"type:jsonb;comment:bitcoin transfer, from may be multiple"`
	From          string    `json:"from" gorm:"type:varchar(64);not null;index"`
	To            string    `json:"to" gorm:"type:varchar(64);not null;index"`
	FromAAAddress string    `json:"from_aa_address" gorm:"type:varchar(42);comment:from aa address"`
	Value         int64     `json:"value" gorm:"comment:bitcoin transfer value"`
	Status        int       `json:"status" gorm:"type:SMALLINT"`
	BtcBlockTime  time.Time `json:"btc_block_time"`
}

func (Deposit) TableName() string {
	return "`deposit_history`"
}
