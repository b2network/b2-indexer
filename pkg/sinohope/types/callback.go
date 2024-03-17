package types

const (
	RequestTypeWithdrawal = 0
	RequestTypeRecharge   = 1
)

type TransactionNotify struct {
	RequestType   int    `json:"requestType"`
	RequestID     string `json:"requestId"`
	RequestDetail any    `json:"requestDetail"`
	ExtraInfo     string `json:"extraInfo"`
}

type RequestDetail struct {
	SinoID          string `json:"sinoId"`
	TxHash          string `json:"txHash"`
	BlockHash       string `json:"blockHash"`
	ConfirmNumber   int    `json:"confirmNumber"`
	WalletID        string `json:"walletId"`
	ChainSymbol     string `json:"chainSymbol"`
	AssetID         string `json:"assetId"`
	TxDirection     int    `json:"txDirection"`
	Note            string `json:"note"`
	Nonce           int    `json:"nonce"`
	From            string `json:"from"`
	To              string `json:"to"`
	ToTag           string `json:"toTag"`
	Amount          string `json:"amount"`
	Decimal         int    `json:"decimal"`
	Fee             string `json:"fee"`
	FeeAsset        string `json:"feeAsset"`
	FeeDecimal      int    `json:"feeDecimal"`
	FeeAssetDecimal int    `json:"feeAssetDecimal"`
	UsedFee         string `json:"usedFee"`
	GasPrice        string `json:"gasPrice"`
	GasLimit        string `json:"gasLimit"`
	State           int    `json:"state"`
}
