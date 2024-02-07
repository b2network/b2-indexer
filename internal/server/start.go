package server

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/b2network/b2-indexer/internal/client"
	"github.com/b2network/b2-indexer/internal/config"
	"github.com/b2network/b2-indexer/internal/logic/b2node"
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
			HTTPPostMode: true,                  // Bitcoin core only supports HTTP POST mode
			DisableTLS:   bitcoinCfg.DisableTLS, // Bitcoin core does not provide TLS by default
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

		db, err := GetDBContextFromCmd(cmd)
		if err != nil {
			logger.Errorw("failed to get db context", "error", err.Error())
			return err
		}

		bindexerService := bitcoin.NewIndexerService(bidxer, db, bidxLogger)

		errCh := make(chan error)
		go func() {
			if err := bindexerService.Start(); err != nil {
				errCh <- err
			}
		}()

		select {
		case err := <-errCh:
			return err
		case <-time.After(5 * time.Second): // assume server started successfully
		}

		// start b2node indexer service
		b2nodeIndexerLoggerOpt := logger.NewOptions()
		b2nodeIndexerLoggerOpt.Format = ctx.Config.LogFormat
		b2nodeIndexerLoggerOpt.Level = ctx.Config.LogLevel
		b2nodeIndexerLoggerOpt.EnableColor = true
		b2nodeIndexerLoggerOpt.Name = "[b2node-indexer]"
		b2nodeIndexerLogger := logger.New(b2nodeIndexerLoggerOpt)
		b2nodeIndexergrpcConn, err := client.GetClientConnection(bitcoinCfg.Bridge.B2NodeGRPCHost, client.WithClientPortOption(bitcoinCfg.Bridge.B2NodeGRPCPort))
		if err != nil {
			return err
		}
		b2nodeIndexBridge, err := b2node.NewNodeClient(
			bitcoinCfg.Bridge.B2NodePrivKey,
			bitcoinCfg.Bridge.B2NodeChainID,
			bitcoinCfg.Bridge.B2NodeAddressPrefix,
			b2nodeIndexergrpcConn,
			bitcoinCfg.Bridge.B2NodeAPI,
			bitcoinCfg.Bridge.B2NodeDenom,
			bitcoinCfg.Bridge.B2NodeGasPrices,
			b2nodeIndexerLogger,
		)
		if err != nil {
			logger.Errorw("failed to create b2node", "error", err.Error())
			return err
		}

		b2nodeDB, err := GetDBContextFromCmd(cmd)
		if err != nil {
			logger.Errorw("failed to get db context", "error", err.Error())
			return err
		}

		b2IndexerService := b2node.NewB2NodeIndexerService(b2nodeIndexBridge, b2nodeDB, b2nodeIndexerLogger)

		b2IndexerErrCh := make(chan error)
		go func() {
			if err := b2IndexerService.Start(); err != nil {
				b2IndexerErrCh <- err
			}
		}()

		select {
		case err := <-b2IndexerErrCh:
			return err
		case <-time.After(5 * time.Second): // assume server started successfully
		}

		// start l1->b2node
		bridgeB2NodeLoggerOpt := logger.NewOptions()
		bridgeB2NodeLoggerOpt.Format = ctx.Config.LogFormat
		bridgeB2NodeLoggerOpt.Level = ctx.Config.LogLevel
		bridgeB2NodeLoggerOpt.EnableColor = true
		bridgeB2NodeLoggerOpt.Name = "[bridge-deposit-b2node]"
		bridgeB2NodeLogger := logger.New(bridgeB2NodeLoggerOpt)
		b2grpcConn, err := client.GetClientConnection(bitcoinCfg.Bridge.B2NodeGRPCHost, client.WithClientPortOption(bitcoinCfg.Bridge.B2NodeGRPCPort))
		if err != nil {
			return err
		}
		bridgeB2node, err := b2node.NewNodeClient(
			bitcoinCfg.Bridge.B2NodePrivKey,
			bitcoinCfg.Bridge.B2NodeChainID,
			bitcoinCfg.Bridge.B2NodeAddressPrefix,
			b2grpcConn,
			bitcoinCfg.Bridge.B2NodeAPI,
			bitcoinCfg.Bridge.B2NodeDenom,
			bitcoinCfg.Bridge.B2NodeGasPrices,
			bridgeB2NodeLogger,
		)
		if err != nil {
			logger.Errorw("failed to create b2node", "error", err.Error())
			return err
		}

		bridgeB2NodeService := bitcoin.NewBridgeDepositB2NodeService(bridgeB2node, db, bridgeB2NodeLogger)
		bridgeB2NodeErrCh := make(chan error)
		go func() {
			if err := bridgeB2NodeService.OnStart(); err != nil {
				bridgeB2NodeErrCh <- err
			}
		}()

		select {
		case err := <-bridgeB2NodeErrCh:
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

		bridge, err := bitcoin.NewBridge(bitcoinCfg.Bridge, path.Join(home, "config"), bridgeLogger)
		if err != nil {
			logger.Errorw("failed to create bitcoin bridge", "error", err.Error())
			return err
		}

		bridgeService := bitcoin.NewBridgeDepositService(bridge, db, bridgeLogger)
		bridgeErrCh := make(chan error)
		go func() {
			if err := bridgeService.Start(); err != nil {
				bridgeErrCh <- err
			}
		}()

		select {
		case err := <-bridgeErrCh:
			return err
		case <-time.After(5 * time.Second): // assume server started successfully
		}
	}

	if bitcoinCfg.Eps.EnableEps {
		epsLoggerOpt := logger.NewOptions()
		epsLoggerOpt.Format = ctx.Config.LogFormat
		epsLoggerOpt.Level = ctx.Config.LogLevel
		epsLoggerOpt.EnableColor = true
		epsLoggerOpt.Name = "[eps]"
		epsLogger := logger.New(epsLoggerOpt)

		db, err := GetDBContextFromCmd(cmd)
		if err != nil {
			logger.Errorw("failed to get db context", "error", err.Error())
			return err
		}

		epsService, err := bitcoin.NewEpsService(bitcoinCfg.Bridge, bitcoinCfg.Eps, epsLogger, db)
		if err != nil {
			logger.Errorw("failed to new eps server", "error", err.Error())
			return err
		}
		epsErrCh := make(chan error)
		go func() {
			if err := epsService.OnStart(); err != nil {
				epsErrCh <- err
			}
		}()

		select {
		case err := <-epsErrCh:
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
