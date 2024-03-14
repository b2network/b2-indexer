package service

import (
	"context"
	pb "github.com/b2network/b2-indexer/api/protobuf"
	"github.com/b2network/b2-indexer/api/protobuf/vo"
)

type helloServer struct {
	pb.UnimplementedHelloServiceServer
}

func newHelloServer() *helloServer {
	return &helloServer{}
}

func (s *helloServer) GetHello(ctx context.Context, req *vo.HelloRequest) (*vo.HelloResponse, error) {
	return &vo.HelloResponse{
		Code:    Success,
		Message: "success",
		Data: &vo.HelloResponse_Data{
			Info: "hello",
		},
	}, nil
}
