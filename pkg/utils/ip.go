package utils

import (
	"context"
	"net"
	"strings"

	"github.com/b2network/b2-indexer/pkg/log"
	"google.golang.org/grpc/metadata"
)

func ClientIP(ctx context.Context, logger log.Logger) string {
	var ip string
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	logger.Infof("client ip:%v", md.Get("X-Forwarded-For"))

	for _, forward := range md.Get("X-Forwarded-For") {
		for _, ip := range strings.Split(forward, ",") {
			if ip = strings.TrimSpace(ip); ip != "" && !HasLocalIPAddr(ip) {
				return ip
			}
		}
	}

	for _, ip = range md.Get("x-real-ip") {
		if ip = strings.TrimSpace(ip); ip != "" && !HasLocalIPAddr(ip) {
			return ip
		}
	}

	return ""
}

func HasLocalIPAddr(ip string) bool {
	return HasLocalIP(net.ParseIP(ip))
}

func HasLocalIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}

	return ip4[0] == 10 || // 10.0.0.0/8
		(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) || // 172.16.0.0/12
		(ip4[0] == 169 && ip4[1] == 254) || // 169.254.0.0/16
		(ip4[0] == 192 && ip4[1] == 168) // 192.168.0.0/16
}
