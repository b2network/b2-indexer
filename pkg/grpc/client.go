package grpc

import (
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/pkg/errors"
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

type ClientOptionPort uint32

var defaultClientOptionPort ClientOptionPort = 9000

func WithClientPortOption(port uint32) ClientOptionPort {
	return ClientOptionPort(port)
}

// GetClientConnection Get grpc connection client
func GetClientConnection(serviceName string, options ...interface{}) (*grpc.ClientConn, error) {
	for _, option := range options {
		if value, ok := option.(ClientOptionPort); ok {
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
		return nil, errors.WithStack(err)
	}
	clientInstances[connectionHash] = conn

	return conn, nil
}

func generateServiceUniqueName(serviceName string, servicePort uint32) string {
	hashData := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", serviceName, servicePort)))
	return fmt.Sprintf("%x", hashData)
}
