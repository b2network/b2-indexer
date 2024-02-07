package client

import (
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/b2network/b2-indexer/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	clientInstances map[string]*grpc.ClientConn
	writeMutex      sync.Mutex
)

func init() {
	clientInstances = make(map[string]*grpc.ClientConn, 0)
}

type OptionPort uint32

var defaultClientOptionPort OptionPort = 9000

func WithClientPortOption(port uint32) OptionPort {
	return OptionPort(port)
}

// GetClientConnection Get grpc connection client
func GetClientConnection(serviceName string, options ...interface{}) (*grpc.ClientConn, error) {
	for _, option := range options {
		if value, ok := option.(OptionPort); ok {
			defaultClientOptionPort = value
		}
	}

	connectionHash := generateServiceUniqueName(serviceName, uint32(defaultClientOptionPort))
	if instance, ok := clientInstances[connectionHash]; ok {
		return instance, nil
	}

	writeMutex.Lock()
	defer writeMutex.Unlock()

	// connect service
	serviceHost := fmt.Sprintf("%s:%d", serviceName, defaultClientOptionPort)
	conn, err := grpc.Dial(serviceHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Errorf("get grpc connection error", err)
		return nil, err
	}
	clientInstances[connectionHash] = conn

	return conn, nil
}

func generateServiceUniqueName(serviceName string, servicePort uint32) string {
	hashData := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", serviceName, servicePort)))
	return fmt.Sprintf("%x", hashData)
}
