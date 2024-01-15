package config_test

import (
	"os"
	"testing"

	"github.com/b2network/b2-indexer/internal/config"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	// clean BITCOIN env set
	// This is because the value set by the environment variable affects viper reading file
	os.Unsetenv("BITCOIN_NETWORK_NAME")
	os.Unsetenv("BITCOIN_RPC_HOST")
	os.Unsetenv("BITCOIN_RPC_PORT")
	os.Unsetenv("BITCOIN_RPC_USER")
	os.Unsetenv("BITCOIN_RPC_PASS")
	os.Unsetenv("BITCOIN_WALLET_NAME")
	os.Unsetenv("BITCOIN_ENABLE_INDEXER")
	os.Unsetenv("BITCOIN_INDEXER_LISTEN_ADDRESS")
	os.Unsetenv("BITCOIN_BRIDGE_ETH_RPC_URL")
	os.Unsetenv("BITCOIN_BRIDGE_CONTRACT_ADDRESS")
	os.Unsetenv("BITCOIN_BRIDGE_ETH_PRIV_KEY")
	os.Unsetenv("BITCOIN_BRIDGE_ABI")
	os.Unsetenv("BITCOIN_BRIDGE_GAS_LIMIT")
	os.Unsetenv("BITCOIN_BRIDGE_AA_SCA_REGISTRY")
	os.Unsetenv("BITCOIN_BRIDGE_AA_KERNEL_FACTORY")
	config, err := config.LoadBitcoinConfig("./testdata")
	require.NoError(t, err)
	require.Equal(t, "signet", config.NetworkName)
	require.Equal(t, "localhost", config.RPCHost)
	require.Equal(t, "8332", config.RPCPort)
	require.Equal(t, "b2node", config.RPCUser)
	require.Equal(t, "b2node", config.RPCPass)
	require.Equal(t, "b2node", config.WalletName)
	require.Equal(t, true, config.EnableIndexer)
	require.Equal(t, "tb1qfhhxljfajcppfhwa09uxwty5dz4xwfptnqmvtv", config.IndexerListenAddress)
	require.Equal(t, "localhost:8545", config.Bridge.EthRPCURL)
	require.Equal(t, "0xB457BF68D71a17Fa5030269Fb895e29e6cD2DFF2", config.Bridge.ContractAddress)
	require.Equal(t, "", config.Bridge.EthPrivKey)
	require.Equal(t, "abi.json", config.Bridge.ABI)
	require.Equal(t, uint64(3000), config.Bridge.GasLimit)
	require.Equal(t, "0xB457BF68D71a17Fa5030269Fb895e29e6cD2DFF3", config.Bridge.AASCARegistry)
	require.Equal(t, "0xB457BF68D71a17Fa5030269Fb895e29e6cD2DFF4", config.Bridge.AAKernelFactory)
}

func TestConfigEnv(t *testing.T) {
	os.Setenv("BITCOIN_NETWORK_NAME", "testnet")
	os.Setenv("BITCOIN_RPC_HOST", "127.0.0.1")
	os.Setenv("BITCOIN_RPC_PORT", "8888")
	os.Setenv("BITCOIN_RPC_USER", "abc")
	os.Setenv("BITCOIN_RPC_PASS", "abcd")
	os.Setenv("BITCOIN_WALLET_NAME", "b2node")
	os.Setenv("BITCOIN_ENABLE_INDEXER", "false")
	os.Setenv("BITCOIN_INDEXER_LISTEN_ADDRESS", "tb1qgm39cu009lyvq93afx47pp4h9wxq5x92lxxgnz")
	os.Setenv("BITCOIN_BRIDGE_ETH_RPC_URL", "127.0.0.1:8545")
	os.Setenv("BITCOIN_BRIDGE_CONTRACT_ADDRESS", "0xB457BF68D71a17Fa5030269Fb895e29e6cD2DF22")
	os.Setenv("BITCOIN_BRIDGE_ETH_PRIV_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	os.Setenv("BITCOIN_BRIDGE_ABI", "aaa.abi")
	os.Setenv("BITCOIN_BRIDGE_GAS_LIMIT", "23333")
	os.Setenv("BITCOIN_ENABLE_COMMITTER", "false")
	os.Setenv("BITCOIN_BRIDGE_AA_SCA_REGISTRY", "0xB457BF68D71a17Fa5030269Fb895e29e6cD2DF23")
	os.Setenv("BITCOIN_BRIDGE_AA_KERNEL_FACTORY", "0xB457BF68D71a17Fa5030269Fb895e29e6cD2DF24")
	os.Setenv("BITCOIN_EVM_ENABLE_LISTENER", "false")
	os.Setenv("BITCOIN_EVM_DEPOSIT", "0x01bee1bfa4116bd0440a1108ef6cb6a2f6eb9b611d8f53260aec20d39e84ee88")
	os.Setenv("BITCOIN_EVM_WITHDRAW", "0xda335c6ae73006d1145bdcf9a98bc76d789b653b13fe6200e6fc4c5dd54add85")

	config, err := config.LoadBitcoinConfig("./")
	require.NoError(t, err)
	require.Equal(t, "testnet", config.NetworkName)
	require.Equal(t, "127.0.0.1", config.RPCHost)
	require.Equal(t, "8888", config.RPCPort)
	require.Equal(t, "abc", config.RPCUser)
	require.Equal(t, "abcd", config.RPCPass)
	require.Equal(t, "b2node", config.WalletName)
	require.Equal(t, false, config.EnableIndexer)
	require.Equal(t, "tb1qgm39cu009lyvq93afx47pp4h9wxq5x92lxxgnz", config.IndexerListenAddress)
	require.Equal(t, "127.0.0.1:8545", config.Bridge.EthRPCURL)
	require.Equal(t, "0xB457BF68D71a17Fa5030269Fb895e29e6cD2DF22", config.Bridge.ContractAddress)
	require.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", config.Bridge.EthPrivKey)
	require.Equal(t, "aaa.abi", config.Bridge.ABI)
	require.Equal(t, uint64(23333), config.Bridge.GasLimit)
	require.Equal(t, "0xB457BF68D71a17Fa5030269Fb895e29e6cD2DF23", config.Bridge.AASCARegistry)
	require.Equal(t, "0xB457BF68D71a17Fa5030269Fb895e29e6cD2DF24", config.Bridge.AAKernelFactory)
}

func TestChainParams(t *testing.T) {
	testCases := []struct {
		network string
		params  *chaincfg.Params
	}{
		{
			network: "mainnet",
			params:  &chaincfg.MainNetParams,
		},
		{
			network: "testnet",
			params:  &chaincfg.TestNet3Params,
		},
		{
			network: "signet",
			params:  &chaincfg.SigNetParams,
		},
		{
			network: "simnet",
			params:  &chaincfg.SimNetParams,
		},
		{
			network: "regtest",
			params:  &chaincfg.RegressionNetParams,
		},
		{
			network: "invalid",
			params:  &chaincfg.TestNet3Params,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.network, func(t *testing.T) {
			result := config.ChainParams(tc.network)
			if result == nil || result != tc.params {
				t.Errorf("ChainParams(%s) = %v, expected %v", tc.network, result, tc.params)
			}
		})
	}
}
