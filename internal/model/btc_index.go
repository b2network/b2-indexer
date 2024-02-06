package model

type BtcIndex struct {
	Base
	BtcIndexBlock int64 `json:"btc_index_block" gorm:"comment:bitcoin index block"`
	BtcIndexTx    int64 `json:"btc_index_tx" gorm:"comment:bitcoin index tx"`
}

func (BtcIndex) TableName() string {
	return "btc_index"
}
