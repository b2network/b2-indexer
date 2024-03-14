package server

import (
	"github.com/b2network/b2-indexer/internal/app/service"
	"github.com/b2network/b2-indexer/pkg/grpc"
	"log"
)

func Run(ctx *Context) (err error) {
	err = grpc.Run(ctx.HttpConfig, service.RegisterGrpcFunc(), service.RegisterGateway)
	if err != nil {
		log.Panicf(err.Error())
	}
	return nil
}
