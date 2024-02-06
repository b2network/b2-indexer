package b2node

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/b2network/b2-indexer/pkg/log"

	"github.com/b2network/b2-indexer/pkg/rpc"

	sdkmath "cosmossdk.io/math"

	clientTx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	eTypes "github.com/evmos/ethermint/types"
	bridgeTypes "github.com/evmos/ethermint/x/bridge/types"
	"google.golang.org/grpc"
)

const (
	DefaultBaseGasPrice = 10_000_000
)

type NodeClient struct {
	PrivateKey ethsecp256k1.PrivKey
	Address    string
	ChainID    string
	GrpcConn   *grpc.ClientConn
	RPCUrl     string
	log        log.Logger
}

type GasPriceRsp struct {
	ID      int64  `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  string `json:"result"`
}

func NewNodeClient(privateKeyHex string, chainID string, address string, grpcConn *grpc.ClientConn, rpcURL string, logger log.Logger) (*NodeClient, error) {
	privatekeyBytes, err := hex.DecodeString(privateKeyHex)
	if nil != err {
		return nil, err
	}
	return &NodeClient{
		PrivateKey: ethsecp256k1.PrivKey{
			Key: privatekeyBytes,
		},
		Address:  address,
		ChainID:  chainID,
		GrpcConn: grpcConn,
		RPCUrl:   rpcURL,
		log:      logger,
	}, nil
}

func (n *NodeClient) GetAccountInfo(address string) (*eTypes.EthAccount, error) {
	authClient := authTypes.NewQueryClient(n.GrpcConn)
	res, err := authClient.Account(context.Background(), &authTypes.QueryAccountRequest{Address: address})
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

func (n *NodeClient) GetEthGasPrice() (uint64, error) {
	gasPriceByte, err := rpc.HTTPPostJSON("", n.RPCUrl, `{"jsonrpc":"2.0","method":"eth_gasPrice","params":[],"id":73}`)
	if err != nil {
		return 0, fmt.Errorf("[GetEthGasPrice] err: %s", err)
	}
	var g GasPriceRsp
	if err := json.Unmarshal(gasPriceByte, &g); err != nil {
		return 0, fmt.Errorf("[GetEthGasPrice.json.Unmarshal] err: %s", err)
	}
	parseUint, err := strconv.ParseUint(g.Result, 0, 64)
	if err != nil {
		return 0, fmt.Errorf("[GetEthGasPrice.strconv.ParseUint] err: %s", err)
	}
	return parseUint, nil
}

func (n *NodeClient) broadcastTx(ctx context.Context, msgs ...sdk.Msg) (*tx.BroadcastTxResponse, error) {
	gasPrice, err := n.GetEthGasPrice()
	if err != nil {
		return nil, fmt.Errorf("[broadcastTx][GetEthGasPrice] err: %s", err)
	}
	txBytes, err := n.buildSimTx(gasPrice, msgs...)
	if err != nil {
		return nil, fmt.Errorf("[broadcastTx] err: %s", err)
	}
	txClient := tx.NewServiceClient(n.GrpcConn)
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
	ethAccount, err := n.GetAccountInfo(n.Address)
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
		Denom:  "aphoton",
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
	msg := bridgeTypes.NewMsgCreateDeposit(n.Address, hash, from, to, bridgeTypes.CoinType_COIN_TYPE_BTC, value, "")
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
	fmt.Println(msgResponse.TxResponse.TxHash)
	hexData := msgResponse.TxResponse.Data
	byteData, err := hex.DecodeString(hexData)
	if err != nil {
		return fmt.Errorf("[CreateDeposit][hex.DecodeString] err: %s", err)
	}
	pbMsg := &sdk.TxMsgData{}
	err = pbMsg.Unmarshal(byteData)
	if err != nil {
		return fmt.Errorf("[CreateDeposit][pbMsg.Unmarshal] err: %s", err)
	}
	return nil
}

func (n *NodeClient) UpdateDeposit(hash string, status bridgeTypes.DepositStatus, rollupTxHash string, fromAa string) error {
	msg := bridgeTypes.NewMsgUpdateDeposit(n.Address, hash, status, rollupTxHash, fromAa)
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
	fmt.Println(msgResponse.TxResponse.TxHash)
	hexData := msgResponse.TxResponse.Data
	byteData, err := hex.DecodeString(hexData)
	if err != nil {
		return fmt.Errorf("[UpdateDeposit][hex.DecodeString] err: %s", err)
	}
	pbMsg := &sdk.TxMsgData{}
	err = pbMsg.Unmarshal(byteData)
	if err != nil {
		return fmt.Errorf("[UpdateDeposit][pbMsg.Unmarshal] err: %s", err)
	}
	return nil
}

func (n *NodeClient) QueryDeposit(hash string) (*bridgeTypes.Deposit, error) {
	queryClient := bridgeTypes.NewQueryClient(n.GrpcConn)
	res, err := queryClient.Deposit(context.Background(), &bridgeTypes.QueryGetDepositRequest{
		TxHash: hash,
	})
	if err != nil {
		return nil, fmt.Errorf("[QueryDeposit] err: %s", err)
	}
	return &res.Deposit, nil
}
