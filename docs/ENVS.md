# Run-time environment variables

## Indexer configuration

| Variable                           | Type     | Description             | Compulsoriness | Default value | Example value                                            |
|------------------------------------|----------|-------------------------|----------------|---------------|----------------------------------------------------------|
| INDEXER_ROOT_DIR                   | `string` | Config file used dir    | -              |               |                                                          |
| INDEXER_LOG_LEVEL                  | `string` | Log level               | -              | `info`        | `info debug warn error panic fatal`                      |
| INDEXER_LOG_FORMAT                 | `string` | Log format              | -              | `console`     | ``                                                       |
| INDEXER_DATABASE_SOURCE            | `string` | database source         | Required       |               | `postgres://postgres:postgres@127.0.0.1:5432/b2-indexer` |
| INDEXER_DATABASE_MAX_IDLE_CONNS    | `number` | database max idle conns | -              | `10`          | `10`                                                     |
| INDEXER_DATABASE_MAX_OPEN_CONNS    | `number` | database max open conns | -              | `20`          | `20`                                                     |
| INDEXER_DATABASE_CONN_MAX_LIFETIME | `number` | database max lifetime   | -              | `3600`        | `3600`                                                   |

## Bitcoin configuration

| Variable                                    | Type       | Description                                           | Compulsoriness | Default value | Example value                        |
|---------------------------------------------|------------|-------------------------------------------------------|----------------|---------------|--------------------------------------|
| BITCOIN_NETWORK_NAME                        | `string`   | bitcoin network name                                  | Required       | testnet3      | `mainnet testnet3 regtest`           |
| BITCOIN_RPC_HOST                            | `string`   | bitcoin rpc host                                      | Required       |               | `127.0.0.1`                          |
| BITCOIN_RPC_PORT                            | `string`   | bitcoin rpc port                                      | Required       |               | `8332`                               |
| BITCOIN_RPC_USER                            | `string`   | bitcoin rpc user                                      | Required       |               |                                      |
| BITCOIN_RPC_PASS                            | `string`   | bitcoin rpc password                                  | Required       |               |                                      |
| BITCOIN_DISABLE_TLS                         | `bool`     | bitcoin disable tls                                   | Required       | `true`        |                                      |
| BITCOIN_WALLET_NAME                         | `string`   | bitcoin wallet name                                   | Required       |               |                                      |
| BITCOIN_ENABLE_INDEXER                      | `bool`     | enable indexer service                                | Required       |               | `false true`                         |
| BITCOIN_INDEXER_LISTEN_ADDRESS              | `string`   | indexer service listen btc address                    | Required       |               |                                      |
| BITCOIN_INDEXER_LISTEN_TARGET_CONFIRMATIONS | `number`   | target confirmations, adjust as needed                | -              | `1`           |                                      |
| BITCOIN_BRIDGE_ETH_RPC_URL                  | `string`   | bridge contract eth rpc url                           | Required       |               | `https://zkevm-rpc.bsquared.network` |
| BITCOIN_BRIDGE_ETH_PRIV_KEY                 | `string`   | bridge contract eth invoke priv key                   | Required       |               |                                      |
| BITCOIN_BRIDGE_CONTRACT_ADDRESS             | `string`   | bridge contract address                               | Required       |               |                                      |
| BITCOIN_BRIDGE_ABI                          | `string`   | bridge contract abi, if not set, will use default abi | -              |               |                                      |
| BITCOIN_BRIDGE_GAS_LIMIT                    | `number`   | bridge contract gas limit                             | Required       |               | `3000000`                            |
| BITCOIN_BRIDGE_AA_SCA_REGISTRY              | `string`   | aa sca registry                                       | Required       |               |                                      |
| BITCOIN_BRIDGE_AA_KERNEL_FACTORY            | `string`   | aa sca registry                                       | Required       |               |                                      |
| BITCOIN_BRIDGE_B2_NODE_API                  | `string`   | b2 node api                                           | Required       |               |                                      |
| BITCOIN_BRIDGE_B2_NODE_PRIV_KEY             | `string`   | b2 node priv key                                      | Required       |               |                                      |
| BITCOIN_BRIDGE_B2_NODE_GRPC_HOST            | `string`   | b2 node grpc host                                     | Required       |               |                                      |
| BITCOIN_BRIDGE_B2_NODE_GRPC_PORT            | `string`   | b2 node grpc port                                     | Required       |               |                                      |
| BITCOIN_BRIDGE_B2_NODE_DENOM                | `string`   | b2 node denom                                         | Required       | `aphoton`     |                                      |
| ENABLE_EPS                                  | `bool`     | enable eps service                                    | Required       |               | false true                           |
| EPS_URL                                     | `string`   | eps url                                               | Required       |               |                                      |
| EPS_AUTHORIZATION                           | `string`   | eps authorization                                     | Required       |               |                                      |
| BITCOIN_BRIDGE_DEPOSIT                      | `string`   | bridge deposit event hash                             | Required       |               |                                      |
| BITCOIN_BRIDGE_WITHDRAW                     | `string`   | bridge withdraw event hash                            | Required       |               |                                      |
| BITCOIN_BRIDGE_UNISAT_API_KEY               | `string`   | bridge withdraw unisat api_key                        | Required       |               |                                      |
| BITCOIN_BRIDGE_PUBLICKEYS                   | `[]string` | bridge withdraw sign publickey                        | Required       |               |                                      |
| BITCOIN_BRIDGE_TIME_INTERVAL                | `int64`    | bridge withdraw time interval                         | Required       |               |                                      |
| BITCOIN_BRIDGE_MULTISIG_NUM                 | `int`      | bridge withdraw multisig num                          | Required       |               |                                      |
| BITCOIN_BRIDGE_WITHDRAW_ENABLE_LISTENER     | `bool`     | enable bridge withdraw service                        | Required       |               | false true                           |
| ENABLE-ROLLUP-LISTENER                      | `bool`     | enable rollup indexer service                         | Required       |               | false true                           |

## http configuration

| Variable       | Type     | Description | Compulsoriness | Default value | Example value |
|----------------|----------|-------------|----------------|---------------|---------------|
| HTTP_PORT      | `string` | Http port   | -              | 8080          | -             |
| HTTP_GRPC_PORT | `string` | grpc port   | -              | 8081          | -             |
| HTTP_IP_WHITE_LIST | `string` | ip white list   | -              |           | -             |

# Service requirement environment variable

## b2-indexer

```
BITCOIN_NETWORK_NAME
BITCOIN_RPC_HOST
BITCOIN_RPC_PORT
BITCOIN_RPC_USER
BITCOIN_RPC_PASS
BITCOIN_ENABLE_INDEXER
BITCOIN_INDEXER_LISTEN_ADDRESS
BITCOIN_BRIDGE_ETH_RPC_URL
BITCOIN_BRIDGE_CONTRACT_ADDRESS
BITCOIN_BRIDGE_ETH_PRIV_KEY
BITCOIN_BRIDGE_GAS_LIMIT
BITCOIN_BRIDGE_GAS_PRICE_MULTIPLE
BITCOIN_BRIDGE_B2_EXPLORER_URL
BITCOIN_BRIDGE_AA_PARTICLE_RPC
BITCOIN_BRIDGE_AA_PARTICLE_PROJECT_ID
BITCOIN_BRIDGE_AA_PARTICLE_SERVER_KEY
BITCOIN_BRIDGE_AA_PARTICLE_CHAIN_ID
BITCOIN_BRIDGE_WITHDRAW_ENABLE_LISTENER
INDEXER_LOG_LEVEL
INDEXER_LOG_FORMAT
INDEXER_DATABASE_SOURCE
INDEXER_DATABASE_MAX_IDLE_CONNS
INDEXER_DATABASE_MAX_OPEN_CONNS
INDEXER_DATABASE_CONN_MAX_LIFETIME
BITCOIN_BRIDGE_DEPOSIT
BITCOIN_BRIDGE_WITHDRAW
ENABLE-ROLLUP-LISTENER
```

## b2-indexer-http

```
BITCOIN_INDEXER_LISTEN_ADDRESS
HTTP_IP_WHITE_LIST
INDEXER_LOG_LEVEL
INDEXER_LOG_FORMAT
INDEXER_DATABASE_SOURCE
INDEXER_DATABASE_MAX_IDLE_CONNS
INDEXER_DATABASE_MAX_OPEN_CONNS
INDEXER_DATABASE_CONN_MAX_LIFETIME
HTTP_PORT
HTTP_GRPC_PORT
```
