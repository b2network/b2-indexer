package model

type RollupDeposit struct {
	Base
	BtcTxHash        string `json:"btc_tx_hash" gorm:"type:varchar(64);not null;default:'';comment:bitcoin tx hash"`
	BtcFromAAAddress string `json:"btc_from_aa_address" gorm:"type:varchar(42);default:'';comment:from aa address"`
	BtcValue         int64  `json:"btc_value" gorm:"type:bigint;default:0;comment:bitcoin transfer value"`
	B2BlockNumber    uint64 `json:"b2_block_number" gorm:"type:bigint;comment:b2 block number"`
	B2BlockHash      string `json:"b2_block_hash" gorm:"type:varchar(256);comment:b2 block hash"`
	B2TxFrom         string `json:"b2_tx_from" gorm:"type:varchar(42);default:'';comment:from address"`
	B2TxHash         string `json:"b2_tx_hash" gorm:"type:varchar(256);default:'';uniqueIndex;comment:b2 network tx hash"`
	B2TxIndex        uint   `json:"b2_tx_index" gorm:"type:bigint;comment:b2 tx index"`
	B2LogIndex       uint   `json:"b2_log_index" gorm:"type:int;comment:b2 log index"`
	Status           int    `json:"status" gorm:"type:smallint;default:1"`
}

func (RollupDeposit) TableName() string {
	return "rollup_deposit_history"
}

type RollupDepositColumns struct {
	BtcTxHash        string
	BtcFromAAAddress string
	BtcValue         string
	B2TxFrom         string
	B2BlockNumber    string
	B2BlockHash      string
	B2TxHash         string
	B2TxIndex        string
	B2LogIndex       string
	Status           string
}

func (RollupDeposit) Column() RollupDepositColumns {
	return RollupDepositColumns{
		BtcTxHash:        "btc_tx_hash",
		BtcFromAAAddress: "btc_from_aa_address",
		BtcValue:         "btc_value",
		B2TxFrom:         "b2_tx_from",
		B2TxHash:         "b2_tx_hash",
		B2BlockNumber:    "b2_block_number",
		B2BlockHash:      "b2_block_hash",
		B2TxIndex:        "b2_tx_index",
		B2LogIndex:       "b2_log_index",
		Status:           "status",
	}
}
