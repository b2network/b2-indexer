package bitcoin_test

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"os"
	"path"
	"testing"
	"time"

	"github.com/b2network/b2-indexer/internal/config"
	"github.com/b2network/b2-indexer/internal/logic/bitcoin"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/btcsuite/btcd/wire"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBridge(t *testing.T) {
	abiPath := path.Join("./testdata")

	ABI := ""

	abi, err := os.ReadFile(path.Join("./testdata", "abi.json"))
	if err != nil {
		// load default abi
		ABI = config.DefaultDepositAbi
	} else {
		ABI = string(abi)
	}

	privateKey, err := crypto.HexToECDSA("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}

	bridgeCfg := config.BridgeConfig{
		EthRPCURL:       "http://localhost:8545",
		ContractAddress: "0x123456789abcdef",
		EthPrivKey:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ABI:             "abi.json",
		GasLimit:        1000000,
		AASCARegistry:   "0x123456789abcdefgh",
		AAKernelFactory: "0x123456789abcdefg",
	}

	bridge, err := bitcoin.NewBridge(bridgeCfg, abiPath, log.NewNopLogger())
	assert.NoError(t, err)
	assert.NotNil(t, bridge)
	assert.Equal(t, bridgeCfg.EthRPCURL, bridge.EthRPCURL)
	assert.Equal(t, common.HexToAddress("0x123456789abcdef"), bridge.ContractAddress)
	assert.Equal(t, privateKey, bridge.EthPrivKey)
	assert.Equal(t, ABI, bridge.ABI)
	assert.Equal(t, common.HexToAddress("0x123456789abcdefgh"), bridge.AASCARegistry)
	assert.Equal(t, common.HexToAddress("0x123456789abcdefg"), bridge.AAKernelFactory)
}

// TestLocalTransfer only test in local
func TestLocalTransfer(t *testing.T) {
	bridge := bridgeWithConfig(t)
	testCase := []struct {
		name string
		args []interface{}
		err  error
	}{
		{
			name: "success",
			args: []interface{}{
				"tb1qjda2l5spwyv4ekwe9keddymzuxynea2m2kj0qy",
				int64(123456),
			},
			err: nil,
		},
		{
			name: "fail: address empty",
			args: []interface{}{
				"",
				int64(1234),
			},
			err: errors.New("bitcoin address is empty"),
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			hex, err := bridge.Transfer(tc.args[0].(string), tc.args[1].(int64))
			if err != nil {
				assert.Equal(t, tc.err, err)
			}
			t.Log(hex)
		})
	}
}

// TestLocalBitcoinAddressToEthAddress only test in local
func TestLocalBitcoinAddressToEthAddress(t *testing.T) {
	bridge := bridgeWithConfig(t)
	testCase := []struct {
		name           string
		bitcoinAddress string
	}{
		{
			name:           "success: Segwit (bech32)",
			bitcoinAddress: "tb1qjda2l5spwyv4ekwe9keddymzuxynea2m2kj0qy",
		},
		{
			name:           "success: Segwit (bech32)",
			bitcoinAddress: "bc1qf60zw2gec5qg2mk4nyjl0slnytu0s0p28k9her",
		},
		{
			name:           "success: Legacy",
			bitcoinAddress: "1KEFsFXrvuzMGd7Sdkwp7iTDcEcEv3GP1y",
		},
		{
			name:           "success: Segwit",
			bitcoinAddress: "3Q4g8hgbwZLZ7vA6U1Xp1UsBs7NBnC7zKS",
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			ethAddress, err := bridge.BitcoinAddressToEthAddress(tc.bitcoinAddress)
			require.NoError(t, err)
			if !common.IsHexAddress(ethAddress) {
				t.Errorf("bitcoinAddress: %s, ethAddress: %s", tc.bitcoinAddress, ethAddress)
			}
		})
	}
}

func TestABIPack(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		abiData, err := os.ReadFile(path.Join("./testdata", "abi.json"))
		if err != nil {
			t.Fatal(err)
		}
		expectedMethod := "deposit"
		expectedArgs := []interface{}{common.HexToAddress("0x12345678"), new(big.Int).SetInt64(1111)}
		expectedResult := []byte{
			71, 231, 239, 36, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 18, 52, 86, 120, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 4, 87,
		}

		// Create a mock bridge object
		b := &bitcoin.Bridge{}

		// Call the ABIPack method
		result, err := b.ABIPack(string(abiData), expectedMethod, expectedArgs...)
		// Check for errors
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Compare the result with the expected result
		require.Equal(t, result, expectedResult)
	})

	t.Run("Invalid ABI data", func(t *testing.T) {
		abiData := `{"inputs": [{"type": "address", "name": "to"}, {"type": "uint256", "name": "value"}`
		expectedError := errors.New("unexpected EOF")

		// Create a mock bridge object
		b := &bitcoin.Bridge{}

		// Call the ABIPack method
		_, err := b.ABIPack(abiData, "method", "arg1", "arg2")

		require.EqualError(t, err, expectedError.Error())
	})
}

func bridgeWithConfig(t *testing.T) *bitcoin.Bridge {
	config, err := config.LoadBitcoinConfig("")
	require.NoError(t, err)

	bridge, err := bitcoin.NewBridge(config.Bridge, "./testdata", log.NewNopLogger())
	require.NoError(t, err)
	return bridge
}

func TestLocalDepositWaitMined(t *testing.T) {
	bridge := bridgeWithConfig(t)
	uuid := randHash(t)
	address := "tb1qjda2l5spwyv4ekwe9keddymzuxynea2m2kj0qy1"
	value := 123
	bigValue := 11111111111111111

	// params check
	_, _, _, err := bridge.Deposit("", address, int64(value))
	if err != nil {
		assert.EqualError(t, errors.New("tx id is empty"), err.Error())
	}
	_, _, _, err = bridge.Deposit(uuid, "", int64(value))
	if err != nil {
		assert.EqualError(t, errors.New("bitcoin address is empty"), err.Error())
	}

	// normal
	b2Tx, _, _, err := bridge.Deposit(uuid, address, int64(value))
	if err != nil {
		assert.NoError(t, err)
	}
	_, err = bridge.WaitMined(context.Background(), b2Tx, nil)
	if err != nil {
		assert.NoError(t, err)
	}

	// uuid check
	_, _, _, err = bridge.Deposit(uuid, address, int64(value))
	if err != nil {
		assert.EqualError(t, bitcoin.ErrBrdigeDepositTxHashExist, err.Error())
	}

	// insufficient balance
	_, _, _, err = bridge.Deposit(randHash(t), address, int64(bigValue))
	if err != nil {
		assert.EqualError(t, bitcoin.ErrBrdigeDepositContractInsufficientBalance, err.Error())
	} else {
		t.Fatal("insufficient balance check failed")
	}

	// context timeout
	b2Tx2, _, _, err := bridge.Deposit(randHash(t), address, int64(value))
	if err != nil {
		assert.NoError(t, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Millisecond)
	defer cancel()
	_, err = bridge.WaitMined(ctx, b2Tx2, nil)
	if err != nil {
		assert.EqualError(t, context.DeadlineExceeded, err.Error())
	} else {
		t.Fatal("context deadline check failed")
	}
}

func randHash(t *testing.T) string {
	randomTx := wire.NewMsgTx(wire.TxVersion)
	randomData := make([]byte, 32)
	_, err := rand.Read(randomData)
	assert.NoError(t, err)
	randomTx.AddTxOut(&wire.TxOut{
		PkScript: randomData,
		Value:    0,
	})
	return randomTx.TxHash().String()
}
