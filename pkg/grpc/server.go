package grpc

import (
	"context"
	"fmt"
	"github.com/b2network/b2-indexer/internal/app/middleware"
	"github.com/b2network/b2-indexer/internal/config"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"log"
	"net"
	"net/http"
)

const (
	AuthHeaderKey          = "X-Auth-Payload"
	AuthorizationHeaderKey = "Authorization"
)

type GrpcRegisterFn func(*grpc.Server)
type GrpcGatewayRegisterFn func(ctx context.Context, mux *runtime.ServeMux, endPoint string, option []grpc.DialOption) error

func Run(cfg *config.HttpConfig, grpcFn GrpcRegisterFn, gatewayFn GrpcGatewayRegisterFn) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: true,
				UseProtoNames:   true,
			},
		}),
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			switch key {
			case AuthHeaderKey, AuthorizationHeaderKey:
				return key, true
			}
			return "", false

		}),
	)
	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := gatewayFn(ctx, mux, fmt.Sprintf(":%v", cfg.GrpcPort), opts); err != nil {
		log.Fatal("register grpc gateway server failed")
	}

	grpcSvc := grpc.NewServer()
	grpcFn(grpcSvc)

	go func() {
		handler := middleware.Cors(mux)
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", cfg.HttpPort), handler).Error())
	}()
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%v", cfg.GrpcPort))
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		grpcSvc.Serve(lis)
	}()
	reflection.Register(grpcSvc)
	log.Println("http server started in port", cfg.HttpPort)
	log.Println("grpc server started in port", cfg.GrpcPort)
	select {}
}
