# Run-time environment variables

## Indexer configuration

| Variable | Type| Description | Compulsoriness | Default value | Example value |
| --- | --- | --- | --- | --- | --- |
| INDEXER_ROOT_DIR | `string` | Config file used dir  | - |  |  |
| INDEXER_LOG_LEVEL | `string` | Log level | - | `info` | `info debug warn error panic fatal` |
| INDEXER_LOG_FORMAT | `string` | Log format | - | `console` | `` |
| INDEXER_DATABASE_SOURCE | `string` | database source | Required |  | `postgres://postgres:postgres@127.0.0.1:5432/b2-indexer` |
| INDEXER_DATABASE_MAX_IDLE_CONNS | `number` | database max idle conns| - | `10` | `10` |
| INDEXER_DATABASE_MAX_OPEN_CONNS | `number` | database max open conns| - | `20` | `20` |
| INDEXER_DATABASE_CONN_MAX_LIFETIME | `number` | database max lifetime| - | `3600` | `3600` |

## Bitcoin configuration

| Variable | Type| Description | Compulsoriness | Default value | Example value |
| --- | --- | --- | --- | --- | --- |
| BITCOIN_NETWORK_NAME | `string` | bitcoin network name | Required | testnet3 | `mainnet testnet3 regtest` |
| BITCOIN_RPC_HOST | `string` | bitcoin rpc host | Required |  | `127.0.0.1` |
| BITCOIN_RPC_PORT | `string` | bitcoin rpc port | Required |  | `8332` |
| BITCOIN_RPC_USER | `string` | bitcoin rpc user | Required |  |  |
| BITCOIN_RPC_PASS | `string` | bitcoin rpc password| Required |  |  |
| BITCOIN_DISABLE_TLS | `bool` | bitcoin disable tls| Required |`true`  |  |
| BITCOIN_WALLET_NAME | `string` | bitcoin wallet name| Required |  |  |
| BITCOIN_ENABLE_INDEXER | `bool` | enable indexer service | Required |  | `false true` |
| BITCOIN_INDEXER_LISTEN_ADDRESS | `string` | indexer service listen btc address | Required |  |  |
| BITCOIN_BRIDGE_ETH_RPC_URL | `string` | bridge contract eth rpc url | Required |  | `https://zkevm-rpc.bsquared.network` |
| BITCOIN_BRIDGE_ETH_PRIV_KEY | `string` | bridge contract eth invoke priv key | Required |  |  |
| BITCOIN_BRIDGE_CONTRACT_ADDRESS | `string` | bridge contract address| Required |  |  |
| BITCOIN_BRIDGE_ABI | `string` | bridge contract abi, if not set, will use default abi | - |  |  |
| BITCOIN_BRIDGE_GAS_LIMIT | `number` | bridge contract gas limit  | Required |  | `3000000` |
| BITCOIN_BRIDGE_AA_SCA_REGISTRY | `string` | aa sca registry | Required |  |  |
| BITCOIN_BRIDGE_AA_KERNEL_FACTORY | `string` | aa sca registry | Required |  |  |
