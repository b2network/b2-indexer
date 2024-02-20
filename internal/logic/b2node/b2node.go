package b2node

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/b2network/b2-indexer/internal/types"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/b2network/b2-indexer/pkg/rpc"

	sdkmath "cosmossdk.io/math"

	clientTx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	eTypes "github.com/evmos/ethermint/types"
	bridgeTypes "github.com/evmos/ethermint/x/bridge/types"
	feeTypes "github.com/evmos/ethermint/x/feemarket/types"
	"google.golang.org/grpc"
)

const (
	DefaultBaseGasPrice = 10_000_000

	EventTypeCreateDeposit = "EventCreateDeposit"
	EventTypeSignWithdraw  = "EventSignWithdraw"
	EventTypeUpdateDeposit = "EventUpdateDeposit"
)

var txLock sync.Mutex

type NodeClient struct {
	PrivateKey    ethsecp256k1.PrivKey
	AddressPrefix string
	B2NodeAddress string
	ChainID       string
	GrpcConn      *grpc.ClientConn
	API           string
	CoinDenom     string
	log           log.Logger
}

func NewNodeClient(
	privateKeyHex string,
	grpcConn *grpc.ClientConn,
	api string,
	coinDenom string,
	logger log.Logger,
) (*NodeClient, error) {
	privatekeyBytes, err := hex.DecodeString(privateKeyHex)
	if nil != err {
		return nil, err
	}

	prefix, err := bech32Prefix(api)
	if err != nil {
		return nil, err
	}

	pk := ethsecp256k1.PrivKey{
		Key: privatekeyBytes,
	}

	b2NodeAddress, err := b2NodeAddress(pk, prefix)
	if err != nil {
		return nil, err
	}

	block, err := latestBlock(api)
	if err != nil {
		return nil, err
	}

	return &NodeClient{
		PrivateKey:    pk,
		AddressPrefix: prefix,
		ChainID:       block.Block.Header.ChainID,
		GrpcConn:      grpcConn,
		API:           api,
		CoinDenom:     coinDenom,
		log:           logger,
		B2NodeAddress: b2NodeAddress,
	}, nil
}

func (n *NodeClient) BridgeModuleEventType(eventType string) string {
	return "ethermint.bridge.v1." + eventType
}

func (n *NodeClient) GetGrpcConn() *grpc.ClientConn {
	return n.GrpcConn
}

func (n *NodeClient) GetAccountInfo(address string) (*eTypes.EthAccount, error) {
	authClient := authTypes.NewQueryClient(n.GetGrpcConn())
	ctx := context.Background()
	res, err := authClient.Account(ctx, &authTypes.QueryAccountRequest{Address: address})
	if err != nil {
		return nil, fmt.Errorf("[NodeClient] GetAccountInfo err: %s", err)
	}
	ethAccount := &eTypes.EthAccount{}
	err = ethAccount.Unmarshal(res.GetAccount().GetValue())
	if err != nil {
		return nil, fmt.Errorf("[NodeClient][ethAccount.Unmarshal] err: %s", err)
	}
	return ethAccount, nil
}

func (n *NodeClient) GetGasPrice() (uint64, error) {
	baseFee, err := n.BaseFee()
	if err != nil {
		return 0, err
	}
	baseFee *= 2
	return baseFee, nil
}

func (n *NodeClient) broadcastTx(ctx context.Context, msgs ...sdk.Msg) (*tx.BroadcastTxResponse, error) {
	txLock.Lock()
	defer txLock.Unlock()
	gasPrice, err := n.GetGasPrice()
	if err != nil {
		return nil, fmt.Errorf("[broadcastTx][GetEthGasPrice] err: %s", err)
	}
	txBytes, err := n.buildSimTx(gasPrice, msgs...)
	if err != nil {
		return nil, fmt.Errorf("[broadcastTx] err: %s", err)
	}
	txClient := tx.NewServiceClient(n.GetGrpcConn())
	res, err := txClient.BroadcastTx(ctx, &tx.BroadcastTxRequest{
		Mode:    tx.BroadcastMode_BROADCAST_MODE_BLOCK,
		TxBytes: txBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("[broadcastTx] err: %s", err)
	}
	return res, err
}

func (n *NodeClient) buildSimTx(gasPrice uint64, msgs ...sdk.Msg) ([]byte, error) {
	encCfg := simapp.MakeTestEncodingConfig()
	txBuilder := encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msgs...)
	if err != nil {
		return nil, fmt.Errorf("[BuildSimTx][SetMsgs] err: %s", err)
	}

	ethAccount, err := n.GetAccountInfo(n.B2NodeAddress)
	if nil != err {
		return nil, fmt.Errorf("[BuildSimTx][GetAccountInfo]err: %s", err)
	}
	signV2 := signing.SignatureV2{
		PubKey: n.PrivateKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode: encCfg.TxConfig.SignModeHandler().DefaultMode(),
		},
		Sequence: ethAccount.BaseAccount.Sequence,
	}
	err = txBuilder.SetSignatures(signV2)
	if err != nil {
		return nil, fmt.Errorf("[BuildSimTx][SetSignatures 1]err: %s", err)
	}

	txBuilder.SetGasLimit(DefaultBaseGasPrice)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.Coin{
		Denom:  n.CoinDenom,
		Amount: sdkmath.NewIntFromUint64(gasPrice * DefaultBaseGasPrice),
	}))

	signerData := xauthsigning.SignerData{
		ChainID:       n.ChainID,
		AccountNumber: ethAccount.BaseAccount.AccountNumber,
		Sequence:      ethAccount.BaseAccount.Sequence,
	}

	sigV2, err := clientTx.SignWithPrivKey(
		encCfg.TxConfig.SignModeHandler().DefaultMode(), signerData,
		txBuilder, &n.PrivateKey, encCfg.TxConfig, ethAccount.BaseAccount.Sequence)
	if err != nil {
		return nil, fmt.Errorf("[BuildSimTx][SignWithPrivKey] err: %s", err)
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return nil, fmt.Errorf("[BuildSimTx][SetSignatures 2] err: %s", err)
	}
	txBytes, err := encCfg.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("[BuildSimTx][GetTx] err: %s", err)
	}
	return txBytes, err
}

func (n *NodeClient) CreateDeposit(hash string, from string, to string, value int64) error {
	msg := bridgeTypes.NewMsgCreateDeposit(n.B2NodeAddress, hash, from, to, bridgeTypes.CoinType_COIN_TYPE_BTC, value, "")
	ctx := context.Background()
	msgResponse, err := n.broadcastTx(ctx, msg)
	if err != nil {
		return fmt.Errorf("[CreateDeposit] err: %s", err)
	}
	code := msgResponse.TxResponse.Code
	rawLog := msgResponse.TxResponse.RawLog
	if code != 0 {
		switch code {
		case bridgeTypes.ErrIndexExist.ABCICode():
			return bridgeTypes.ErrIndexExist
		case bridgeTypes.ErrNotCallerGroupMembers.ABCICode():
			return bridgeTypes.ErrNotCallerGroupMembers
		}
		n.log.Errorw("code", code)
		return fmt.Errorf("[CreateDeposit][msgResponse.TxResponse.Code] err: %s", rawLog)
	}
	return nil
}

func (n *NodeClient) UpdateDeposit(hash string, status bridgeTypes.DepositStatus, rollupTxHash string, fromAa string) error {
	msg := bridgeTypes.NewMsgUpdateDeposit(n.B2NodeAddress, hash, status, rollupTxHash, fromAa)
	ctx := context.Background()
	msgResponse, err := n.broadcastTx(ctx, msg)
	if err != nil {
		return fmt.Errorf("[UpdateDeposit] err: %s", err)
	}
	code := msgResponse.TxResponse.Code
	rawLog := msgResponse.TxResponse.RawLog
	if code != 0 {
		switch code {
		case bridgeTypes.ErrIndexNotExist.ABCICode():
			return bridgeTypes.ErrIndexNotExist
		case bridgeTypes.ErrInvalidStatus.ABCICode():
			return bridgeTypes.ErrInvalidStatus
		case bridgeTypes.ErrNotCallerGroupMembers.ABCICode():
			return bridgeTypes.ErrNotCallerGroupMembers
		}
		n.log.Errorw("code", code)
		return fmt.Errorf("[UpdateDeposit][msgResponse.TxResponse.Code] err: %s", rawLog)
	}
	return nil
}

func (n *NodeClient) QueryDeposit(hash string) (*bridgeTypes.Deposit, error) {
	queryClient := bridgeTypes.NewQueryClient(n.GetGrpcConn())
	res, err := queryClient.Deposit(context.Background(), &bridgeTypes.QueryGetDepositRequest{
		TxHash: hash,
	})
	if err != nil {
		switch err { //nolint
		case bridgeTypes.ErrIndexNotExist:
			return nil, bridgeTypes.ErrIndexNotExist
		}
		return nil, fmt.Errorf("[QueryDeposit] err: %s", err)
	}
	return &res.Deposit, nil
}

func (n *NodeClient) LatestBlock() (int64, error) {
	block, err := latestBlock(n.API)
	if err != nil {
		return 0, err
	}
	blockHeight, err := strconv.ParseInt(block.Block.Header.Height, 10, 64)
	if err != nil {
		return 0, err
	}
	return blockHeight, nil
}

func latestBlock(api string) (*B2NodeBlock, error) {
	latestBlockJSON, err := rpc.HTTPGet(fmt.Sprintf("%s/%s", api, "cosmos/base/tendermint/v1beta1/blocks/latest"))
	if err != nil {
		return nil, err
	}
	var block B2NodeBlock
	err = ParseJSONB2Node(latestBlockJSON, &block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

func (n *NodeClient) ParseBlockBridgeEvent(height int64, index int64) ([]*types.B2NodeTxParseResult, error) {
	txsJSON, err := rpc.HTTPGet(fmt.Sprintf("%s/%s%d&%s%s",
		n.API,
		"cosmos/tx/v1beta1/txs?events=tx.height=",
		height,
		"message.module=",
		"bridge",
	))
	if err != nil {
		return nil, err
	}
	var txs B2NodeTxs
	err = ParseJSONB2Node(txsJSON, &txs)
	if err != nil {
		return nil, err
	}
	b2NodeTxParseResult := make([]*types.B2NodeTxParseResult, 0)
	total, err := strconv.Atoi(txs.Total)
	if err != nil {
		return nil, err
	}
	if total > 0 {
		for txIndex, tx := range txs.TxResponses {
			if int64(txIndex) < index {
				continue
			}
			blockHeight, err := strconv.ParseInt(tx.Height, 10, 64)
			if err != nil {
				return nil, err
			}
			for _, log := range tx.Logs {
				for _, event := range log.Events {
					switch event.Type {
					case n.BridgeModuleEventType(EventTypeCreateDeposit):
						createDepositID := ""
						for _, attr := range event.Attributes {
							if attr.Key == "tx_hash" {
								createDepositID = strings.Trim(attr.Value, "\"")
							}
						}
						txResult := types.B2NodeTxParseResult{
							Height:              blockHeight,
							TxHash:              tx.Txhash,
							EventType:           EventTypeCreateDeposit,
							BridgeModuleTxIndex: txIndex,
							RawLog:              tx.RawLog,
							TxCode:              tx.Code,
							TxData:              tx.Data,
							BridgeEventID:       createDepositID,
						}
						b2NodeTxParseResult = append(b2NodeTxParseResult, &txResult)
					case n.BridgeModuleEventType(EventTypeSignWithdraw):
						withdrawTxID := ""
						for _, attr := range event.Attributes {
							if attr.Key == "tx_id" {
								withdrawTxID = strings.Trim(attr.Value, "\"")
							}
						}
						txResult := types.B2NodeTxParseResult{
							Height:              blockHeight,
							TxHash:              tx.Txhash,
							EventType:           EventTypeSignWithdraw,
							BridgeModuleTxIndex: txIndex,
							RawLog:              tx.RawLog,
							TxCode:              tx.Code,
							TxData:              tx.Data,
							BridgeEventID:       withdrawTxID,
						}
						b2NodeTxParseResult = append(b2NodeTxParseResult, &txResult)
					case n.BridgeModuleEventType(EventTypeUpdateDeposit):
						updateDepositID := ""
						for _, attr := range event.Attributes {
							if attr.Key == "tx_hash" {
								updateDepositID = strings.Trim(attr.Value, "\"")
							}
						}
						txResult := types.B2NodeTxParseResult{
							Height:              blockHeight,
							TxHash:              tx.Txhash,
							EventType:           EventTypeUpdateDeposit,
							BridgeModuleTxIndex: txIndex,
							RawLog:              tx.RawLog,
							TxCode:              tx.Code,
							TxData:              tx.Data,
							BridgeEventID:       updateDepositID,
						}
						b2NodeTxParseResult = append(b2NodeTxParseResult, &txResult)
					}
				}
			}
		}
	}
	return b2NodeTxParseResult, nil
}

func (n *NodeClient) BaseFee() (uint64, error) {
	queryClient := feeTypes.NewQueryClient(n.GetGrpcConn())
	res, err := queryClient.Params(context.Background(), &feeTypes.QueryParamsRequest{})
	if err != nil {
		return 0, err
	}
	return res.Params.BaseFee.Uint64(), nil
}

func bech32Prefix(api string) (string, error) {
	bech32PrefixJSON, err := rpc.HTTPGet(fmt.Sprintf("%s/%s", api, "cosmos/auth/v1beta1/bech32"))
	if err != nil {
		return "", err
	}
	var bech32Prefix Bech32Prefix
	err = ParseJSONB2Node(bech32PrefixJSON, &bech32Prefix)
	if err != nil {
		return "", err
	}
	return bech32Prefix.Bech32Prefix, nil
}

func b2NodeAddress(privateKey ethsecp256k1.PrivKey, prefix string) (string, error) {
	privKey, err := privateKey.ToECDSA()
	if err != nil {
		return "", err
	}
	ethAddress := crypto.PubkeyToAddress(privKey.PublicKey)
	bz, err := hex.DecodeString(ethAddress.Hex()[2:])
	if err != nil {
		return "", err
	}
	b2nodeAddress, err := bech32.ConvertAndEncode(prefix, bz)
	if err != nil {
		return "", err
	}
	return b2nodeAddress, nil
}

func (n *NodeClient) CreateWithdraw(txID string, txHashList []string, encodedData string) error {
	// private key -> adddress
	msg := bridgeTypes.NewMsgCreateWithdraw(n.B2NodeAddress, txID, txHashList, encodedData)
	ctx := context.Background()
	msgResponse, err := n.broadcastTx(ctx, msg)
	if err != nil {
		return fmt.Errorf("[CreateWithdraw] err: %s", err)
	}
	code := msgResponse.TxResponse.Code
	rawLog := msgResponse.TxResponse.RawLog
	if code != 0 {
		switch code {
		case bridgeTypes.ErrIndexExist.ABCICode():
			return bridgeTypes.ErrIndexExist
		case bridgeTypes.ErrNotCallerGroupMembers.ABCICode():
			return bridgeTypes.ErrNotCallerGroupMembers
		}
		n.log.Errorw("code", code)
		return fmt.Errorf("[CreateWithdraw][msgResponse.TxResponse.Code] err: %s", rawLog)
	}
	hexData := msgResponse.TxResponse.Data
	byteData, err := hex.DecodeString(hexData)
	if err != nil {
		return fmt.Errorf("[CreateWithdraw][hex.DecodeString] err: %s", err)
	}
	pbMsg := &sdk.TxMsgData{}
	err = pbMsg.Unmarshal(byteData)
	if err != nil {
		return fmt.Errorf("[CreateWithdraw][pbMsg.Unmarshal] err: %s", err)
	}
	return nil
}

func (n *NodeClient) QueryWithdraw(txID string) (*bridgeTypes.Withdraw, error) {
	queryClient := bridgeTypes.NewQueryClient(n.GetGrpcConn())
	res, err := queryClient.Withdraw(context.Background(), &bridgeTypes.QueryGetWithdrawRequest{
		TxId: txID,
	})
	if err != nil {
		if err == bridgeTypes.ErrIndexNotExist {
			return nil, bridgeTypes.ErrIndexNotExist
		}
		return nil, fmt.Errorf("[QueryWithdraw] err: %s", err)
	}
	return &res.Withdraw, nil
}

func (n *NodeClient) UpdateWithdraw(txID string, status bridgeTypes.WithdrawStatus) error {
	msg := bridgeTypes.NewMsgUpdateWithdraw(n.B2NodeAddress, txID, status)
	ctx := context.Background()
	msgResponse, err := n.broadcastTx(ctx, msg)
	if err != nil {
		return fmt.Errorf("[UpdateWithdraw] err: %s", err)
	}
	code := msgResponse.TxResponse.Code
	rawLog := msgResponse.TxResponse.RawLog
	if code != 0 {
		switch code {
		case bridgeTypes.ErrIndexNotExist.ABCICode():
			return bridgeTypes.ErrIndexNotExist
		case bridgeTypes.ErrInvalidStatus.ABCICode():
			return bridgeTypes.ErrInvalidStatus
		case bridgeTypes.ErrNotCallerGroupMembers.ABCICode():
			return bridgeTypes.ErrNotCallerGroupMembers
		}
		n.log.Errorw("code", code)
		return fmt.Errorf("[UpdateWithdraw][msgResponse.TxResponse.Code] err: %s", rawLog)
	}
	hexData := msgResponse.TxResponse.Data
	byteData, err := hex.DecodeString(hexData)
	if err != nil {
		return fmt.Errorf("[UpdateWithdraw][hex.DecodeString] err: %s", err)
	}
	pbMsg := &sdk.TxMsgData{}
	err = pbMsg.Unmarshal(byteData)
	if err != nil {
		return fmt.Errorf("[UpdateWithdraw][pbMsg.Unmarshal] err: %s", err)
	}
	return nil
}
