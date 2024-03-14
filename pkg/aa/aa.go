package aa

import (
	"encoding/json"
	"fmt"

	"github.com/b2network/b2-indexer/pkg/rpc"
)

var AddressNotFoundErrCode = "1001"

type Response struct {
	Code    string
	Message string
	Data    struct {
		Pubkey string
	}
}

func GetPubKey(api, btcAddress string) (string, string, error) {
	res, err := rpc.HTTPGet(api + "/" + btcAddress)
	if err != nil {
		return "", "", err
	}

	btcResp := Response{}

	err = json.Unmarshal(res, &btcResp)
	if err != nil {
		return "", "", err
	}
	if btcResp.Code != "0" {
		return "", "", fmt.Errorf("GetPubKey err: %s", btcResp.Message)
	}

	return btcResp.Code, btcResp.Data.Pubkey, nil
}
