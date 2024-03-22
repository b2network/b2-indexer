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
	// IndexerListenTargetConfirmations defines the number of confirmations to listen on
	IndexerListenTargetConfirmations uint64 `mapstructure:"indexer-listen-target-confirmations" env:"BITCOIN_INDEXER_LISTEN_TARGET_CONFIRMATIONS" envDefault:"1"`
	// Bridge defines the bridge config
	Bridge BridgeConfig `mapstructure:"bridge"`
	Eps    EpsConfig    `mapstructure:"eps"`
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
	// if deposit invoke b2 failed(status != 1), Whether to allow invoke eoa trnasfer
	EnableEoaTransfer bool `mapstructure:"enable-eoa-transfer" env:"BITCOIN_BRIDGE_ENABLE_EOA_TRANSFER" envDefault:"true"`
	// AAPubKeyAPI get pubkey by btc address
	AAPubKeyAPI string `mapstructure:"aa-pubkey-api" env:"BITCOIN_BRIDGE_AA_PUBKEY_API"`
	// AAParticleRPC defines the particle api
	AAParticleRPC string `mapstructure:"aa-particle-rpc" env:"BITCOIN_BRIDGE_AA_PARTICLE_RPC"`
	// AAParticleProjectID defines the particle project id
	AAParticleProjectID string `mapstructure:"aa-particle-project-id" env:"BITCOIN_BRIDGE_AA_PARTICLE_PROJECT_ID"`
	// AAParticleServerKey defines the particle server key
	AAParticleServerKey string `mapstructure:"aa-particle-server-key" env:"BITCOIN_BRIDGE_AA_PARTICLE_SERVER_KEY"`
	// AAParticleChainID defines the particle chain id
	AAParticleChainID int `mapstructure:"aa-particle-chain-id" env:"BITCOIN_BRIDGE_AA_PARTICLE_CHAIN_ID"`
	// GasPriceMultiple defines the gas price multiple, TODO: temp fix, base gas_price * n
	GasPriceMultiple int64 `mapstructure:"gas-price-multiple" env:"BITCOIN_BRIDGE_GAS_PRICE_MULTIPLE" envDefault:"5"`
	// B2ExplorerURL defines the b2 explorer url, TODO: temp use explorer gas prices
	B2ExplorerURL string `mapstructure:"b2-explorer-url" env:"BITCOIN_BRIDGE_B2_EXPLORER_URL"`
	// EnableListener defines whether to enable the listener
	EnableWithdrawListener bool `mapstructure:"enable-withdraw-listener" env:"BITCOIN_BRIDGE_WITHDRAW_ENABLE_LISTENER"`
	// Deposit defines the deposit event hash
	Deposit string `mapstructure:"deposit" env:"BITCOIN_BRIDGE_DEPOSIT"`
	// Withdraw defines the withdraw event hash
	Withdraw string `mapstructure:"withdraw" env:"BITCOIN_BRIDGE_WITHDRAW"`
	// UnisatApiKey defines unisat api_key
	UnisatAPIKey string `mapstructure:"unisat-api-key" env:"BITCOIN_BRIDGE_UNISAT_API_KEY"`
	// PublicKeys defines signer publickey
	PublicKeys []string `mapstructure:"publickeys" env:"BITCOIN_BRIDGE_PUBLICKEYS"`
	// TimeInterval defines withdraw time interval
	TimeInterval int64 `mapstructure:"time-interval" env:"BITCOIN_BRIDGE_TIME_INTERVAL"`
	// MultisigNum defines withdraw multisig number
	MultisigNum int `mapstructure:"multisig-num" env:"BITCOIN_BRIDGE_MULTISIG_NUM"`
}

type EvmConfig struct {
	// EnableListener defines whether to enable the listener
	EnableWithdrawListener bool `mapstructure:"enable-withdraw-listener" env:"BITCOIN_BRIDGE_WITHDRAW_ENABLE_LISTENER"`
	// Deposit defines the deposit event hash
	Deposit string `mapstructure:"deposit" env:"BITCOIN_BRIDGE_DEPOSIT"`
	// Withdraw defines the withdraw event hash
	Withdraw string `mapstructure:"withdraw" env:"BITCOIN_BRIDGE_WITHDRAW"`
	// UnisatApiKey defines unisat api_key
	UnisatAPIKey string `mapstructure:"unisat-api-key" env:"BITCOIN_BRIDGE_UNISAT_API_KEY"`
	// PublicKeys defines signer publickey
	PublicKeys []string `mapstructure:"publickeys" env:"BITCOIN_BRIDGE_PUBLICKEYS"`
	// TimeInterval defines withdraw time interval
	TimeInterval int64 `mapstructure:"time-interval" env:"BITCOIN_BRIDGE_TIME_INTERVAL"`
	// MultisigNum defines withdraw multisig number
	MultisigNum int `mapstructure:"multisig-num" env:"BITCOIN_BRIDGE_MULTISIG_NUM"`
}

type EpsConfig struct {
	EnableEps     bool   `mapstructure:"enable-eps" env:"ENABLE_EPS"`
	URL           string `mapstructure:"url" env:"EPS_URL"`
	Authorization string `mapstructure:"authorization" env:"EPS_AUTHORIZATION"`
}

// HTTPConfig defines the http server config
type HTTPConfig struct {
	// port defines the http server port
	HTTPPort string `mapstructure:"http-port" env:"HTTP_PORT" envDefault:"9090"`
	// port defines the grpc server port
	GrpcPort string `mapstructure:"grpc-port" env:"HTTP_GRPC_PORT" envDefault:"9091"`
	// ipWhiteList defines the ip white list, Only those in the whitelist can be called
	IPWhiteList string `mapstructure:"ip-white-list" env:"HTTP_IP_WHITE_LIST"`
}

const (
	BitcoinConfigFileName  = "bitcoin.toml"
	AppConfigFileName      = "indexer.toml"
	HTTPConfigFileName     = "http.toml"
	BitcoinConfigEnvPrefix = "BITCOIN"
	AppConfigEnvPrefix     = "APP"
	HTTPConfigEnvPrefix    = "HTTP"
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

func LoadHTTPConfig(homePath string) (*HTTPConfig, error) {
	config := HTTPConfig{}
	configFile := path.Join(homePath, HTTPConfigFileName)
	v := viper.New()
	v.SetConfigFile(configFile)

	v.SetEnvPrefix(HTTPConfigEnvPrefix)
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
