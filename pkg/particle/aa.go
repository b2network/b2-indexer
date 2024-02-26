package particle

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Particle struct {
	particleRPC  string
	particleAuth string
	chainID      int
}

type Req struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      string `json:"id"`
	ChainID int    `json:"chainId"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type Response struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      string `json:"id"`
	ChainID int    `json:"chainId"`
}

type AAGetBTCAccountReqParams struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	BtcPublicKey string `json:"btcPublicKey"`
}

type AAGetBTCAccountResult struct {
	Response
	Result []struct {
		ChainID             int    `json:"chainId"`
		IsDeployed          bool   `json:"isDeployed"`
		EoaAddress          string `json:"eoaAddress"`
		FactoryAddress      string `json:"factoryAddress"`
		EntryPointAddress   string `json:"entryPointAddress"`
		SmartAccountAddress string `json:"smartAccountAddress"`
		Owner               string `json:"owner"`
		Name                string `json:"name"`
		Version             string `json:"version"`
		Index               int    `json:"index"`
		BtcPublicKey        string `json:"btcPublicKey"`
	} `json:"result"`
}

func NewParticle(rpc, projectID, serverKey string, chainID int) (*Particle, error) {
	_, err := url.ParseRequestURI(rpc)
	if err != nil {
		return nil, err
	}
	return &Particle{
		particleRPC:  rpc,
		particleAuth: "Basic " + base64.StdEncoding.EncodeToString([]byte(projectID+":"+serverKey)),
		chainID:      chainID,
	}, nil
}

func (p *Particle) AAGetBTCAccount(btcPubKeys []string) (*AAGetBTCAccountResult, error) {
	params := []AAGetBTCAccountReqParams{}
	for _, pubkey := range btcPubKeys {
		params = append(params, AAGetBTCAccountReqParams{
			Name:         "BTC",
			Version:      "1.0.0",
			BtcPublicKey: pubkey,
		})
	}
	particleReq := Req{
		ID:      strconv.FormatInt(time.Now().UnixMicro(), 10),
		ChainID: p.chainID,
		Method:  "particle_aa_getBTCAccount",
		Params:  params,
		Jsonrpc: "2.0",
	}
	aaGetBTCAccountResult := AAGetBTCAccountResult{}
	err := p.do(particleReq, &aaGetBTCAccountResult)
	if err != nil {
		return nil, err
	}
	return &aaGetBTCAccountResult, nil
}

func (p *Particle) do(particleReq Req, particleResponse any) error {
	bodyJSON, err := json.Marshal(particleReq)
	if err != nil {
		return err
	}

	b := strings.NewReader(string(bodyJSON))
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest("POST", p.particleRPC, b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", p.particleAuth)

	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("StatusCode: %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, &particleResponse)
	if err != nil {
		return err
	}
	return nil
}
