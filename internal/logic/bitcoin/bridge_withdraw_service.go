package bitcoin

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/b2network/b2-indexer/internal/config"
	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/internal/types"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	bridgeTypes "github.com/evmos/ethermint/x/bridge/types"
	"github.com/go-resty/resty/v2"
	"gorm.io/gorm"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	BridgeWithdrawServiceName = "BitcoinBridgeWithdrawService"
	WithdrawHandleTime        = 10
)

// BridgeWithdrawService indexes transactions for json-rpc service.
type BridgeWithdrawService struct {
	service.BaseService

	btcCli *rpcclient.Client
	ethCli *ethclient.Client
	config *config.BitconConfig
	db     *gorm.DB
	log    log.Logger
	b2node types.B2NODEBridge
}

// NewBridgeWithdrawService returns a new service instance.
func NewBridgeWithdrawService(
	btcCli *rpcclient.Client,
	ethCli *ethclient.Client,
	config *config.BitconConfig,
	db *gorm.DB,
	log log.Logger,
	b2node types.B2NODEBridge,
) *BridgeWithdrawService {
	is := &BridgeWithdrawService{btcCli: btcCli, ethCli: ethCli, config: config, db: db, log: log, b2node: b2node}
	is.BaseService = *service.NewBaseService(nil, BridgeWithdrawServiceName, is)
	return is
}

// OnStart implements service.Service by subscribing for new blocks
// and indexing them by events.
func (bis *BridgeWithdrawService) OnStart() error {
	if !bis.db.Migrator().HasTable(&model.Withdraw{}) {
		err := bis.db.AutoMigrate(&model.Withdraw{})
		if err != nil {
			bis.log.Errorw("BridgeWithdrawService create withdraw table", "error", err.Error())
			return err
		}
	}
	if !bis.db.Migrator().HasTable(&model.WithdrawTx{}) {
		err := bis.db.AutoMigrate(&model.WithdrawTx{})
		if err != nil {
			bis.log.Errorw("BridgeWithdrawService create withdrawTx table", "error", err.Error())
			return err
		}
	}

	go func() {
		// submit withdraw tx msg
		for {
			timeInterval := bis.config.Bridge.TimeInterval
			time.Sleep(time.Duration(timeInterval) * time.Second)
			var withdrawList []model.Withdraw
			err := bis.db.Model(&model.Withdraw{}).Where(fmt.Sprintf("%s = ?", model.Withdraw{}.Column().Status), model.BtcTxWithdrawPending).Find(&withdrawList).Error
			if err != nil {
				bis.log.Errorw("BridgeWithdrawService get blockNumber failed", "error", err)
				continue
			}
			var destAddressList []string
			var amounts []int64
			var ids []int64
			var b2TxHashes []string
			for _, v := range withdrawList {
				ids = append(ids, v.ID)
				destAddressList = append(destAddressList, v.BtcTo)
				amounts = append(amounts, v.BtcValue)
				b2TxHashes = append(b2TxHashes, v.B2TxHash)
			}
			b2TxHashesByte, err := json.Marshal(b2TxHashes)
			if err != nil {
				bis.log.Errorw("BridgeWithdrawService Marshal b2TxHashes err", "error", err, "id", ids)
				continue
			}
			txID, btcTx, err := bis.ConstructTx(destAddressList, amounts, b2TxHashesByte)
			if err != nil {
				if errors.Is(err, errors.New("no unspent tx")) {
					continue
				}
				bis.log.Errorw("BridgeWithdrawService transferToBtc failed: ", "err", err)
				continue
			}
			err = bis.db.Transaction(func(tx *gorm.DB) error {
				err = tx.Model(&model.Withdraw{}).Where("id in (?)", ids).Update(model.Withdraw{}.Column().Status, model.BtcTxWithdrawSubmitTxMsg).Error
				if err != nil {
					bis.log.Errorw("BridgeWithdrawService broadcast tx update db err", "error", err, "id", ids)
					return err
				}

				withdrawTxData := model.WithdrawTx{
					BtcTxID:    txID,
					BtcTx:      btcTx,
					B2TxHashes: string(b2TxHashesByte),
				}
				if err = bis.db.Create(&withdrawTxData).Error; err != nil {
					bis.log.Errorw("BridgeWithdrawService create withdrawTx err", "b2TxHashes", b2TxHashes)
					return err
				}

				// create witdraw record
				err = bis.b2node.CreateWithdraw(txID, b2TxHashes, btcTx)
				if err != nil {
					if !errors.Is(err, bridgeTypes.ErrIndexExist) {
						bis.Logger.Info("BridgeWithdrawService CreateWithdraw to b2node tx err", "error", err)
						return err
					}
				}

				bis.Logger.Info("BridgeWithdrawService broadcast tx success", "id", ids, "b2TxHashes", b2TxHashes)
				return nil
			})
		}
	}()

	go func() {
		for {
			// broadcast transaction
			time.Sleep(time.Duration(WithdrawHandleTime) * time.Second)
			var withdrawTxList []model.WithdrawTx
			err := bis.db.Model(&model.WithdrawTx{}).Where(fmt.Sprintf("%s = ?", model.Withdraw{}.Column().Status), model.BtcTxWithdrawSignatureCompleted).Find(&withdrawTxList).Error
			if err != nil {
				bis.log.Errorw("BridgeWithdrawService get broadcast tx failed", "error", err)
				continue
			}
			for _, v := range withdrawTxList {
				pack, err := psbt.NewFromRawBytes(strings.NewReader(v.BtcTx), true)
				if err != nil {
					bis.log.Errorw("BridgeWithdrawService get psbt tx err", "error", err)
					continue
				}
				tx := pack.UnsignedTx
				preTx := pack.Inputs

				withdrawInfo, err := bis.b2node.QueryWithdraw(v.BtcTxID)
				if err != nil {
					bis.Logger.Info("BridgeWithdrawService QueryWithdraw err", "error", err)
					continue
				}
				count := 0
				var signes [][]model.Sign
				for _, signHex := range withdrawInfo.Signatures {
					signByte, err := hex.DecodeString(signHex)
					if err != nil {
						bis.log.Errorw("BridgeWithdrawService DecodeString sign err", "error", err, "btc_tx_id", withdrawInfo.TxId)
						continue
					}
					var sign []model.Sign
					err = json.Unmarshal(signByte, &sign)
					if err != nil {
						bis.log.Errorw("BridgeWithdrawService Unmarshal sign err", "error", err, "btc_tx_id", withdrawInfo.TxId)
						continue
					}
					signes = append(signes, sign)
					count++
					if count == 2 {
						break
					}
				}
				for i, in := range tx.TxIn {
					sign01 := signes[0][i].Sign
					sign02 := signes[1][i].Sign
					in.Witness = wire.TxWitness{nil, sign01, sign02, preTx[i].WitnessUtxo.PkScript}
				}
				txHash, err := bis.BroadcastTx(tx)
				if err != nil {
					bis.log.Errorw("BridgeWithdrawService broadcast tx err", "error", err)
					continue
				}
				updateFields := map[string]interface{}{
					model.WithdrawTx{}.Column().BtcTxHash: txHash,
					model.WithdrawTx{}.Column().Status:    model.BtcTxWithdrawBroadcastSuccess,
				}
				err = bis.db.Model(&model.WithdrawTx{}).Where(fmt.Sprintf("%s = ?", model.WithdrawTx{}.Column().BtcTxID), v.BtcTxID).Updates(updateFields).Error
				if err != nil {
					bis.log.Errorw("BridgeWithdrawService broadcast tx update db err", "error", err, "id", v.ID)
					continue
				}
				bis.Logger.Info("BridgeWithdrawService broadcast tx success", "id", v.ID, "btcTxID", v.BtcTxID)
			}
		}
	}()

	go func() {
		for {
			time.Sleep(time.Duration(WithdrawHandleTime) * time.Second)
			// complete tx
			var withdrawTxList []model.WithdrawTx
			err := bis.db.Model(&model.WithdrawTx{}).Where(fmt.Sprintf("%s = ?", model.WithdrawTx{}.Column().Status), model.BtcTxWithdrawSuccess).Find(&withdrawTxList).Error
			if err != nil {
				bis.log.Errorw("BridgeWithdrawService get broadcast tx failed", "error", err)
				continue
			}
			for _, v := range withdrawTxList {
				err := bis.b2node.UpdateWithdraw(v.BtcTxID, bridgeTypes.WithdrawStatus_WITHDRAW_STATUS_COMPLETED)
				if err != nil {
					if !errors.Is(err, bridgeTypes.ErrIndexExist) {
						bis.Logger.Info("BridgeWithdrawService UpdateWithdraw err", "error", err, "txID", v.BtcTxID)
						continue
					}
				}
				err = bis.db.Model(&model.WithdrawTx{}).Where("id = ?", v.ID).Update(model.WithdrawTxColumns{}.Status, model.BtcTxWithdrawCompleted).Error
				if err != nil {
					bis.Logger.Info("BridgeWithdrawService Update WithdrawTx status err", "error", err, "txID", v.BtcTxID)
					continue
				}
			}
		}
	}()

	for {
		// listen withdraw
		time.Sleep(time.Duration(WithdrawHandleTime) * time.Second)
		var lastBlock uint64
		var withdraw model.Withdraw
		err := bis.db.Model(&model.Withdraw{}).Last(&withdraw).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				bis.log.Errorw("BridgeWithdrawService get blockNumber failed", "error", err)
				continue
			}
		} else {
			lastBlock = withdraw.B2BlockNumber
		}
		addresses := []common.Address{
			common.HexToAddress(bis.config.Bridge.ContractAddress),
		}
		topics := [][]common.Hash{
			{
				common.HexToHash(bis.config.Bridge.Deposit),
				common.HexToHash(bis.config.Bridge.Withdraw),
			},
		}
		for {
			latestBlock, err := bis.ethCli.BlockNumber(context.Background())
			if err != nil {
				bis.log.Errorw("BridgeWithdrawService HeaderByNumber is failed:", "err", err)
				continue
			}
			bis.Logger.Info("BridgeWithdrawService ethClient height", "height", latestBlock, "lastBlock", lastBlock)
			if latestBlock <= lastBlock {
				continue
			}
			for i := lastBlock + 1; i <= latestBlock; i++ {
				bis.log.Infow("BridgeWithdrawService get log height:", "height", i)
				query := ethereum.FilterQuery{
					FromBlock: big.NewInt(0).SetUint64(i),
					ToBlock:   big.NewInt(0).SetUint64(i),
					Topics:    topics,
					Addresses: addresses,
				}
				logs, err := bis.ethCli.FilterLogs(context.Background(), query)
				if err != nil {
					bis.log.Errorw("BridgeWithdrawService failed to fetch block", "height", i, "err", err)
					continue
				}

				for _, vlog := range logs {
					eventHash := common.BytesToHash(vlog.Topics[0].Bytes())
					if eventHash == common.HexToHash(bis.config.Bridge.Withdraw) {
						data := WithdrawEvent{
							FromAddress: TopicToAddress(vlog, 1),
							ToAddress:   DataToString(vlog, 0),
							Amount:      DataToBigInt(vlog, 1),
						}
						value, err := json.Marshal(&data)
						if err != nil {
							bis.log.Errorw("BridgeWithdrawService listener withdraw Marshal failed: ", "err", err)
							continue
						}
						bis.Logger.Info("BridgeWithdrawService listener withdraw event: ", "num", i, "withdraw", string(value))

						amount := DataToBigInt(vlog, 1)
						destAddrStr := DataToString(vlog, 0)
						withdrawData := model.Withdraw{
							BtcFrom:       bis.config.IndexerListenAddress,
							BtcTo:         destAddrStr,
							BtcValue:      amount.Int64(),
							B2BlockNumber: vlog.BlockNumber,
							B2BlockHash:   vlog.BlockHash.String(),
							B2TxHash:      vlog.TxHash.String(),
							B2TxIndex:     vlog.TxIndex,
							B2LogIndex:    vlog.Index,
						}
						if err = bis.db.Create(&withdrawData).Error; err != nil {
							bis.log.Errorw("BridgeWithdrawService create withdraw failed", "error", err, "withdraw", withdraw)
							return err
						}
					}
				}
				lastBlock = i
			}
		}
	}
}

func (bis *BridgeWithdrawService) BroadcastTx(tx *wire.MsgTx) (*chainhash.Hash, error) {
	mempoolURL := bis.GetMempoolURL()
	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return nil, err
	}
	fmt.Println(hex.EncodeToString(buf.Bytes()))
	fmt.Println(len(hex.EncodeToString(buf.Bytes())))
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/tx", mempoolURL), strings.NewReader(hex.EncodeToString(buf.Bytes())))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	txHash, err := chainhash.NewHashFromStr(string(body))
	if err != nil {
		return nil, err
	}
	return txHash, nil
}

func (bis *BridgeWithdrawService) GetUnspentList(address string, cursor int64) (int64, int64, []*model.UnspentOutput, error) {
	var total int64
	var satoshiTotal int64
	url := bis.GetUisatURL()
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", bis.config.Bridge.UnisatAPIKey)).
		Get(fmt.Sprintf("%s/v1/indexer/address/%s/utxo-data?cursor=%d", url, address, cursor))
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService client url err", "error", err)
		return total, satoshiTotal, nil, err
	}
	if resp.StatusCode() != 200 {
		bis.log.Errorw("BridgeWithdrawService client res err", "res", resp)
		return total, satoshiTotal, nil, errors.New(resp.Status())
	}
	var respData model.UnisatResponse
	err = json.Unmarshal(resp.Body(), &respData)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService unmarshal err", "error", err)
		return total, satoshiTotal, nil, err
	}
	if respData.Code != 0 {
		bis.log.Errorw("BridgeWithdrawService get utxo data err", "resp", respData)
		return total, satoshiTotal, nil, errors.New(respData.Msg)
	}
	utxoDataByte, err := json.Marshal(respData.Data)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService marshal utxo data err", "error", err)
		return total, satoshiTotal, nil, err
	}
	var utxoData model.UtxoData
	err = json.Unmarshal(utxoDataByte, &utxoData)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService unmarshal utxo data err", "error", err)
		return total, satoshiTotal, nil, err
	}
	unspentOutputs := make([]*model.UnspentOutput, 0)
	total = utxoData.Total
	for _, v := range utxoData.Utxo {
		txHash, err := chainhash.NewHashFromStr(v.Txid)
		if err != nil {
			return total, satoshiTotal, nil, err
		}
		unspentOutputs = append(unspentOutputs, &model.UnspentOutput{
			Outpoint: wire.NewOutPoint(txHash, uint32(v.Vout)),
			Output:   wire.NewTxOut(v.Satoshi, []byte(v.ScriptPk)),
		})
		satoshiTotal += v.Satoshi
	}
	return total, satoshiTotal, unspentOutputs, nil
}

func (bis *BridgeWithdrawService) GetUisatURL() string {
	networkName := bis.config.NetworkName
	switch networkName {
	case chaincfg.MainNetParams.Name:
		return "https://open-api.unisat.io"
	case chaincfg.TestNet3Params.Name, "testnet":
		return "https://open-api-testnet.unisat.io"
	}
	return ""
}

func (bis *BridgeWithdrawService) GetMempoolURL() string {
	networkName := bis.config.NetworkName
	switch networkName {
	case chaincfg.MainNetParams.Name:
		return "https://mempool.space/api"
	case chaincfg.TestNet3Params.Name, "testnet":
		return "https://mempool.space/testnet/api"
	case chaincfg.SigNetParams.Name:
		return "https://mempool.space/signet/api"
	}
	return ""
}

func (bis *BridgeWithdrawService) ConstructTx(destAddressList []string, amounts []int64, b2TxHashes []byte) (string, string, error) {
	sourceAddrStr := bis.config.IndexerListenAddress

	var defaultNet *chaincfg.Params
	networkName := bis.config.NetworkName
	defaultNet = config.ChainParams(networkName)

	// get sourceAddress UTXO
	sourceAddr, err := btcutil.DecodeAddress(sourceAddrStr, defaultNet)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService ConstructTx DecodeAddress err: ", "err", err)
		return "", "", err
	}

	total, satoshiTotal, unspentTxs, err := bis.GetUnspentList(sourceAddrStr, 0)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService GetUnspentList err: ", "err", err)
		return "", "", err
	}
	if len(unspentTxs) == 0 {
		return "", "", errors.New("no unspent tx")
	}
	var totalTransferAmount int64
	for _, v := range amounts {
		totalTransferAmount += v
	}
	if satoshiTotal <= totalTransferAmount {
		for i := 0; int64(i) < total/16; i++ {
			_, satoshiTotalTemp, unspentTxsTemp, err := bis.GetUnspentList(sourceAddrStr, int64(i))
			if err != nil {
				bis.log.Errorw("BridgeWithdrawService GetUnspentList err: ", "err", err)
				return "", "", err
			}
			if (satoshiTotal + satoshiTotalTemp) > totalTransferAmount {
				break
			}
			unspentTxs = append(unspentTxs, unspentTxsTemp...)
		}
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	for index, destAddress := range destAddressList {
		destAddr, err := btcutil.DecodeAddress(destAddress, defaultNet)
		if err != nil {
			bis.log.Errorw("BridgeWithdrawService transferToBtc DecodeAddress destAddress failed: ", "err", err)
			return "", "", err
		}
		destinationScript, err := txscript.PayToAddrScript(destAddr)
		if err != nil {
			bis.log.Errorw("BridgeWithdrawService transferToBtc PayToAddrScript destAddress failed: ", "err", err)
			return "", "", err
		}
		tx.AddTxOut(wire.NewTxOut(amounts[index], destinationScript))
	}

	var pInput psbt.PInput
	pInputArry := make([]psbt.PInput, 0)
	totalInputAmount := btcutil.Amount(0)
	for _, unspentTx := range unspentTxs {
		outpoint := wire.NewOutPoint(&unspentTx.Outpoint.Hash, unspentTx.Outpoint.Index)
		txIn := wire.NewTxIn(outpoint, nil, nil)
		tx.AddTxIn(txIn)

		multiSigScript, err := bis.GetMultiSigScript()
		if err != nil {
			bis.log.Errorw("BridgeWithdrawService ConstructTx GetMultiSigScript err", "error", err)
			return "", "", err
		}
		unspentTx.Output.PkScript = multiSigScript
		pInput.WitnessUtxo = unspentTx.Output
		pInput.WitnessScript = multiSigScript
		pInputArry = append(pInputArry, pInput)
		totalInputAmount += btcutil.Amount(unspentTx.Output.Value)
		if int64(totalInputAmount) > (totalTransferAmount + bis.config.Fee) {
			break
		}
	}

	changeAmount := int64(totalInputAmount) - bis.config.Fee - totalTransferAmount
	if changeAmount < 0 {
		return "", "", errors.New("insufficient balance")
	}
	changeScript, err := txscript.PayToAddrScript(sourceAddr)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService transferToBtc PayToAddrScript sourceAddr failed: ", "err", err)
		return "", "", err
	}
	tx.AddTxOut(wire.NewTxOut(changeAmount, changeScript))

	txCopy := tx.Copy()
	unsignedPsbt, err := psbt.NewFromUnsignedTx(txCopy)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService NewFromUnsignedTx err: ", "err", err)
		return "", "", err
	}
	unsignedPsbt.Inputs = pInputArry
	var unknown psbt.Unknown
	var unknowns []*psbt.Unknown
	unknown.Key = []byte("b2TxHashes")
	unknown.Value = b2TxHashes
	unknowns = append(unknowns, &unknown)
	unsignedPsbt.Unknowns = unknowns
	psbtData, err := unsignedPsbt.B64Encode()
	if err != nil {
		return "", "", err
	}
	return tx.TxHash().String(), psbtData, nil
}

func (bis *BridgeWithdrawService) GetMultiSigScript() ([]byte, error) {
	var defaultNet *chaincfg.Params
	networkName := bis.config.NetworkName
	defaultNet = config.ChainParams(networkName)

	pubKey1 := bis.config.Bridge.PublicKeys[0]
	pubKey2 := bis.config.Bridge.PublicKeys[1]
	pubKey3 := bis.config.Bridge.PublicKeys[2]
	privateKeyByte01, err := hex.DecodeString(pubKey1)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService GetMultiSigScript DecodeString err", "error", err)
		return nil, err
	}
	addressPubKey01, err := btcutil.NewAddressPubKey(privateKeyByte01, defaultNet)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService GetMultiSigScript NewAddressPubKey err", "error", err)
		return nil, err
	}
	privateKeyByte02, err := hex.DecodeString(pubKey2)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService GetMultiSigScript DecodeString err", "error", err)
		return nil, err
	}
	addressPubKey02, err := btcutil.NewAddressPubKey(privateKeyByte02, defaultNet)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService GetMultiSigScript NewAddressPubKey err", "error", err)
		return nil, err
	}
	privateKeyByte03, err := hex.DecodeString(pubKey3)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService GetMultiSigScript DecodeString err", "error", err)
		return nil, err
	}
	addressPubKey03, err := btcutil.NewAddressPubKey(privateKeyByte03, defaultNet)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService GetMultiSigScript NewAddressPubKey err", "error", err)
		return nil, err
	}
	multiSigScript, err := txscript.MultiSigScript([]*btcutil.AddressPubKey{addressPubKey01, addressPubKey02, addressPubKey03}, 2)
	if err != nil {
		bis.log.Errorw("BridgeWithdrawService get MultiSigScript err", "error", err)
		return nil, err
	}
	return multiSigScript, nil
}
