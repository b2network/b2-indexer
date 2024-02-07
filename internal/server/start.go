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

		bidxLogger := newLogger(ctx, "[bitcoin-indexer]")
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
		b2nodeIndexerLogger := newLogger(ctx, "[b2node-indexer]")
		b2nodeIndexBridge, err := b2NodeClient(ctx.BitcoinConfig, b2nodeIndexerLogger)
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
		bridgeB2NodeLogger := newLogger(ctx, "[bridge-deposit-b2node]")
		bridgeDepositB2node, err := b2NodeClient(bitcoinCfg, bridgeB2NodeLogger)
		if err != nil {
			return err
		}

		bridgeDepositB2NodeService := bitcoin.NewBridgeDepositB2NodeService(bridgeDepositB2node, db, bridgeB2NodeLogger)
		bridgeDepositB2NodeErrCh := make(chan error)
		go func() {
			if err := bridgeDepositB2NodeService.Start(); err != nil {
				bridgeDepositB2NodeErrCh <- err
			}
		}()

		select {
		case err := <-bridgeDepositB2NodeErrCh:
			return err
		case <-time.After(5 * time.Second): // assume server started successfully
		}

		// start l1->l2 bridge service
		bridgeLogger := newLogger(ctx, "[bridge-deposit]")
		bridge, err := bitcoin.NewBridge(bitcoinCfg.Bridge, path.Join(home, "config"), bridgeLogger)
		if err != nil {
			logger.Errorw("failed to create bitcoin bridge", "error", err.Error())
			return err
		}
		bridgeB2node, err := b2NodeClient(bitcoinCfg, bridgeLogger)
		if err != nil {
			return err
		}

		bridgeService := bitcoin.NewBridgeDepositService(bridge, bridgeB2node, db, bridgeLogger)
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

func b2NodeClient(cfg *config.BitconConfig, b2NodeLog logger.Logger) (*b2node.NodeClient, error) {
	b2grpcConn, err := client.GetClientConnection(cfg.Bridge.B2NodeGRPCHost, client.WithClientPortOption(cfg.Bridge.B2NodeGRPCPort))
	if err != nil {
		return nil, err
	}
	bridgeB2node, err := b2node.NewNodeClient(
		cfg.Bridge.B2NodePrivKey,
		cfg.Bridge.B2NodeChainID,
		cfg.Bridge.B2NodeAddressPrefix,
		b2grpcConn,
		cfg.Bridge.B2NodeAPI,
		cfg.Bridge.B2NodeDenom,
		cfg.Bridge.B2NodeGasPrices,
		b2NodeLog,
	)
	if err != nil {
		logger.Errorw("failed to create b2node", "error", err.Error())
		return nil, err
	}
	return bridgeB2node, nil
}

func newLogger(ctx *Context, name string) logger.Logger {
	bridgeB2NodeLoggerOpt := logger.NewOptions()
	bridgeB2NodeLoggerOpt.Format = ctx.Config.LogFormat
	bridgeB2NodeLoggerOpt.Level = ctx.Config.LogLevel
	bridgeB2NodeLoggerOpt.EnableColor = true
	bridgeB2NodeLoggerOpt.Name = name
	bridgeB2NodeLogger := logger.New(bridgeB2NodeLoggerOpt)
	return bridgeB2NodeLogger
}
