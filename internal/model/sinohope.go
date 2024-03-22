package model

type Sinohope struct {
	Base
	RequestID     string `json:"request_id" gorm:"type:varchar(64);not null;default:'';uniqueIndex;comment:sinohope request id"`
	RequestType   int    `json:"request_type" gorm:"type:SMALLINT;default:0;comment:sinohope callback type"`
	RequestDetail string `json:"request_detail" gorm:"type:jsonb;comment:sinohope request detail"`
	ExtraInfo     int    `json:"extra_info" gorm:"type:jsonb;comment:sinohope request extra_info"`
}

type SinohopeColumns struct {
	RequestID     string
	RequestType   string
	RequestDetail string
	ExtraInfo     string
}

func (Sinohope) TableName() string {
	return "sinohope"
}

func (Sinohope) Column() SinohopeColumns {
	return SinohopeColumns{
		RequestID:     "request_id",
		RequestType:   "request_type",
		RequestDetail: "request_detail",
		ExtraInfo:     "extra_info",
	}
}
