package server

import (
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/b2network/b2-indexer/internal/config"
	"github.com/b2network/b2-indexer/internal/logic/bitcoin"
	logger "github.com/b2network/b2-indexer/pkg/log"
	"github.com/btcsuite/btcd/rpcclient"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/spf13/cobra"
)

func Start(ctx *Context, cmd *cobra.Command) (err error) {
	home := ctx.cfg.RootDir

	bitcoinCfg, err := config.LoadBitcoinConfig(path.Join(home, "config"))
	if err != nil {
		logger.Errorw("failed to load bitcoin config", "error", err.Error())
		return err
	}

	if bitcoinCfg.EnableIndexer {
		logger.Infow("bitcoin index service starting!!!")
		bclient, err := rpcclient.New(&rpcclient.ConnConfig{
			Host:         bitcoinCfg.RPCHost + ":" + bitcoinCfg.RPCPort,
			User:         bitcoinCfg.RPCUser,
			Pass:         bitcoinCfg.RPCPass,
			HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
			DisableTLS:   true, // Bitcoin core does not provide TLS by default
		}, nil)
		if err != nil {
			logger.Errorw("failed to create bitcoin client", "error", err.Error())
			return err
		}
		defer func() {
			bclient.Shutdown()
		}()
		bitcoinParam := config.ChainParams(bitcoinCfg.NetworkName)

		bidxLogger := logger.New(logger.NewOptions())
		bidxer, err := bitcoin.NewBitcoinIndexer(bidxLogger, bclient, bitcoinParam, bitcoinCfg.IndexerListenAddress)
		if err != nil {
			logger.Errorw("failed to new bitcoin indexer indexer", "error", err.Error())
			return err
		}
		// check bitcoin core status, whether the request succeed
		_, err = bidxer.BlockChainInfo()
		if err != nil {
			logger.Errorw("failed to get bitcoin core status", "error", err.Error())
			return err
		}

		bridge, err := bitcoin.NewBridge(bitcoinCfg.Bridge, path.Join(home, "config"))
		if err != nil {
			logger.Errorw("failed to create bitcoin bridge", "error", err.Error())
			return err
		}

		// TODO: use PostgreSQL
		bitcoinidxDB, err := OpenBitcoinIndexerDB(home, dbm.GoLevelDBBackend)
		if err != nil {
			logger.Errorw("failed to open bitcoin indexer DB", "error", err.Error())
			return err
		}

		bindexerService := bitcoin.NewIndexerService(bidxer, bridge, bitcoinidxDB, bidxLogger)
		// bindexerService.SetLogger(logger)

		errCh := make(chan error)
		go func() {
			if err := bindexerService.OnStart(); err != nil {
				errCh <- err
			}
		}()

		select {
		case err := <-errCh:
			return err
		case <-time.After(5 * time.Second): // assume server started successfully
		}
	}
	// wait quit
	code := WaitForQuitSignals()
	logger.Infow("server stop!!!", "quit code", code)
	return nil
}

func OpenBitcoinIndexerDB(rootDir string, backendType dbm.BackendType) (dbm.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return dbm.NewDB("bitoinindexer", backendType, dataDir)
}

func WaitForQuitSignals() int {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGHUP)
	sig := <-sigs
	return int(sig.(syscall.Signal)) + 128
}
