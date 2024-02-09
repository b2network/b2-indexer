package b2node_test

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/b2network/b2-indexer/internal/client"
	"github.com/b2network/b2-indexer/internal/logic/b2node"
	"github.com/b2network/b2-indexer/pkg/log"
	bridgeTypes "github.com/evmos/ethermint/x/bridge/types"
	"github.com/stretchr/testify/require"
)

var privateKeHex = ""

func TestLocalGetAccountInfo(t *testing.T) {
	address := "ethm12tufpdtvgpks2yv96dzkhwhtgr2zunaxwe0mn4"
	rpcUrl := "http://localhost:1317"
	grpcConn, err := client.GetClientConnection("127.0.0.1", client.WithClientPortOption(9090))
	require.NoError(t, err)
	nodeClient, err := b2node.NewNodeClient(privateKeHex, grpcConn, rpcUrl, "aphoton", log.NewNopLogger())
	require.NoError(t, err)
	addInfo, err := nodeClient.GetAccountInfo(address)
	require.NoError(t, err)
	t.Log(addInfo.CodeHash)
	t.Log(addInfo.BaseAccount.Sequence)
	t.Log(addInfo.BaseAccount.Address)
}

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
	rpcUrl := "http://127.0.0.1:1317"
	grpcConn, err := client.GetClientConnection("127.0.0.1", client.WithClientPortOption(9090))
	require.NoError(t, err)
	client, err := b2node.NewNodeClient(privateKeHex, grpcConn, rpcUrl, "aphoton", log.NewNopLogger())
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
