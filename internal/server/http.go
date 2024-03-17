package server

import (
	"context"
	"log"

	"github.com/b2network/b2-indexer/internal/app/service"
	"github.com/b2network/b2-indexer/pkg/grpc"
	"gorm.io/gorm"
)

func Run(ctx context.Context, serverCtx *Context, db *gorm.DB) (err error) {
	err = grpc.Run(ctx, serverCtx.HTTPConfig, db, service.RegisterGrpcFunc(), service.RegisterGateway)
	if err != nil {
		log.Panicf(err.Error())
	}
	return nil
}
