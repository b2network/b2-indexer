package service

import (
	"context"
	"fmt"
	"net/http"
	"os"

	pb "github.com/b2network/b2-indexer/api/protobuf"
	"github.com/b2network/b2-indexer/pkg/log"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

const (
	Success = 0
)

func version(mux *runtime.ServeMux, version int64) {
	pattern := runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2}, []string{"v1", "doc", "version"}, ""))
	mux.Handle("GET", pattern, func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		log.Infow("request info", "request", r, "pathParams", pathParams)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(fmt.Sprintf(`{"version": "%d"}`, version)))
		if err != nil {
			log.Errorw("version wirte info err:", "error", err)
		}
	})
}

func registerDoc(mux *runtime.ServeMux, path string) {
	pattern := runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2}, []string{"v1", "doc", "swagger"}, ""))
	mux.Handle("GET", pattern, func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		log.Infow("registerDoc request info", "request", r, "pathParams", pathParams)
		fileContent, err := os.ReadFile(path)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write([]byte(`{"code":1,"message":"Response read error"}`))
			if err != nil {
				log.Errorw("registerDoc error", "error", err)
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
	return nil
}

func RegisterGrpcFunc() func(server *grpc.Server) {
	return func(svc *grpc.Server) {
		pb.RegisterHelloServiceServer(svc, newHelloServer())
	}
}
