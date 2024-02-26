package b2node_test

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/b2network/b2-indexer/internal/client"
	"github.com/b2network/b2-indexer/internal/config"
	"github.com/b2network/b2-indexer/internal/logic/b2node"
	"github.com/b2network/b2-indexer/pkg/log"
	bridgeTypes "github.com/evmos/ethermint/x/bridge/types"
	"github.com/stretchr/testify/require"
)

func TestLocalCreateDeposit(t *testing.T) {
	client := mockClient(t)
	txHash := generateTransactionHash(t)
	testCases := []struct {
		Name   string
		TxHash string
		From   string
		To     string
		Value  int64
		err    error
	}{
		{
			"success",
			txHash,
			"tb1qukxc3sy3s3k5n5z9cxt3xyywgcjmp2tzudlz2n",
			"3HctoF43JZCjAQrad1MqGtn5EsF57f5CCN",
			11,
			nil,
		},
		{
			"fail: tx hash exist",
			txHash,
			"tb1qukxc3sy3s3k5n5z9cxt3xyywgcjmp2tzudlz2n",
			"3HctoF43JZCjAQrad1MqGtn5EsF57f5CCN",
			11,
			bridgeTypes.ErrIndexExist,
		},
	}

	for _, tc := range testCases {
		err := client.CreateDeposit(tc.TxHash, tc.From, tc.To, tc.Value)
		if err != nil {
			require.Equal(t, tc.err, err)
		}
	}
}

func TestLocalQueryDeposit(t *testing.T) {
	client := mockClient(t)
	txHash := generateTransactionHash(t)
	from := "tb1qukxc3sy3s3k5n5z9cxt3xyywgcjmp2tzudlz2n"
	to := "3HctoF43JZCjAQrad1MqGtn5EsF57f5CCN"
	var value int64 = 11
	err := client.CreateDeposit(txHash, from, to, value)
	require.NoError(t, err)
	deposit, err := client.QueryDeposit(txHash)
	require.NoError(t, err)
	require.Equal(t, from, deposit.From)
	require.Equal(t, to, deposit.To)
	require.Equal(t, value, deposit.Value)
}

func TestLocalUpdateDeposit(t *testing.T) {
	client := mockClient(t)
	txHash := generateTransactionHash(t)
	rollupTxHash := generateTransactionHash(t)
	from := "tb1qukxc3sy3s3k5n5z9cxt3xyywgcjmp2tzudlz2n"
	fromAA := "0xffff"
	to := "3HctoF43JZCjAQrad1MqGtn5EsF57f5CCN"
	var value int64 = 11
	err := client.CreateDeposit(txHash, from, to, value)
	require.NoError(t, err)
	deposit, err := client.QueryDeposit(txHash)
	require.NoError(t, err)
	require.Equal(t, from, deposit.From)
	require.Equal(t, to, deposit.To)
	require.Equal(t, value, deposit.Value)
	require.Equal(t, bridgeTypes.DepositStatus_DEPOSIT_STATUS_PENDING, deposit.Status)

	// update
	err = client.UpdateDeposit(txHash, bridgeTypes.DepositStatus_DEPOSIT_STATUS_COMPLETED, rollupTxHash, fromAA)
	require.NoError(t, err)
	deposit, err = client.QueryDeposit(txHash)
	require.NoError(t, err)
	require.Equal(t, bridgeTypes.DepositStatus_DEPOSIT_STATUS_COMPLETED, deposit.Status)
	t.Fail()
}

func mockClient(t *testing.T) *b2node.NodeClient {
	config, err := config.LoadBitcoinConfig("")
	require.NoError(t, err)
	grpcConn, err := client.GetClientConnection(config.Bridge.B2NodeGRPCHost,
		client.WithClientPortOption(config.Bridge.B2NodeGRPCPort))
	require.NoError(t, err)
	client, err := b2node.NewNodeClient(config.Bridge.B2NodePrivKey,
		grpcConn,
		config.Bridge.B2NodeAPI,
		config.Bridge.B2NodeDenom,
		log.NewNopLogger())
	require.NoError(t, err)
	return client
}

func generateTransactionHash(t *testing.T) string {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	require.NoError(t, err)
	hash := sha256.Sum256(randomBytes)
	hashString := hex.EncodeToString(hash[:])
	return hashString
}

func TestLocalParseBlock(t *testing.T) {
	client := mockClient(t)
	_, err := client.ParseBlockBridgeEvent(8372, 0)
	require.NoError(t, err)
}

func TestLocalLatestBlock(t *testing.T) {
	client := mockClient(t)
	_, err := client.LatestBlock()
	require.NoError(t, err)
}

func TestLocalBaseFee(t *testing.T) {
	client := mockClient(t)
	_, err := client.BaseFee()
	require.NoError(t, err)
}
