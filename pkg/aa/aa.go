package aa

import (
	"encoding/json"

	"github.com/b2network/b2-indexer/pkg/log"
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

func GetPubKey(api, btcAddress string) (*Response, error) {
	res, err := rpc.HTTPGet(api + "/v1/btc/pubkey/" + btcAddress)
	if err != nil {
		return nil, err
	}

	log.Infof("get pubkey response:%v", string(res))

	btcResp := Response{}

	err = json.Unmarshal(res, &btcResp)
	if err != nil {
		return nil, err
	}

	return &btcResp, nil
}
