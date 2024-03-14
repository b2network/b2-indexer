package server

import (
	"log"

	"github.com/b2network/b2-indexer/internal/app/service"
	"github.com/b2network/b2-indexer/pkg/grpc"
)

func Run(ctx *Context) (err error) {
	err = grpc.Run(ctx.HTTPConfig, service.RegisterGrpcFunc(), service.RegisterGateway)
	if err != nil {
		log.Panicf(err.Error())
	}
	return nil
}
