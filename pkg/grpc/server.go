package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/b2network/b2-indexer/internal/app/middleware"
	"github.com/b2network/b2-indexer/internal/config"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	AuthHeaderKey          = "X-Auth-Payload"
	AuthorizationHeaderKey = "Authorization"
)

type (
	GrpcRegisterFn    func(*grpc.Server)
	GatewayRegisterFn func(ctx context.Context, mux *runtime.ServeMux, endPoint string, option []grpc.DialOption) error
)

func Run(cfg *config.HTTPConfig, grpcFn GrpcRegisterFn, gatewayFn GatewayRegisterFn) error {
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
		log.Println("register grpc gateway server failed")
		return err
	}

	grpcSvc := grpc.NewServer()
	grpcFn(grpcSvc)

	go func() {
		handler := middleware.Cors(mux)
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", cfg.HTTPPort), handler).Error())
	}()
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%v", cfg.GrpcPort))
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		err = grpcSvc.Serve(lis)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
	}()
	reflection.Register(grpcSvc)
	log.Println("http server started in port", cfg.HTTPPort)
	log.Println("grpc server started in port", cfg.GrpcPort)
	select {}
}
