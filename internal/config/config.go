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
	RootDir  string `mapstructure:"home" env:"HOME"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
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
	// WalletName defines the bitcoin wallet name
	WalletName string `mapstructure:"wallet-name" env:"BITCOIN_WALLET_NAME"`
	// EnableIndexer defines whether to enable the indexer
	EnableIndexer bool `mapstructure:"enable-indexer" env:"BITCOIN_ENABLE_INDEXER"`
	// EnableCommitter defines whether to enable the committer
	EnableCommitter bool `mapstructure:"enable-committer" env:"BITCOIN_ENABLE_COMMITTER"`
	// IndexerListenAddress defines the address to listen on
	IndexerListenAddress string `mapstructure:"indexer-listen-address" env:"BITCOIN_INDEXER_LISTEN_ADDRESS"`
	// Bridge defines the bridge config
	Bridge BridgeConfig `mapstructure:"bridge"`
	// Fee defines the bitcoin tx fee
	Fee int64 `mapstructure:"fee" env:"BITCOIN_FEE"`
	// Evm defines the evm config
	Evm EvmConfig `mapstructure:"evm"`
}

type BridgeConfig struct {
	// EthRPCURL defines the ethereum rpc url
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
}

type EvmConfig struct {
	// EnableListener defines whether to enable the listener
	EnableListener bool `mapstructure:"enable-listener" env:"BITCOIN_BRIDGE_ENABLE_LISTENER"`
	// Deposit defines the deposit event hash
	Deposit string `mapstructure:"deposit" env:"BITCOIN_BRIDGE_DEPOSIT"`
	// Withdraw defines the withdraw event hash
	Withdraw string `mapstructure:"withdraw" env:"BITCOIN_BRIDGE_WITHDRAW"`
}

const (
	BitcoinConfigFileName  = "bitcoin.toml"
	AppConfigFileName      = "app.toml"
	BitcoinConfigEnvPrefix = "BITCOIN"
)

func LoadConfig(homePath string) (*Config, error) {
	config := Config{}
	configFile := path.Join(homePath, AppConfigFileName)
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

	err = viper.Unmarshal(&config)
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
