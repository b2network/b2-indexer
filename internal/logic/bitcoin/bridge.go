package bitcoin

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"path"

	b2aa "github.com/b2network/b2-go-aa-utils"
	"github.com/b2network/b2-indexer/internal/config"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var ErrBrdigeDepositTxIDExist = errors.New("non-repeatable processing")

// Bridge bridge
// TODO: only L1 -> L2, More calls may be supported later
type Bridge struct {
	EthRPCURL       string
	EthPrivKey      *ecdsa.PrivateKey
	ContractAddress common.Address
	ABI             string
	GasLimit        uint64
	// AA contract address
	AASCARegistry   common.Address
	AAKernelFactory common.Address
}

// NewBridge new bridge
func NewBridge(bridgeCfg config.BridgeConfig, abiFileDir string) (*Bridge, error) {
	rpcURL, err := url.ParseRequestURI(bridgeCfg.EthRPCURL)
	if err != nil {
		return nil, err
	}

	privateKey, err := crypto.HexToECDSA(bridgeCfg.EthPrivKey)
	if err != nil {
		return nil, err
	}

	var ABI string

	abi, err := os.ReadFile(path.Join(abiFileDir, bridgeCfg.ABI))
	if err != nil {
		// load default abi
		ABI = config.DefaultDepositAbi
	} else {
		ABI = string(abi)
	}

	return &Bridge{
		EthRPCURL:       rpcURL.String(),
		ContractAddress: common.HexToAddress(bridgeCfg.ContractAddress),
		EthPrivKey:      privateKey,
		ABI:             ABI,
		GasLimit:        bridgeCfg.GasLimit,
		AASCARegistry:   common.HexToAddress(bridgeCfg.AASCARegistry),
		AAKernelFactory: common.HexToAddress(bridgeCfg.AAKernelFactory),
	}, nil
}

// Deposit to ethereum
func (b *Bridge) Deposit(hash string, bitcoinAddress string, amount int64) (string, error) {
	if bitcoinAddress == "" {
		return "", fmt.Errorf("bitcoin address is empty")
	}

	if hash == "" {
		return "", fmt.Errorf("tx id is empty")
	}

	ctx := context.Background()

	toAddress, err := b.BitcoinAddressToEthAddress(bitcoinAddress)
	if err != nil {
		return "", fmt.Errorf("btc address to eth address err:%w", err)
	}

	// TODO: hash check
	// txHash, err := chainhash.NewHashFromStr(hash)
	// if err != nil {
	// 	return "", err
	// }

	data, err := b.ABIPack(b.ABI, "deposit", common.HexToAddress(toAddress), new(big.Int).SetInt64(amount))
	if err != nil {
		return "", fmt.Errorf("abi pack err:%w", err)
	}

	receipt, err := b.sendTransaction(ctx, b.EthPrivKey, b.ContractAddress, data, 0)
	if err != nil {
		return "", fmt.Errorf("eth call err:%w", err)
	}

	if receipt.Status != 1 {
		receiptStr, err := receipt.MarshalJSON()
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("tx failed, receipt:%s", receiptStr)
	}
	return receipt.TxHash.String(), nil
}

// Transfer to ethereum
func (b *Bridge) Transfer(bitcoinAddress string, amount int64) (string, error) {
	if bitcoinAddress == "" {
		return "", fmt.Errorf("bitcoin address is empty")
	}

	ctx := context.Background()

	toAddress, err := b.BitcoinAddressToEthAddress(bitcoinAddress)
	if err != nil {
		return "", fmt.Errorf("btc address to eth address err:%w", err)
	}

	receipt, err := b.sendTransaction(ctx, b.EthPrivKey, common.HexToAddress(toAddress), nil, amount*10000000000)
	if err != nil {
		return "", fmt.Errorf("eth call err:%w", err)
	}

	if receipt.Status != 1 {
		receiptStr, err := receipt.MarshalJSON()
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("tx failed, receipt:%s", receiptStr)
	}
	return receipt.TxHash.String(), nil
}

func (b *Bridge) sendTransaction(ctx context.Context, fromPriv *ecdsa.PrivateKey,
	toAddress common.Address, data []byte, value int64,
) (*types.Receipt, error) {
	client, err := ethclient.Dial(b.EthRPCURL)
	if err != nil {
		return nil, err
	}

	publicKey := fromPriv.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}
	nonce, err := client.PendingNonceAt(ctx, crypto.PubkeyToAddress(*publicKeyECDSA))
	if err != nil {
		return nil, err
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	legacyTx := types.LegacyTx{
		Nonce:    nonce,
		To:       &toAddress,
		Value:    big.NewInt(value),
		Gas:      b.GasLimit,
		GasPrice: gasPrice,
	}

	if data != nil {
		legacyTx.Data = data
	}

	tx := types.NewTx(&legacyTx)

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	// sign tx
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), fromPriv)
	if err != nil {
		return nil, err
	}

	// send tx
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, err
	}

	// wait tx confirm
	return bind.WaitMined(ctx, client, signedTx)
}

// ABIPack the given method name to conform the ABI. Method call's data
func (b *Bridge) ABIPack(abiData string, method string, args ...interface{}) ([]byte, error) {
	contractAbi, err := abi.JSON(bytes.NewReader([]byte(abiData)))
	if err != nil {
		return nil, err
	}
	return contractAbi.Pack(method, args...)
}

// BitcoinAddressToEthAddress bitcoin address to eth address
func (b *Bridge) BitcoinAddressToEthAddress(bitcoinAddress string) (string, error) {
	client, err := ethclient.Dial(b.EthRPCURL)
	if err != nil {
		return "", err
	}

	targetEthAddress, err := b2aa.GetSCAAddress(client, b.AASCARegistry, b.AAKernelFactory, bitcoinAddress)
	if err != nil {
		return "", err
	}
	return targetEthAddress.String(), nil
}
