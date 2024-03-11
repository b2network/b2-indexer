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
	"strings"

	"github.com/b2network/b2-indexer/internal/config"
	b2types "github.com/b2network/b2-indexer/internal/types"
	"github.com/b2network/b2-indexer/pkg/aa"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/b2network/b2-indexer/pkg/particle"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	ErrBrdigeDepositTxHashExist                 = errors.New("non-repeatable processing")
	ErrBrdigeDepositContractInsufficientBalance = errors.New("insufficient balance")
	ErrBridgeWaitMinedStatus                    = errors.New("tx wait mined status failed")
	ErrBridgeFromGasInsufficient                = errors.New("gas required exceeds allowanc")
	ErrAAAddressNotFound                        = errors.New("address not found")
)

// Bridge bridge
// TODO: only L1 -> L2, More calls may be supported later
type Bridge struct {
	EthRPCURL        string
	EthPrivKey       *ecdsa.PrivateKey
	ContractAddress  common.Address
	ABI              string
	GasLimit         uint64
	GasPriceMultiple int64
	logger           log.Logger
	// particle aa
	particle     *particle.Particle
	bitcoinParam *chaincfg.Params
	// eoa transfer switch
	enableEoaTransfer bool
	// aa server
	AAPubKeyAPI string
}

// NewBridge new bridge
func NewBridge(bridgeCfg config.BridgeConfig, abiFileDir string, log log.Logger, bitcoinParam *chaincfg.Params) (*Bridge, error) {
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

	particle, err := particle.NewParticle(
		bridgeCfg.AAParticleRPC,
		bridgeCfg.AAParticleProjectID,
		bridgeCfg.AAParticleServerKey,
		bridgeCfg.AAParticleChainID)
	if err != nil {
		return nil, err
	}

	return &Bridge{
		EthRPCURL:         rpcURL.String(),
		ContractAddress:   common.HexToAddress(bridgeCfg.ContractAddress),
		EthPrivKey:        privateKey,
		ABI:               ABI,
		GasLimit:          bridgeCfg.GasLimit,
		logger:            log,
		particle:          particle,
		bitcoinParam:      bitcoinParam,
		enableEoaTransfer: bridgeCfg.EnableEoaTransfer,
		AAPubKeyAPI:       bridgeCfg.AAPubKeyAPI,
	}, nil
}

// Deposit to ethereum
func (b *Bridge) Deposit(
	hash string,
	bitcoinAddress b2types.BitcoinFrom,
	amount int64,
	oldTx *types.Transaction,
) (*types.Transaction, []byte, string, error) {
	if bitcoinAddress.Address == "" {
		return nil, nil, "", fmt.Errorf("bitcoin address is empty")
	}

	if hash == "" {
		return nil, nil, "", fmt.Errorf("tx id is empty")
	}

	ctx := context.Background()

	toAddress, err := b.BitcoinAddressToEthAddress(bitcoinAddress)
	if err != nil {
		return nil, nil, "", fmt.Errorf("btc address to eth address err:%w", err)
	}

	data, err := b.ABIPack(b.ABI, "depositV2", common.HexToHash(hash), common.HexToAddress(toAddress), new(big.Int).SetInt64(amount))
	if err != nil {
		return nil, nil, "", fmt.Errorf("abi pack err:%w", err)
	}

	if oldTx != nil {
		tx, err := b.retrySendTransaction(ctx, oldTx, b.EthPrivKey)
		if err != nil {
			return nil, nil, "", err
		}
		return tx, oldTx.Data(), oldTx.To().String(), nil
	}

	tx, err := b.sendTransaction(ctx, b.EthPrivKey, b.ContractAddress, data, new(big.Int).SetInt64(0))
	if err != nil {
		return nil, nil, "", err
	}

	return tx, data, toAddress, nil
}

// Transfer to ethereum
// TODO: temp handle, future remove
func (b *Bridge) Transfer(bitcoinAddress b2types.BitcoinFrom, amount int64) (*types.Transaction, error) {
	if bitcoinAddress.Address == "" {
		return nil, fmt.Errorf("bitcoin address is empty")
	}

	ctx := context.Background()

	toAddress, err := b.BitcoinAddressToEthAddress(bitcoinAddress)
	if err != nil {
		return nil, fmt.Errorf("btc address to eth address err:%w", err)
	}
	receipt, err := b.sendTransaction(ctx,
		b.EthPrivKey,
		common.HexToAddress(toAddress),
		nil,
		new(big.Int).Mul(new(big.Int).SetInt64(amount), new(big.Int).SetInt64(10000000000)),
	)
	if err != nil {
		return nil, fmt.Errorf("eth call err:%w", err)
	}

	return receipt, nil
}

func (b *Bridge) sendTransaction(ctx context.Context, fromPriv *ecdsa.PrivateKey,
	toAddress common.Address, data []byte, value *big.Int,
) (*types.Transaction, error) {
	client, err := ethclient.Dial(b.EthRPCURL)
	if err != nil {
		return nil, err
	}
	nonce, err := client.PendingNonceAt(ctx, crypto.PubkeyToAddress(fromPriv.PublicKey))
	if err != nil {
		return nil, err
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	if b.GasPriceMultiple != 0 {
		gasPrice.Mul(gasPrice, big.NewInt(b.GasPriceMultiple))
	}

	b.logger.Infof("gas price:", gasPrice.String())

	callMsg := ethereum.CallMsg{
		From:     crypto.PubkeyToAddress(fromPriv.PublicKey),
		To:       &toAddress,
		Value:    value,
		GasPrice: gasPrice,
	}
	if data != nil {
		callMsg.Data = data
	}

	// use eth_estimateGas only check deposit err
	gas, err := client.EstimateGas(ctx, callMsg)
	if err != nil {
		// Other errors may occur that need to be handled
		// The estimated gas cannot block the sending of a transaction
		b.logger.Errorw("estimate gas err", "error", err.Error())
		if strings.Contains(err.Error(), ErrBrdigeDepositTxHashExist.Error()) {
			return nil, ErrBrdigeDepositTxHashExist
		}

		if strings.Contains(err.Error(), ErrBrdigeDepositContractInsufficientBalance.Error()) {
			return nil, ErrBrdigeDepositContractInsufficientBalance
		}

		if strings.Contains(err.Error(), ErrBridgeFromGasInsufficient.Error()) {
			return nil, ErrBridgeFromGasInsufficient
		}

		// estimate gas err, return, try again
		return nil, err
	}
	gas *= 2
	legacyTx := types.LegacyTx{
		Nonce:    nonce,
		To:       &toAddress,
		Value:    value,
		Gas:      gas,
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

	return signedTx, nil
}
func (b *Bridge) retrySendTransaction(
	ctx context.Context,
	oldTx *types.Transaction,
	fromPriv *ecdsa.PrivateKey,
) (*types.Transaction, error) {
	client, err := ethclient.Dial(b.EthRPCURL)
	if err != nil {
		return nil, err
	}
	nonce := oldTx.Nonce()
	gasPrice := oldTx.GasPrice()

	if b.GasPriceMultiple != 0 {
		gasPrice.Mul(gasPrice, big.NewInt(b.GasPriceMultiple))
	}

	b.logger.Infof("gas price:", gasPrice.String())

	callMsg := ethereum.CallMsg{
		From:     crypto.PubkeyToAddress(fromPriv.PublicKey),
		To:       oldTx.To(),
		Value:    oldTx.Value(),
		GasPrice: gasPrice,
	}
	if oldTx.Data() != nil {
		callMsg.Data = oldTx.Data()
	}

	// use eth_estimateGas only check deposit err
	gas, err := client.EstimateGas(ctx, callMsg)
	if err != nil {
		// Other errors may occur that need to be handled
		// The estimated gas cannot block the sending of a transaction
		b.logger.Errorw("estimate gas err", "error", err.Error())
		if strings.Contains(err.Error(), ErrBrdigeDepositTxHashExist.Error()) {
			return nil, ErrBrdigeDepositTxHashExist
		}

		if strings.Contains(err.Error(), ErrBrdigeDepositContractInsufficientBalance.Error()) {
			return nil, ErrBrdigeDepositContractInsufficientBalance
		}

		if strings.Contains(err.Error(), ErrBridgeFromGasInsufficient.Error()) {
			return nil, ErrBridgeFromGasInsufficient
		}

		// estimate gas err, return, try again
		return nil, err
	}
	gas *= 2
	newlegacyTx := types.LegacyTx{
		Nonce:    nonce,
		To:       oldTx.To(),
		Value:    oldTx.Value(),
		Gas:      gas,
		GasPrice: gasPrice,
	}

	if oldTx.Data() != nil {
		newlegacyTx.Data = oldTx.Data()
	}

	tx := types.NewTx(&newlegacyTx)

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

	return signedTx, nil
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
func (b *Bridge) BitcoinAddressToEthAddress(bitcoinAddress b2types.BitcoinFrom) (string, error) {
	code, pubkey, err := aa.GetPubKey(b.AAPubKeyAPI, bitcoinAddress.Address)
	if err != nil {
		b.logger.Errorw("get pub key:", "pubkey", pubkey, "address", bitcoinAddress.Address)
		return "", err
	}
	if code == aa.AddressNotFoundErrCode {
		return "", ErrAAAddressNotFound
	}
	b.logger.Infow("get pub key:", "pubkey", pubkey, "address", bitcoinAddress.Address)
	aaBtcAccount, err := b.particle.AAGetBTCAccount([]string{pubkey})
	if err != nil {
		return "", err
	}

	if len(aaBtcAccount.Result) != 1 {
		b.logger.Errorw("AAGetBTCAccount", "result", aaBtcAccount)
		return "", fmt.Errorf("AAGetBTCAccount result not match")
	}
	b.logger.Infow("AAGetBTCAccount", "result", aaBtcAccount.Result[0])
	return aaBtcAccount.Result[0].SmartAccountAddress, nil
}

// WaitMined wait tx mined
func (b *Bridge) WaitMined(ctx context.Context, tx *types.Transaction, _ []byte) (*types.Receipt, error) {
	client, err := ethclient.Dial(b.EthRPCURL)
	if err != nil {
		return nil, err
	}

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return nil, err
	}

	if receipt.Status != 1 {
		b.logger.Errorw("wait mined status err", "error", ErrBridgeWaitMinedStatus, "receipt", receipt)
		return receipt, ErrBridgeWaitMinedStatus
	}
	return receipt, nil
}

func (b *Bridge) TransactionReceipt(hash string) (*types.Receipt, error) {
	client, err := ethclient.Dial(b.EthRPCURL)
	if err != nil {
		return nil, err
	}

	receipt, err := client.TransactionReceipt(context.Background(), common.HexToHash(hash))
	if err != nil {
		return nil, err
	}
	return receipt, nil
}
func (b *Bridge) TransactionByHash(hash string) (*types.Transaction, bool, error) {
	client, err := ethclient.Dial(b.EthRPCURL)
	if err != nil {
		return nil, false, err
	}

	tx, isPending, err := client.TransactionByHash(context.Background(), common.HexToHash(hash))
	if err != nil {
		return nil, false, err
	}
	return tx, isPending, nil
}
func (b *Bridge) EnableEoaTransfer() bool {
	return b.enableEoaTransfer
}
