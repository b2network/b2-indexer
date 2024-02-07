package config

import (
	"os"
	"path"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/caarlos0/env/v6"
	"github.com/spf13/viper"
)

// Config is the global config.
type Config struct {
	// The root directory for all data.
	// This should be set in viper so it can unmarshal into this struct
	RootDir  string `mapstructure:"root-dir" env:"INDEXER_ROOT_DIR"`
	LogLevel string `mapstructure:"log-level" env:"INDEXER_LOG_LEVEL" envDefault:"info"`
	// "console","json"
	LogFormat               string `mapstructure:"log-format" env:"INDEXER_LOG_FORMAT" envDefault:"console"`
	DatabaseSource          string `mapstructure:"database-source" env:"INDEXER_DATABASE_SOURCE" envDefault:"postgres://postgres:postgres@127.0.0.1:5432/b2-indexer"`
	DatabaseMaxIdleConns    int    `mapstructure:"database-max-idle-conns"  env:"INDEXER_DATABASE_MAX_IDLE_CONNS" envDefault:"10"`
	DatabaseMaxOpenConns    int    `mapstructure:"database-max-open-conns" env:"INDEXER_DATABASE_MAX_OPEN_CONNS" envDefault:"20"`
	DatabaseConnMaxLifetime int    `mapstructure:"database-conn-max-lifetime" env:"INDEXER_DATABASE_CONN_MAX_LIFETIME" envDefault:"3600"`
}

// BitconConfig defines the bitcoin config
type BitconConfig struct {
	// NetworkName defines the bitcoin network name
	NetworkName string `mapstructure:"network-name" env:"BITCOIN_NETWORK_NAME"`
	// RPCHost defines the bitcoin rpc host
	RPCHost string `mapstructure:"rpc-host" env:"BITCOIN_RPC_HOST"`
	// RPCPort defines the bitcoin rpc port
	RPCPort string `mapstructure:"rpc-port" env:"BITCOIN_RPC_PORT"`
	// RPCUser defines the bitcoin rpc user
	RPCUser string `mapstructure:"rpc-user" env:"BITCOIN_RPC_USER"`
	// RPCPass defines the bitcoin rpc password
	RPCPass string `mapstructure:"rpc-pass" env:"BITCOIN_RPC_PASS"`
	// DisableTLS defines the bitcoin whether tls is required
	DisableTLS bool `mapstructure:"disable-tls" env:"BITCOIN_DISABLE_TLS" envDefault:"true"`
	// WalletName defines the bitcoin wallet name
	WalletName string `mapstructure:"wallet-name" env:"BITCOIN_WALLET_NAME"`
	// EnableIndexer defines whether to enable the indexer
	EnableIndexer bool `mapstructure:"enable-indexer" env:"BITCOIN_ENABLE_INDEXER"`
	// IndexerListenAddress defines the address to listen on
	IndexerListenAddress string `mapstructure:"indexer-listen-address" env:"BITCOIN_INDEXER_LISTEN_ADDRESS"`
	// Bridge defines the bridge config
	Bridge BridgeConfig `mapstructure:"bridge"`
	// Fee defines the bitcoin tx fee
	Fee int64 `mapstructure:"fee" env:"BITCOIN_FEE"`
	// Evm defines the evm config
	Evm EvmConfig `mapstructure:"evm"`
	Eps EpsConfig `mapstructure:"eps"`
}

type BridgeConfig struct {
	// EthRPCURL defines the ethereum rpc url, b2 rollup rpc
	EthRPCURL string `mapstructure:"eth-rpc-url" env:"BITCOIN_BRIDGE_ETH_RPC_URL"`
	// EthPrivKey defines the invoke ethereum private key
	EthPrivKey string `mapstructure:"eth-priv-key" env:"BITCOIN_BRIDGE_ETH_PRIV_KEY"`
	// ContractAddress defines the l1 -> l2 bridge contract address
	ContractAddress string `mapstructure:"contract-address" env:"BITCOIN_BRIDGE_CONTRACT_ADDRESS"`
	// ABI defines the l1 -> l2 bridge contract abi
	ABI string `mapstructure:"abi" env:"BITCOIN_BRIDGE_ABI"`
	// GasLimit defines the  contract gas limit
	GasLimit uint64 `mapstructure:"gas-limit" env:"BITCOIN_BRIDGE_GAS_LIMIT"`
	// AASCARegistry defines the  contract AASCARegistry address
	AASCARegistry string `mapstructure:"aa-sca-registry" env:"BITCOIN_BRIDGE_AA_SCA_REGISTRY"`
	// AAKernelFactory defines the  contract AAKernelFactory address
	AAKernelFactory string `mapstructure:"aa-kernel-factory" env:"BITCOIN_BRIDGE_AA_KERNEL_FACTORY"`
	// B2NodeAPI defines the b2 node api
	B2NodeAPI string `mapstructure:"b2-node-api" env:"BITCOIN_BRIDGE_B2_NODE_API"`
	// B2NodePrivKey defines the b2 node private key
	B2NodePrivKey string `mapstructure:"b2-node-priv-key" env:"BITCOIN_BRIDGE_B2_NODE_PRIV_KEY"`
	// B2NodeAddressPrefix defines the b2 node address prefix, eg: ethm
	B2NodeAddressPrefix string `mapstructure:"b2-node-address-prefix" env:"BITCOIN_BRIDGE_B2_NODE_ADDRESS_PREFIX" envDefault:"ethm"`
	// B2NodeChainID defines the b2 node chain id
	B2NodeChainID string `mapstructure:"b2-node-chain-id" env:"BITCOIN_BRIDGE_B2_NODE_CHAIN_ID"`
	// B2NodeGRPCHost defines the b2 node grpc host
	B2NodeGRPCHost string `mapstructure:"b2-node-grpc-host" env:"BITCOIN_BRIDGE_B2_NODE_GRPC_HOST"`
	// B2NodeGRPCPort defines the b2 node grpc port
	B2NodeGRPCPort uint32 `mapstructure:"b2-node-grpc-port" env:"BITCOIN_BRIDGE_B2_NODE_GRPC_PORT"`
	// B2NodeGasPrices defines the b2 node gas prices
	B2NodeGasPrices uint64 `mapstructure:"b2-node-gas-prices" env:"BITCOIN_BRIDGE_B2_NODE_GAS_PRICES" envDefault:"100000"`
	// B2NodeDenom defines the b2 node denom
	B2NodeDenom string `mapstructure:"b2-node-denom" env:"BITCOIN_BRIDGE_B2_NODE_DENOM" envDefault:"aphoton"`
}

type EvmConfig struct {
	// EnableListener defines whether to enable the listener
	EnableListener bool `mapstructure:"enable-listener" env:"BITCOIN_BRIDGE_ENABLE_LISTENER"`
	// Deposit defines the deposit event hash
	Deposit string `mapstructure:"deposit" env:"BITCOIN_BRIDGE_DEPOSIT"`
	// Withdraw defines the withdraw event hash
	Withdraw string `mapstructure:"withdraw" env:"BITCOIN_BRIDGE_WITHDRAW"`
}

type EpsConfig struct {
	EnableEps     bool   `mapstructure:"enable-eps" env:"ENABLE_EPS"`
	URL           string `mapstructure:"url" env:"EPS_URL"`
	Authorization string `mapstructure:"authorization" env:"EPS_AUTHORIZATION"`
}

const (
	BitcoinConfigFileName  = "bitcoin.toml"
	AppConfigFileName      = "indexer.toml"
	BitcoinConfigEnvPrefix = "BITCOIN"
	AppConfigEnvPrefix     = "APP"
)

func LoadConfig(homePath string) (*Config, error) {
	config := Config{}
	configFile := path.Join(homePath, AppConfigFileName)
	v := viper.New()
	v.SetConfigFile(configFile)

	v.SetEnvPrefix(AppConfigEnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// try load config from file
	err := v.ReadInConfig()
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// if err load config from env
		if err := env.Parse(&config); err != nil {
			return nil, err
		}
		return &config, nil
	}

	err = v.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func LoadBitcoinConfig(homePath string) (*BitconConfig, error) {
	config := BitconConfig{}
	configFile := path.Join(homePath, BitcoinConfigFileName)
	v := viper.New()
	v.SetConfigFile(configFile)

	v.SetEnvPrefix(BitcoinConfigEnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// try load config from file
	err := v.ReadInConfig()
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// if err load config from env
		if err := env.Parse(&config); err != nil {
			return nil, err
		}
		return &config, nil
	}

	err = v.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// ChainParams get chain params by network name
func ChainParams(network string) *chaincfg.Params {
	switch network {
	case chaincfg.MainNetParams.Name:
		return &chaincfg.MainNetParams
	case chaincfg.TestNet3Params.Name:
		return &chaincfg.TestNet3Params
	case chaincfg.SigNetParams.Name:
		return &chaincfg.SigNetParams
	case chaincfg.SimNetParams.Name:
		return &chaincfg.SimNetParams
	case chaincfg.RegressionNetParams.Name:
		return &chaincfg.RegressionNetParams
	default:
		return &chaincfg.TestNet3Params
	}
}

func DefaultConfig() *Config {
	return &Config{
		RootDir:  "",
		LogLevel: "info",
	}
}

func DefaultBitcoinConfig() *BitconConfig {
	return &BitconConfig{
		EnableIndexer: false,
		NetworkName:   "mainnet",
		RPCHost:       "127.0.0.1",
		RPCUser:       "",
		RPCPass:       "",
		RPCPort:       "8332",
	}
}
