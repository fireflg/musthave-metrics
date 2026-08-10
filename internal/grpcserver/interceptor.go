package grpcserver

import (
	"context"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// RealIPMetadataKey — ключ метаданных, в котором агент передаёт свой IP-адрес.
const RealIPMetadataKey = "x-real-ip"

// TrustedSubnetInterceptor — unary-интерсептор, пропускающий вызов только если
// IP-адрес агента, переданный в метаданных x-real-ip, входит в доверенную
// подсеть trustedSubnet. Иначе возвращает codes.PermissionDenied.
func TrustedSubnetInterceptor(trustedSubnet *net.IPNet, logger *zap.SugaredLogger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		var rawIP string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get(RealIPMetadataKey); len(values) > 0 {
				rawIP = values[0]
			}
		}

		realIP := net.ParseIP(rawIP)
		if realIP == nil || !trustedSubnet.Contains(realIP) {
			logger.Warnw("Rejected request from untrusted subnet", "x-real-ip", rawIP, "method", info.FullMethod)
			return nil, status.Error(codes.PermissionDenied, "agent IP is not in trusted subnet")
		}

		return handler(ctx, req)
	}
}
