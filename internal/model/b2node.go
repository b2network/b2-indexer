package model

type B2Node struct {
	Base
	Height              int64  `json:"height" gorm:"column:height"`
	BridgeModuleTxIndex int    `json:"bridge_module_tx_index" gorm:"column:bridge_module_tx_index"`
	TxHash              string `json:"tx_hash" gorm:"column:tx_hash;type:varchar(66);not null;default:''"`
	EventType           string `json:"event_type" gorm:"column:event_type;type:varchar(60);not null;default:''"`
	Messages            string `json:"messages" gorm:"column:messages;not null;default:''"`
	RawLog              string `json:"raw_log" gorm:"column:raw_log;not null;default:''"`
	TxCode              int    `json:"tx_code" gorm:"column:tx_code"`
	TxData              string `json:"tx_data" gorm:"column:tx_data;not null;default:''"`
	BridgeEventID       string `json:"bridge_event_id" gorm:"column:bridge_event_id;type:varchar(66);not null;default:''"`
	Status              int    `json:"status" gorm:"column:status;type:smallint;default:1"`
}

func (B2Node) TableName() string {
	return "b2node_history"
}

type B2NodeColumns struct {
	Height              string
	BridgeModuleTxIndex string
	TxHash              string
	EventType           string
	Messages            string
	RawLog              string
	TxCode              string
	TxData              string
	BridgeEventID       string
	Status              string
}

func (B2Node) Column() B2NodeColumns {
	return B2NodeColumns{
		Height:              "height",
		BridgeModuleTxIndex: "bridge_module_tx_index",
		TxHash:              "tx_hash",
		EventType:           "event_type",
		Messages:            "messages",
		RawLog:              "raw_log",
		TxCode:              "tx_code",
		TxData:              "tx_data",
		BridgeEventID:       "bridge_event_id",
		Status:              "status",
	}
}
