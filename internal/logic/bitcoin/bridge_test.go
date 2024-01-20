package bitcoin_test

import (
	"context"
	"errors"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/b2network/b2-indexer/internal/config"
	"github.com/b2network/b2-indexer/internal/logic/bitcoin"
	"github.com/b2network/b2-indexer/pkg/log"
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

// TestLocalDeposit only test in local
func TestLocalDeposit(t *testing.T) {
	bridge := bridgeWithConfig(t)
	testCase := []struct {
		name string
		args []interface{}
		err  error
	}{
		{
			name: "success",
			args: []interface{}{
				"1c7fd15bd884524c8bc4c3b44e6839c013b4ad951972af454f926e0b6bdc570f",
				"tb1qjda2l5spwyv4ekwe9keddymzuxynea2m2kj0qy",
				int64(1234),
			},
			err: nil,
		},
		{
			name: "fail: address empty",
			args: []interface{}{
				"1c7fd15bd884524c8bc4c3b44e6839c013b4ad951972af454f926e0b6bdc570f",
				"",
				int64(1234),
			},
			err: errors.New("bitcoin address is empty"),
		},
		{
			name: "fail: tx id empty",
			args: []interface{}{
				"",
				"tb1qjda2l5spwyv4ekwe9keddymzuxynea2m2kj0qy",
				int64(1234),
			},
			err: errors.New("tx id is empty"),
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			hex, _, _, err := bridge.Deposit(tc.args[0].(string), tc.args[1].(string), tc.args[2].(int64))
			if err != nil {
				assert.Equal(t, tc.err, err)
			}
			t.Log(hex)
		})
	}
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
		ethAddress     string
	}{
		{
			name:           "success: Segwit (bech32)",
			bitcoinAddress: "tb1qjda2l5spwyv4ekwe9keddymzuxynea2m2kj0qy",
			ethAddress:     "0x2A9E233eE5d68fD70DE6C4b1d1Ffa29256e3ee9D",
		},
		{
			name:           "success: Segwit (bech32)",
			bitcoinAddress: "bc1qf60zw2gec5qg2mk4nyjl0slnytu0s0p28k9her",
			ethAddress:     "0x1b98017D9d6A9B62a2CFb2764D8012e28606BD49",
		},
		{
			name:           "success: Legacy",
			bitcoinAddress: "1KEFsFXrvuzMGd7Sdkwp7iTDcEcEv3GP1y",
			ethAddress:     "0x30f789e9C889A68180ef63F37cac923D89571394",
		},
		{
			name:           "success: Segwit",
			bitcoinAddress: "3Q4g8hgbwZLZ7vA6U1Xp1UsBs7NBnC7zKS",
			ethAddress:     "0x0aB97EA8eDff3e28867EAe9e13C02e5aA6214f59",
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			ethAddress, err := bridge.BitcoinAddressToEthAddress(tc.bitcoinAddress)
			require.NoError(t, err)
			assert.Equal(t, tc.ethAddress, ethAddress)
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

func TestLocalWaitMined(t *testing.T) {
	bridge := bridgeWithConfig(t)
	testCase := []struct {
		name string
		args []interface{}
		err  error
	}{
		{
			name: "success",
			args: []interface{}{
				"123",
				"tb1qjda2l5spwyv4ekwe9keddymzuxynea2m2kj0qy1",
				int64(1234),
			},
			err: nil,
		},
		{
			name: "fail: address empty",
			args: []interface{}{
				"1c7fd15bd884524c8bc4c3b44e6839c013b4ad951972af454f926e0b6bdc570f",
				"",
				int64(1234),
			},
			err: errors.New("bitcoin address is empty"),
		},
		{
			name: "fail: tx id empty",
			args: []interface{}{
				"",
				"tb1qjda2l5spwyv4ekwe9keddymzuxynea2m2kj0qy",
				int64(1234),
			},
			err: errors.New("tx id is empty"),
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			b2Tx, abiPackData, _, err := bridge.Deposit(tc.args[0].(string), tc.args[1].(string), tc.args[2].(int64))
			if err != nil {
				assert.Equal(t, tc.err, err)
			}
			_, err = bridge.WaitMined(context.Background(), b2Tx, abiPackData)
			if err != nil {
				t.Error()
			}

			t.Log(b2Tx)
		})
	}
}
