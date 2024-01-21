package model

import (
	"time"
)

const (
	BtcTxTypeTransfer = 0 // transfer

	DepositB2TxStatusSuccess                    = 0 // success
	DepositB2TxStatusPending                    = 1 // pending
	DepositB2TxStatusFailed                     = 2 // deposit invoke failed
	DepositB2TxStatusWaitMinedFailed            = 3 // deposit wait mined failed
	DepositB2TxStatusTxHashExist                = 4 // tx hash exist, deposit have been called
	DepositB2TxStatusWaitMinedStatusFailed      = 5 // deposit wait mined status failed, status != 1
	DepositB2TxStatusInsufficientBalance        = 6 // deposit insufficient balance
	DepositB2TxStatusContextDeadlineExceeded    = 7 // deposit client context deadline exceeded, Chain transaction is stuck
	DepositB2TxStatusFromAccountGasInsufficient = 8 // deposit evm from account gas insufficient

	DepositB2EoaTxStatusSuccess                 = 0 // eoa transfer success
	DepositB2EoaTxStatusPending                 = 1 // eoa transfer pending
	DepositB2EoaTxStatusFailed                  = 2 // eoa transfer failed
	DepositB2EoaTxStatusWaitMinedFailed         = 3 // eoa transfer wait mined failed
	DepositB2EoaTxStatusContextDeadlineExceeded = 7 // eoa transfer client context deadline exceeded
)

type Deposit struct {
	Base
	BtcBlockNumber   int64     `json:"btc_block_number" gorm:"index;comment:bitcoin block number"`
	BtcTxIndex       int64     `json:"btc_tx_index" gorm:"comment:bitcoin tx index"`
	BtcTxHash        string    `json:"btc_tx_hash" gorm:"type:varchar(64);not null;default:'';uniqueIndex;comment:bitcoin tx hash"`
	BtcTxType        int       `json:"btc_tx_type" gorm:"type:SMALLINT;default:0;comment:btc tx type"`
	BtcFroms         string    `json:"btc_froms" gorm:"type:jsonb;comment:bitcoin transfer, from may be multiple"`
	BtcFrom          string    `json:"btc_from" gorm:"type:varchar(64);not null;default:'';index"`
	BtcTo            string    `json:"btc_to" gorm:"type:varchar(64);not null;default:'';index"`
	BtcFromAAAddress string    `json:"btc_from_aa_address" gorm:"type:varchar(42);default:'';comment:from aa address"`
	BtcValue         int64     `json:"btc_value" gorm:"default:0;comment:bitcoin transfer value"`
	B2TxHash         string    `json:"b2_tx_hash" gorm:"type:varchar(66);not null;default:'';index;comment:b2 network tx hash"`
	B2TxStatus       int       `json:"b2_tx_status" gorm:"type:SMALLINT;default:1"`
	B2TxRetry        int       `json:"b2_tx_retry" gorm:"type:SMALLINT;default:0"`
	B2EoaTxHash      string    `json:"b2_eoa_tx_hash" gorm:"type:varchar(66);not null;default:'';comment:b2 network eoa tx hash"`
	B2EoaTxStatus    int       `json:"b2_eoa_tx_status" gorm:"type:SMALLINT;default:1"`
	BtcBlockTime     time.Time `json:"btc_block_time"`
}

type DepositColumns struct {
	BtcBlockNumber   string
	BtcTxIndex       string
	BtcTxHash        string
	BtcTxType        string
	BtcFroms         string
	BtcFrom          string
	BtcTo            string
	BtcFromAAAddress string
	BtcValue         string
	B2TxHash         string
	B2TxStatus       string
	B2TxRetry        string
	B2EoaTxHash      string
	B2EoaTxStatus    string
	BtcBlockTime     string
}

func (Deposit) TableName() string {
	return "deposit_history"
}

func (Deposit) Column() DepositColumns {
	return DepositColumns{
		BtcBlockNumber:   "btc_block_number",
		BtcTxIndex:       "btc_tx_index",
		BtcTxHash:        "btc_tx_hash",
		BtcTxType:        "btc_tx_type",
		BtcFroms:         "btc_froms",
		BtcFrom:          "btc_from",
		BtcTo:            "btc_to",
		BtcFromAAAddress: "btc_from_aa_address",
		BtcValue:         "btc_value",
		B2TxHash:         "b2_tx_hash",
		B2TxStatus:       "b2_tx_status",
		B2EoaTxHash:      "b2_eoa_tx_hash",
		B2EoaTxStatus:    "b2_eoa_tx_status",
		BtcBlockTime:     "btc_block_time",
		B2TxRetry:        "b2_tx_retry",
	}
}
