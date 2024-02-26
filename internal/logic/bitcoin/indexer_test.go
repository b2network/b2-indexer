package bitcoin_test

import (
	"testing"

	"github.com/b2network/b2-indexer/internal/config"
	"github.com/b2network/b2-indexer/internal/logic/bitcoin"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"

	// tmlog "github.com/cometbft/cometbft/libs/log"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestNewBitcoinIndexer(t *testing.T) {
	testCases := []struct {
		name           string
		listendAddress string
		errMsg         string
	}{
		{
			"success",
			"tb1qukxc3sy3s3k5n5z9cxt3xyywgcjmp2tzudlz2n",
			"",
		},
		{
			"success: segwit",
			"3HctoF43JZCjAQrad1MqGtn5EsF57f5CCN",
			"",
		},
		{
			"success: legacy",
			"1CpnsCEQ3Q4d15rLkrANnfd9GtYNHJRsYb",
			"",
		},
		{
			"success: segwit(bech32)",
			"bc1qj2hkaplmmka9lqjj4p23t2z2lrd4vv8fjqa36g",
			"",
		},
		{
			"fail: format fail",
			"tb1qukxc3sy3s3k5n5z9cxt3xyywgcjmp2tzudlz2n1",
			"decode listen address err:decoded address is of unknown format",
		},
		{
			"fail: address null",
			"",
			"decode listen address err:decoded address is of unknown format",
		},
		{
			"fail: check sum",
			"1CpnsCEQ3Q4d15rLkrANnfd9GtYNHJRsYy",
			"decode listen address err:checksum mismatch",
		},
	}

	for _, tc := range testCases {
		_, err := bitcoin.NewBitcoinIndexer(
			log.NewNopLogger(),
			mockRpcClient(t),
			&chaincfg.MainNetParams, // chainParams Do not affect the address
			tc.listendAddress,
			1,
		)
		if err != nil {
			require.EqualError(t, err, tc.errMsg)
		}
	}
}

func TestParseAddress(t *testing.T) {
	testCases := []struct {
		name          string
		pkScriptHex   string
		parsedAddress string
		pkAddress     string
		chainParams   *chaincfg.Params
		errMsg        string
	}{
		{
			"success",
			"0x51207099e4b23427fc40ba4777bbf52cfd0b7444d69a3e21ef281270723f54c0c14b",
			"tb1pwzv7fv35yl7ypwj8w7al2t8apd6yf4568cs772qjwper74xqc99sk8x7tk",
			"tb1pwzv7fv35yl7ypwj8w7al2t8apd6yf4568cs772qjwper74xqc99sk8x7tk",
			&chaincfg.SigNetParams,
			"",
		},
		{
			"success: main net",
			"0x5120916e7f2636a8754793a5257198d9bef0d6afbea8d09cc2a36b5901869d6b0ad5",
			"bc1pj9h87f3k4p650ya9y4ce3kd77rt2l04g6zwv9gmttyqcd8ttpt2sva77pe",
			"bc1pj9h87f3k4p650ya9y4ce3kd77rt2l04g6zwv9gmttyqcd8ttpt2sva77pe",
			&chaincfg.MainNetParams,
			"",
		},
		{
			"success: sim net",
			"0x5120916e7f2636a8754793a5257198d9bef0d6afbea8d09cc2a36b5901869d6b0ad5",
			"sb1pj9h87f3k4p650ya9y4ce3kd77rt2l04g6zwv9gmttyqcd8ttpt2suyzkzn",
			"sb1pj9h87f3k4p650ya9y4ce3kd77rt2l04g6zwv9gmttyqcd8ttpt2suyzkzn",
			&chaincfg.SimNetParams,
			"",
		},
		{
			"fail: unsupported script type",
			"0x51207099e4b23427fc40ba4777bbf52cfd0b7444d69a3e21ef281270723f54c0c1",
			"1CpnsCEQ3Q4d15rLkrANnfd9GtYNHJRsYb",
			"tb1pwzv7fv35yl7ypwj8w7al2t8apd6yf4568cs772qjwper74xqc99sk8x7tk",
			&chaincfg.SigNetParams,
			"parse pkscript err:unsupported script type",
		},
		{
			"fail: empty pk",
			"0x",
			"1CpnsCEQ3Q4d15rLkrANnfd9GtYNHJRsYb",
			"tb1pwzv7fv35yl7ypwj8w7al2t8apd6yf4568cs772qjwper74xqc99sk8x7tk",
			&chaincfg.SigNetParams,
			"parse pkscript err:unsupported script type",
		},
	}

	for _, tc := range testCases {
		pkScript, err := hexutil.Decode(tc.pkScriptHex)
		require.NoError(t, err)
		tmpAddress, err := mockBitcoinIndexer(t, tc.chainParams).ParseAddress(pkScript)
		if err != nil {
			require.EqualError(t, err, tc.errMsg)
			continue
		}
		if tmpAddress != tc.parsedAddress && tmpAddress != tc.pkAddress {
			t.Errorf("test:%s expected %s, got %s", tc.name, tc.parsedAddress, tmpAddress)
		}
	}
}

// TestLocalParseTx only test in local
// data source: testnet network
func TestLocalParseTx(t *testing.T) {
	to := "tb1qjda2l5spwyv4ekwe9keddymzuxynea2m2kj0qy"
	indexer := bitcoinIndexerWithConfig(t, to)
	testCases := []struct {
		name   string
		height int64
	}{
		{
			name:   "success",
			height: 2540186,
		},
		{
			name:   "success empty",
			height: 2540180,
		},
	}

	for _, tc := range testCases {
		results, _, err := indexer.ParseBlock(tc.height, 0)
		require.NoError(t, err)
		for _, v := range results {
			if len(v.From) == 0 {
				t.Error("From is empty")
			}

			if len(v.Tos) == 0 {
				t.Error("Tos is empty")
			}
			require.Equal(t, v.To, to)
		}
	}
}

// TestLocalLatestBlock only test in local
func TestLocalLatestBlock(t *testing.T) {
	indexer := bitcoinIndexerWithConfig(t, "")
	_, err := indexer.LatestBlock()
	require.NoError(t, err)
}

// TestLocalBlockChainInfo only test in local
func TestLocalBlockChainInfo(t *testing.T) {
	indexer := bitcoinIndexerWithConfig(t, "")
	_, err := indexer.BlockChainInfo()
	require.NoError(t, err)
}

func mockRpcClient(t *testing.T) *rpcclient.Client {
	cfg, err := config.LoadBitcoinConfig("")
	require.NoError(t, err)
	connCfg := &rpcclient.ConnConfig{
		Host:         cfg.RPCHost + ":" + cfg.RPCPort,
		User:         cfg.RPCUser,
		Pass:         cfg.RPCPass,
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	client, err := rpcclient.New(connCfg, nil)
	require.NoError(t, err)
	return client
}

func mockBitcoinIndexer(t *testing.T, chainParams *chaincfg.Params) *bitcoin.Indexer {
	cfg, err := config.LoadBitcoinConfig("../../config/testdata")
	indexer, err := bitcoin.NewBitcoinIndexer(
		log.NewNopLogger(),
		mockRpcClient(t),
		chainParams,
		cfg.IndexerListenAddress,
		cfg.IndexerListenTargetConfirmations)
	require.NoError(t, err)
	return indexer
}

func bitcoinIndexerWithConfig(t *testing.T, indexListenAddress string) *bitcoin.Indexer {
	cfg, err := config.LoadBitcoinConfig("")
	require.NoError(t, err)
	connCfg := &rpcclient.ConnConfig{
		Host:         cfg.RPCHost + ":" + cfg.RPCPort,
		User:         cfg.RPCUser,
		Pass:         cfg.RPCPass,
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	client, err := rpcclient.New(connCfg, nil)
	require.NoError(t, err)
	bitcoinParam := config.ChainParams(cfg.NetworkName)
	if indexListenAddress == "" {
		indexListenAddress = cfg.IndexerListenAddress
	}
	indexer, err := bitcoin.NewBitcoinIndexer(
		log.NewNopLogger(),
		client,
		bitcoinParam,
		indexListenAddress,
		cfg.IndexerListenTargetConfirmations,
	)
	require.NoError(t, err)
	return indexer
}
