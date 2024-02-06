package model

type DepositB2NodeIndex struct {
	Base
	IndexBlock int64 `json:"index_block" gorm:"comment:bitcoin index block"`
	IndexTx    int64 `json:"index_tx" gorm:"comment:bitcoin index tx"`
}

func (DepositB2NodeIndex) TableName() string {
	return "deposit_b2node_index"
}
