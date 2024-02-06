package rpc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Block struct {
	ID      int64  `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Difficulty       string        `json:"difficulty"`
		ExtraData        string        `json:"extraData"`
		GasLimit         string        `json:"gasLimit"`
		GasUsed          string        `json:"gasUsed"`
		Hash             string        `json:"hash"`
		LogsBloom        string        `json:"logsBloom"`
		Miner            string        `json:"miner"`
		MixHash          string        `json:"mixHash"`
		Nonce            string        `json:"nonce"`
		Number           string        `json:"number"`
		ParentHash       string        `json:"parentHash"`
		ReceiptsRoot     string        `json:"receiptsRoot"`
		Sha3Uncles       string        `json:"sha3Uncles"`
		Size             string        `json:"size"`
		StateRoot        string        `json:"stateRoot"`
		Timestamp        string        `json:"timestamp"`
		TotalDifficulty  string        `json:"totalDifficulty"`
		Transactions     []interface{} `json:"transactions"`
		TransactionsRoot string        `json:"transactionsRoot"`
		Uncles           []interface{} `json:"uncles"`
	} `json:"result"`
}

func (b Block) ParentHash() string {
	return b.Result.ParentHash
}

func (b Block) Hash() string {
	return b.Result.Hash
}

func (b Block) Miner() string {
	return b.Result.Miner
}

func (b Block) Number() int64 {
	value, err := strconv.ParseInt(strings.ReplaceAll(b.Result.Number, "0x", ""), 16, 64)
	if err != nil {
		return 0
	}
	return value
}

func (b Block) Timestamp() int64 {
	value, err := strconv.ParseInt(strings.ReplaceAll(b.Result.Timestamp, "0x", ""), 16, 64)
	if err != nil {
		return 0
	}
	return value
}

func ParseJSONBlock(data string) Block {
	var c Block
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		fmt.Println("Error =", err)
		return c
	}
	return c
}
