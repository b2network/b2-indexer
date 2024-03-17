package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"

	"github.com/b2network/b2-indexer/internal/app/middleware"
	"github.com/b2network/b2-indexer/internal/config"
	"github.com/b2network/b2-indexer/internal/types"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	AuthHeaderKey          = "X-Auth-Payload"
	AuthorizationHeaderKey = "Authorization"

	TimeoutSecond = 60
)

type (
	RegisterFn        func(*grpc.Server)
	GatewayRegisterFn func(ctx context.Context, mux *runtime.ServeMux, endPoint string, option []grpc.DialOption) error
)

func Run(ctx context.Context, cfg *config.HTTPConfig, db *gorm.DB, grpcFn RegisterFn, gatewayFn GatewayRegisterFn) error {
	ctx, cancel := context.WithCancel(ctx)
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
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := gatewayFn(ctx, mux, fmt.Sprintf(":%v", cfg.GrpcPort), opts); err != nil {
		log.Println("register grpc gateway server failed")
		return err
	}
	grpcOpt := grpc.UnaryInterceptor(grpc.UnaryServerInterceptor(
		func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			ctx = context.WithValue(ctx, types.DBContextKey, db)
			return handler(ctx, req)
		}))
	grpcSvc := grpc.NewServer(grpcOpt)
	grpcFn(grpcSvc)
	handler := middleware.Cors(mux)
	go func() {
		server := &http.Server{
			Addr:         fmt.Sprintf(":%v", cfg.HTTPPort),
			Handler:      handler,
			ReadTimeout:  TimeoutSecond * time.Second,
			WriteTimeout: TimeoutSecond * time.Second,
		}
		log.Fatal(server.ListenAndServe().Error())
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
