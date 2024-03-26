package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/b2network/b2-indexer/internal/app/middleware"
	"github.com/b2network/b2-indexer/internal/config"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func Run(ctx context.Context, cfg *config.HTTPConfig, grpcOpts grpc.ServerOption, grpcFn RegisterFn, gatewayFn GatewayRegisterFn) error {
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
	grpcSvc := grpc.NewServer(grpcOpts)
	grpcFn(grpcSvc)

	errChan := make(chan error)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%v", cfg.HTTPPort),
		Handler:      middleware.Cors(mux),
		ReadTimeout:  TimeoutSecond * time.Second,
		WriteTimeout: TimeoutSecond * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			errChan <- fmt.Errorf("HTTP server error: %v", err)
		}
	}()
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%v", cfg.GrpcPort))
		if err != nil {
			errChan <- fmt.Errorf("gRPC server listen error: %v", err)
			return
		}
		if err := grpcSvc.Serve(lis); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %v", err)
		}
	}()
	reflection.Register(grpcSvc)
	log.Println("http server started in port", cfg.HTTPPort)
	log.Println("grpc server started in port", cfg.GrpcPort)
	for err := range errChan {
		log.Printf("Error occurred: %v, stopping servers", err)
		switch {
		case strings.Contains(err.Error(), "HTTP server"):
			if err := server.Shutdown(context.Background()); err != nil {
				log.Printf("HTTP server shutdown failed: %v", err)
				return err
			}
		case strings.Contains(err.Error(), "gRPC server"):
			grpcSvc.GracefulStop()
		default:
			log.Printf("HTTP server error: %v", err)
			return err
		}
	}
	return nil
}
