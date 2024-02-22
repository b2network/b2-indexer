package config_test

import (
	"os"
	"strconv"
	"testing"

	"github.com/b2network/b2-indexer/internal/config"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/stretchr/testify/require"
)

func TestBitcoinConfig(t *testing.T) {
	// clean BITCOIN env set
	// This is because the value set by the environment variable affects viper reading file
	os.Unsetenv("BITCOIN_NETWORK_NAME")
	os.Unsetenv("BITCOIN_RPC_HOST")
	os.Unsetenv("BITCOIN_RPC_PORT")
	os.Unsetenv("BITCOIN_RPC_USER")
	os.Unsetenv("BITCOIN_RPC_PASS")
	os.Unsetenv("BITCOIN_DISABLE_TLS")
	os.Unsetenv("BITCOIN_WALLET_NAME")
	os.Unsetenv("BITCOIN_ENABLE_INDEXER")
	os.Unsetenv("BITCOIN_INDEXER_LISTEN_ADDRESS")
	os.Unsetenv("BITCOIN_INDEXER_LISTEN_TARGET_CONFIRMATIONS")
	os.Unsetenv("BITCOIN_BRIDGE_ETH_RPC_URL")
	os.Unsetenv("BITCOIN_BRIDGE_CONTRACT_ADDRESS")
	os.Unsetenv("BITCOIN_BRIDGE_ETH_PRIV_KEY")
	os.Unsetenv("BITCOIN_BRIDGE_ABI")
	os.Unsetenv("BITCOIN_BRIDGE_GAS_LIMIT")
	os.Unsetenv("BITCOIN_BRIDGE_AA_PARTICLE_RPC")
	os.Unsetenv("BITCOIN_BRIDGE_AA_PARTICLE_PROJECT_ID")
	os.Unsetenv("BITCOIN_BRIDGE_AA_PARTICLE_SERVER_KEY")
	os.Unsetenv("BITCOIN_BRIDGE_AA_PARTICLE_CHAIN_ID")
	os.Unsetenv("BITCOIN_BRIDGE_B2_NODE_API")
	os.Unsetenv("BITCOIN_BRIDGE_B2_NODE_PRIV_KEY")
	os.Unsetenv("BITCOIN_BRIDGE_B2_NODE_GRPC_HOST")
	os.Unsetenv("BITCOIN_BRIDGE_B2_NODE_GRPC_PORT")
	os.Unsetenv("BITCOIN_BRIDGE_B2_NODE_DENOM")
	os.Unsetenv("ENABLE_EPS")
	os.Unsetenv("EPS_URL")
	os.Unsetenv("EPS_AUTHORIZATION")
	os.Unsetenv("BITCOIN_BRIDGE_DEPOSIT")
	os.Unsetenv("BITCOIN_BRIDGE_WITHDRAW")
	os.Unsetenv("BITCOIN_BRIDGE_UNISAT_API_KEY")
	os.Unsetenv("BITCOIN_BRIDGE_PUBLICKEYS")
	os.Unsetenv("BITCOIN_BRIDGE_TIME_INTERVAL")
	os.Unsetenv("BITCOIN_BRIDGE_MULTISIG_NUM")
	config, err := config.LoadBitcoinConfig("./testdata")
	require.NoError(t, err)
	require.Equal(t, "signet", config.NetworkName)
	require.Equal(t, "localhost", config.RPCHost)
	require.Equal(t, "8332", config.RPCPort)
	require.Equal(t, "b2node", config.RPCUser)
	require.Equal(t, "b2node", config.RPCPass)
	require.Equal(t, true, config.DisableTLS)
	require.Equal(t, "b2node", config.WalletName)
	require.Equal(t, true, config.EnableIndexer)
	require.Equal(t, "tb1qfhhxljfajcppfhwa09uxwty5dz4xwfptnqmvtv", config.IndexerListenAddress)
	require.Equal(t, int64(1), config.IndexerListenTargetConfirmations)
	require.Equal(t, "localhost:8545", config.Bridge.EthRPCURL)
	require.Equal(t, "0xB457BF68D71a17Fa5030269Fb895e29e6cD2DFF2", config.Bridge.ContractAddress)
	require.Equal(t, "", config.Bridge.EthPrivKey)
	require.Equal(t, "abi.json", config.Bridge.ABI)
	require.Equal(t, uint64(3000), config.Bridge.GasLimit)
	require.Equal(t, "127.0.0.1:8080", config.Bridge.AAParticleRPC)
	require.Equal(t, "11111", config.Bridge.AAParticleProjectID)
	require.Equal(t, "22222", config.Bridge.AAParticleServerKey)
	require.Equal(t, 1102, config.Bridge.AAParticleChainID)
	require.Equal(t, "localhost:1317", config.Bridge.B2NodeAPI)
	require.Equal(t, "127.0.0.1", config.Bridge.B2NodeGRPCHost)
	require.Equal(t, uint32(9090), config.Bridge.B2NodeGRPCPort)
	require.Equal(t, "aphoton", config.Bridge.B2NodeDenom)
	require.Equal(t, "", config.Bridge.B2NodePrivKey)
	require.Equal(t, true, config.Eps.EnableEps)
	require.Equal(t, "127.0.0.1", config.Eps.URL)
	require.Equal(t, "", config.Eps.Authorization)
	require.Equal(t, "", config.Bridge.Deposit)
	require.Equal(t, "", config.Bridge.Withdraw)
	require.Equal(t, "", config.Bridge.UnisatAPIKey)
	require.Equal(t, int64(0), config.Bridge.TimeInterval)
	require.Equal(t, []string{""}, config.Bridge.PublicKeys)
	require.Equal(t, 0, config.Bridge.MultisigNum)
}

func TestBitcoinConfigEnv(t *testing.T) {
	os.Setenv("BITCOIN_NETWORK_NAME", "testnet")
	os.Setenv("BITCOIN_RPC_HOST", "127.0.0.1")
	os.Setenv("BITCOIN_RPC_PORT", "8888")
	os.Setenv("BITCOIN_RPC_USER", "abc")
	os.Setenv("BITCOIN_RPC_PASS", "abcd")
	os.Setenv("BITCOIN_DISABLE_TLS", "false")
	os.Setenv("BITCOIN_WALLET_NAME", "b2node")
	os.Setenv("BITCOIN_ENABLE_INDEXER", "false")
	os.Setenv("BITCOIN_INDEXER_LISTEN_ADDRESS", "tb1qgm39cu009lyvq93afx47pp4h9wxq5x92lxxgnz")
	os.Setenv("BITCOIN_INDEXER_LISTEN_TARGET_CONFIRMATIONS", "2")
	os.Setenv("BITCOIN_BRIDGE_ETH_RPC_URL", "127.0.0.1:8545")
	os.Setenv("BITCOIN_BRIDGE_CONTRACT_ADDRESS", "0xB457BF68D71a17Fa5030269Fb895e29e6cD2DF22")
	os.Setenv("BITCOIN_BRIDGE_ETH_PRIV_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	os.Setenv("BITCOIN_BRIDGE_ABI", "aaa.abi")
	os.Setenv("BITCOIN_BRIDGE_GAS_LIMIT", "23333")
	os.Setenv("BITCOIN_BRIDGE_AA_PARTICLE_RPC", "192.168.1.1/aa")
	os.Setenv("BITCOIN_BRIDGE_AA_PARTICLE_PROJECT_ID", "1234")
	os.Setenv("BITCOIN_BRIDGE_AA_PARTICLE_SERVER_KEY", "12345")
	os.Setenv("BITCOIN_BRIDGE_AA_PARTICLE_CHAIN_ID", "1101")
	os.Setenv("BITCOIN_BRIDGE_B2_NODE_API", "localhost:8888")
	os.Setenv("BITCOIN_BRIDGE_B2_NODE_PRIV_KEY", "")
	os.Setenv("BITCOIN_BRIDGE_B2_NODE_GRPC_HOST", "192.168.1.1")
	os.Setenv("BITCOIN_BRIDGE_B2_NODE_GRPC_PORT", "1234")
	os.Setenv("BITCOIN_BRIDGE_B2_NODE_DENOM", "aabbcc")
	os.Setenv("BITCOIN_EVM_ENABLE_LISTENER", "false")
	os.Setenv("BITCOIN_EVM_DEPOSIT", "0x01bee1bfa4116bd0440a1108ef6cb6a2f6eb9b611d8f53260aec20d39e84ee88")
	os.Setenv("BITCOIN_EVM_WITHDRAW", "0xda335c6ae73006d1145bdcf9a98bc76d789b653b13fe6200e6fc4c5dd54add85")
	os.Setenv("ENABLE_EPS", "true")
	os.Setenv("EPS_URL", "127.0.0.1")
	os.Setenv("EPS_AUTHORIZATION", "")
	os.Setenv("BITCOIN_BRIDGE_DEPOSIT", "")
	os.Setenv("BITCOIN_BRIDGE_WITHDRAW", "")
	os.Setenv("BITCOIN_BRIDGE_UNISAT_API_KEY", "")
	os.Setenv("BITCOIN_BRIDGE_TIME_INTERVAL", strconv.FormatInt(0, 10))
	os.Setenv("BITCOIN_BRIDGE_PUBLICKEYS", "")
	os.Setenv("BITCOIN_BRIDGE_MULTISIG_NUM", strconv.FormatInt(0, 10))

	config, err := config.LoadBitcoinConfig("./")
	require.NoError(t, err)
	require.Equal(t, "testnet", config.NetworkName)
	require.Equal(t, "127.0.0.1", config.RPCHost)
	require.Equal(t, "8888", config.RPCPort)
	require.Equal(t, "abc", config.RPCUser)
	require.Equal(t, "abcd", config.RPCPass)
	require.Equal(t, false, config.DisableTLS)
	require.Equal(t, "b2node", config.WalletName)
	require.Equal(t, false, config.EnableIndexer)
	require.Equal(t, "tb1qgm39cu009lyvq93afx47pp4h9wxq5x92lxxgnz", config.IndexerListenAddress)
	require.Equal(t, int64(2), config.IndexerListenTargetConfirmations)
	require.Equal(t, "127.0.0.1:8545", config.Bridge.EthRPCURL)
	require.Equal(t, "0xB457BF68D71a17Fa5030269Fb895e29e6cD2DF22", config.Bridge.ContractAddress)
	require.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", config.Bridge.EthPrivKey)
	require.Equal(t, "aaa.abi", config.Bridge.ABI)
	require.Equal(t, uint64(23333), config.Bridge.GasLimit)
	require.Equal(t, "192.168.1.1/aa", config.Bridge.AAParticleRPC)
	require.Equal(t, "1234", config.Bridge.AAParticleProjectID)
	require.Equal(t, "12345", config.Bridge.AAParticleServerKey)
	require.Equal(t, 1101, config.Bridge.AAParticleChainID)
	require.Equal(t, "localhost:8888", config.Bridge.B2NodeAPI)
	require.Equal(t, "192.168.1.1", config.Bridge.B2NodeGRPCHost)
	require.Equal(t, uint32(1234), config.Bridge.B2NodeGRPCPort)
	require.Equal(t, "aabbcc", config.Bridge.B2NodeDenom)
	require.Equal(t, "", config.Bridge.B2NodePrivKey)
	require.Equal(t, true, config.Eps.EnableEps)
	require.Equal(t, "127.0.0.1", config.Eps.URL)
	require.Equal(t, "", config.Eps.Authorization)
	require.Equal(t, "", config.Bridge.Deposit)
	require.Equal(t, "", config.Bridge.Withdraw)
	require.Equal(t, "", config.Bridge.UnisatAPIKey)
	require.Equal(t, int64(0), config.Bridge.TimeInterval)
	require.Equal(t, []string(nil), config.Bridge.PublicKeys)
	require.Equal(t, 0, config.Bridge.MultisigNum)
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

func TestConfig(t *testing.T) {
	os.Unsetenv("INDEXER_ROOT_DIR")
	os.Unsetenv("INDEXER_LOG_LEVEL")
	os.Unsetenv("INDEXER_LOG_FORMAT")
	os.Unsetenv("INDEXER_DATABASE_SOURCE")
	os.Unsetenv("INDEXER_DATABASE_MAX_IDLE_CONNS")
	os.Unsetenv("INDEXER_DATABASE_MAX_OPEN_CONNS")
	os.Unsetenv("INDEXER_DATABASE_CONN_MAX_LIFETIME")

	config, err := config.LoadConfig("./testdata")
	require.NoError(t, err)
	require.Equal(t, "/data/b2-indexer", config.RootDir)
	require.Equal(t, "info", config.LogLevel)
	require.Equal(t, "json", config.LogFormat)
	require.Equal(t, "postgres://postgres:postgres@127.0.0.2:5432/b2-indexer", config.DatabaseSource)
	require.Equal(t, 1, config.DatabaseMaxIdleConns)
	require.Equal(t, 2, config.DatabaseMaxOpenConns)
	require.Equal(t, 3600, config.DatabaseConnMaxLifetime)
}

func TestConfigEnv(t *testing.T) {
	os.Setenv("INDEXER_ROOT_DIR", "/data/test")
	os.Setenv("INDEXER_LOG_LEVEL", "debug")
	os.Setenv("INDEXER_LOG_FORMAT", "json")
	os.Setenv("INDEXER_DATABASE_SOURCE", "testtest")
	os.Setenv("INDEXER_DATABASE_MAX_IDLE_CONNS", "12")
	os.Setenv("INDEXER_DATABASE_MAX_OPEN_CONNS", "22")
	os.Setenv("INDEXER_DATABASE_CONN_MAX_LIFETIME", "2100")
	config, err := config.LoadConfig("./")
	require.NoError(t, err)
	require.Equal(t, "/data/test", config.RootDir)
	require.Equal(t, "debug", config.LogLevel)
	require.Equal(t, "json", config.LogFormat)
	require.Equal(t, "testtest", config.DatabaseSource)
	require.Equal(t, 12, config.DatabaseMaxIdleConns)
	require.Equal(t, 22, config.DatabaseMaxOpenConns)
	require.Equal(t, 2100, config.DatabaseConnMaxLifetime)
}
