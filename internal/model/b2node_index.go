package model

type B2NodeIndex struct {
	Base
	IndexBlock int64 `json:"index_block" gorm:"comment:b2node index block"`
	IndexTx    int64 `json:"index_tx" gorm:"comment:b2node index tx"`
}

func (B2NodeIndex) TableName() string {
	return "b2node_index"
}
