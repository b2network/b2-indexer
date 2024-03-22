package server

import (
	"context"
	"log"

	"github.com/b2network/b2-indexer/internal/app/service"
	"github.com/b2network/b2-indexer/internal/config"
	"github.com/b2network/b2-indexer/internal/types"
	"github.com/b2network/b2-indexer/pkg/grpc"
	googleGrpc "google.golang.org/grpc"
	"gorm.io/gorm"
)

func Run(ctx context.Context, serverCtx *Context, db *gorm.DB) (err error) {
	if serverCtx.BitcoinConfig.IndexerListenAddress == "" {
		log.Panic("listen address empty")
	}
	grpcOpts := GrpcOpts(serverCtx.BitcoinConfig.IndexerListenAddress, serverCtx.HTTPConfig, db)
	err = grpc.Run(ctx, serverCtx.HTTPConfig, grpcOpts, service.RegisterGrpcFunc(), service.RegisterGateway)
	if err != nil {
		log.Panicf(err.Error())
	}
	return nil
}

func GrpcOpts(listenAddress string, httpConfig *config.HTTPConfig, db *gorm.DB) googleGrpc.ServerOption {
	grpcOpt := googleGrpc.UnaryInterceptor(googleGrpc.UnaryServerInterceptor(
		func(ctx context.Context, req interface{}, _ *googleGrpc.UnaryServerInfo, handler googleGrpc.UnaryHandler) (resp interface{}, err error) {
			ctx = context.WithValue(ctx, types.DBContextKey, db)
			ctx = context.WithValue(ctx, types.ListenAddressContextKey, listenAddress)
			ctx = context.WithValue(ctx, types.HTTPConfigContextKey, httpConfig)
			return handler(ctx, req)
		}))
	return grpcOpt
}
