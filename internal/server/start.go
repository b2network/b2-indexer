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

		db, err := GetDbContextFromCmd(cmd)
		if err != nil {
			logger.Errorw("failed to get db context", "error", err.Error())
			return err
		}

		bindexerService := bitcoin.NewIndexerService(bidxer, bridge, db, bidxLogger)
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

func GetDbContextFromCmd(cmd *cobra.Command) (*gorm.DB, error) {
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
