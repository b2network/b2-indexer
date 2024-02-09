package b2node

import (
	"encoding/json"
	"time"
)

type B2NodeTxs struct { //nolint
	Txs         []Txs         `json:"txs"`
	TxResponses []TxResponses `json:"tx_responses"`
	Pagination  interface{}   `json:"pagination"`
	Total       string        `json:"total"`
}

type Body struct {
	Messages                    []any         `json:"messages"`
	Memo                        string        `json:"memo"`
	TimeoutHeight               string        `json:"timeout_height"`
	ExtensionOptions            []interface{} `json:"extension_options"`
	NonCriticalExtensionOptions []interface{} `json:"non_critical_extension_options"`
}
type PublicKey struct {
	Type string `json:"@type"`
	Key  string `json:"key"`
}
type Single struct {
	Mode string `json:"mode"`
}
type ModeInfo struct {
	Single Single `json:"single"`
}
type SignerInfos struct {
	PublicKey PublicKey `json:"public_key"`
	ModeInfo  ModeInfo  `json:"mode_info"`
	Sequence  string    `json:"sequence"`
}
type Amount struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}
type Fee struct {
	Amount   []Amount `json:"amount"`
	GasLimit string   `json:"gas_limit"`
	Payer    string   `json:"payer"`
	Granter  string   `json:"granter"`
}
type AuthInfo struct {
	SignerInfos []SignerInfos `json:"signer_infos"`
	Fee         Fee           `json:"fee"`
	Tip         interface{}   `json:"tip"`
}
type Txs struct {
	Body       Body     `json:"body"`
	AuthInfo   AuthInfo `json:"auth_info"`
	Signatures []string `json:"signatures"`
}
type Attributes struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type Events struct {
	Type       string       `json:"type"`
	Attributes []Attributes `json:"attributes"`
}
type Logs struct {
	MsgIndex int      `json:"msg_index"`
	Log      string   `json:"log"`
	Events   []Events `json:"events"`
}
type Tx struct {
	Type       string   `json:"@type"`
	Body       Body     `json:"body"`
	AuthInfo   AuthInfo `json:"auth_info"`
	Signatures []string `json:"signatures"`
}
type TxResponses struct {
	Height    string    `json:"height"`
	Txhash    string    `json:"txhash"`
	Codespace string    `json:"codespace"`
	Code      int       `json:"code"`
	Data      string    `json:"data"`
	RawLog    string    `json:"raw_log"`
	Logs      []Logs    `json:"logs"`
	Info      string    `json:"info"`
	GasWanted string    `json:"gas_wanted"`
	GasUsed   string    `json:"gas_used"`
	Tx        Tx        `json:"tx"`
	Timestamp time.Time `json:"timestamp"`
	Events    []Events  `json:"events"`
}

type B2NodeBlock struct { //nolint
	BlockID  BlockID  `json:"block_id"`
	Block    Block    `json:"block"`
	SdkBlock SdkBlock `json:"sdk_block"`
}
type PartSetHeader struct {
	Total int    `json:"total"`
	Hash  string `json:"hash"`
}
type BlockID struct {
	Hash          string        `json:"hash"`
	PartSetHeader PartSetHeader `json:"part_set_header"`
}
type Version struct {
	Block string `json:"block"`
	App   string `json:"app"`
}
type LastBlockID struct {
	Hash          string        `json:"hash"`
	PartSetHeader PartSetHeader `json:"part_set_header"`
}
type Header struct {
	Version            Version     `json:"version"`
	ChainID            string      `json:"chain_id"`
	Height             string      `json:"height"`
	Time               time.Time   `json:"time"`
	LastBlockID        LastBlockID `json:"last_block_id"`
	LastCommitHash     string      `json:"last_commit_hash"`
	DataHash           string      `json:"data_hash"`
	ValidatorsHash     string      `json:"validators_hash"`
	NextValidatorsHash string      `json:"next_validators_hash"`
	ConsensusHash      string      `json:"consensus_hash"`
	AppHash            string      `json:"app_hash"`
	LastResultsHash    string      `json:"last_results_hash"`
	EvidenceHash       string      `json:"evidence_hash"`
	ProposerAddress    string      `json:"proposer_address"`
}
type Data struct {
	Txs []interface{} `json:"txs"`
}
type Evidence struct {
	Evidence []interface{} `json:"evidence"`
}
type Signatures struct {
	BlockIDFlag      string    `json:"block_id_flag"`
	ValidatorAddress string    `json:"validator_address"`
	Timestamp        time.Time `json:"timestamp"`
	Signature        string    `json:"signature"`
}
type LastCommit struct {
	Height     string       `json:"height"`
	Round      int          `json:"round"`
	BlockID    BlockID      `json:"block_id"`
	Signatures []Signatures `json:"signatures"`
}
type Block struct {
	Header     Header     `json:"header"`
	Data       Data       `json:"data"`
	Evidence   Evidence   `json:"evidence"`
	LastCommit LastCommit `json:"last_commit"`
}
type SdkBlock struct {
	Header     Header     `json:"header"`
	Data       Data       `json:"data"`
	Evidence   Evidence   `json:"evidence"`
	LastCommit LastCommit `json:"last_commit"`
}

type CreateDepositMessages struct {
	Type     string `json:"@type"`
	Creator  string `json:"creator"`
	TxHash   string `json:"tx_hash"`
	From     string `json:"from"`
	To       string `json:"to"`
	CoinType string `json:"coin_type"`
	Value    string `json:"value"`
	Data     string `json:"data"`
}

type Bech32Prefix struct {
	Bech32Prefix string `json:"bech32_prefix"`
}

func ParseJSONB2Node(data []byte, dest any) error {
	if err := json.Unmarshal(data, &dest); err != nil {
		return err
	}
	return nil
}
