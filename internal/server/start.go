package server

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/b2network/b2-indexer/internal/config"
	"github.com/b2network/b2-indexer/internal/logic/bitcoin"
	logger "github.com/b2network/b2-indexer/pkg/log"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func Start(ctx *Context, cmd *cobra.Command) (err error) {
	home := ctx.Config.RootDir
	bitcoinCfg := ctx.BitcoinConfig
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

		bidxLoggerOpt := logger.NewOptions()
		bidxLoggerOpt.Format = ctx.Config.LogFormat
		bidxLoggerOpt.Level = ctx.Config.LogLevel
		bidxLoggerOpt.EnableColor = true
		bidxLoggerOpt.Name = "[bitcoin-indexer]"
		bidxLogger := logger.New(bidxLoggerOpt)
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

		db, err := GetDBContextFromCmd(cmd)
		if err != nil {
			logger.Errorw("failed to get db context", "error", err.Error())
			return err
		}

		bindexerService := bitcoin.NewIndexerService(bidxer, db, bidxLogger)

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

		// start l1->l2 bridge service
		bridgeLoggerOpt := logger.NewOptions()
		bridgeLoggerOpt.Format = ctx.Config.LogFormat
		bridgeLoggerOpt.Level = ctx.Config.LogLevel
		bridgeLoggerOpt.EnableColor = true
		bridgeLoggerOpt.Name = "[bridge-deposit]"
		bridgeLogger := logger.New(bridgeLoggerOpt)
		bridgeService := bitcoin.NewBridgeDepositService(bridge, db, bridgeLogger)
		bridgeErrCh := make(chan error)
		go func() {
			if err := bridgeService.OnStart(); err != nil {
				bridgeErrCh <- err
			}
		}()

		select {
		case err := <-bridgeErrCh:
			return err
		case <-time.After(5 * time.Second): // assume server started successfully
		}
	}
	// wait quit
	code := WaitForQuitSignals()
	logger.Infow("server stop!!!", "quit code", code)
	return nil
}

func GetDBContextFromCmd(cmd *cobra.Command) (*gorm.DB, error) {
	if v := cmd.Context().Value(DBContextKey); v != nil {
		db := v.(*gorm.DB)
		return db, nil
	}
	return nil, fmt.Errorf("db context not set")
}

func WaitForQuitSignals() int {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGHUP)
	sig := <-sigs
	return int(sig.(syscall.Signal)) + 128
}
