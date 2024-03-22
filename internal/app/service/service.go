package service

import (
	"context"
	"fmt"
	"net/http"
	"os"

	pb "github.com/b2network/b2-indexer/api/protobuf"
	"github.com/b2network/b2-indexer/internal/config"
	"github.com/b2network/b2-indexer/internal/types"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

const (
	Success = 0
)

func version(mux *runtime.ServeMux, version int64) {
	pattern := runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2}, []string{"v1", "doc", "version"}, ""))
	mux.Handle("GET", pattern, func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(fmt.Sprintf(`{"version": "%d"}`, version)))
		if err != nil {
			log.Errorw("version wirte info err:", "error", err)
		}
	})
}

func registerDoc(mux *runtime.ServeMux, path string) {
	pattern := runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2}, []string{"v1", "doc", "swagger"}, ""))
	mux.Handle("GET", pattern, func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
		fileContent, err := os.ReadFile(path)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write([]byte(`{"code":1,"message":"Response read error"}`))
			if err != nil {
				http.Error(w, "Response write error", http.StatusInternalServerError)
			}
		} else {
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(fileContent)
			if err != nil {
				http.Error(w, "Response write error", http.StatusInternalServerError)
			}
		}
	})
}

func RegisterGateway(ctx context.Context, mux *runtime.ServeMux, endPoint string, option []grpc.DialOption) error {
	version(mux, 1)
	registerDoc(mux, "./api/protobuf/api.swagger.json")
	if err := pb.RegisterHelloServiceHandlerFromEndpoint(ctx, mux, endPoint, option); err != nil {
		log.Fatalf("RegisterHelloServiceHandlerFromEndpoint failed: %v", err)
	}
	if err := pb.RegisterNotifyServiceHandlerFromEndpoint(ctx, mux, endPoint, option); err != nil {
		log.Fatalf("RegisterNotifyServiceHandlerFromEndpoint failed: %v", err)
	}
	return nil
}

func RegisterGrpcFunc() func(server *grpc.Server) {
	return func(svc *grpc.Server) {
		pb.RegisterHelloServiceServer(svc, newHelloServer())
		pb.RegisterNotifyServiceServer(svc, newNotifyServer())
	}
}

func GetDBContext(ctx context.Context) (*gorm.DB, error) {
	if v := ctx.Value(types.DBContextKey); v != nil {
		db := v.(*gorm.DB)
		return db, nil
	}
	return nil, fmt.Errorf("db context not set")
}

func GetListenAddress(ctx context.Context) (string, error) {
	if v := ctx.Value(types.ListenAddressContextKey); v != nil {
		serverCtx := v.(string)
		return serverCtx, nil
	}
	return "", fmt.Errorf("address context not set")
}

func GetHTTPConfig(ctx context.Context) (*config.HTTPConfig, error) {
	if v := ctx.Value(types.HTTPConfigContextKey); v != nil {
		serverCtx := v.(*config.HTTPConfig)
		return serverCtx, nil
	}
	return nil, fmt.Errorf("address context not set")
}
