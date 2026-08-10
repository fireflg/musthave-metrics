package main

import (
	"net"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/grpcserver"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/proto"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// startGRPCServer поднимает gRPC-сервер на addr.
func startGRPCServer(
	addr string,
	metricsService service.MetricsService,
	trustedSubnet *net.IPNet,
	logger *zap.Logger,
) *grpc.Server {
	if addr == "" {
		return nil
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatal("Failed to listen for gRPC", zap.String("addr", addr), zap.Error(err))
	}

	var opts []grpc.ServerOption
	if trustedSubnet != nil {
		opts = append(opts, grpc.UnaryInterceptor(grpcserver.TrustedSubnetInterceptor(trustedSubnet, logger.Sugar())))
	}

	srv := grpc.NewServer(opts...)
	proto.RegisterMetricsServer(srv, grpcserver.NewMetricsServer(metricsService))

	go func() {
		logger.Sugar().Infof("Starting gRPC server on %s", addr)
		if err := srv.Serve(listener); err != nil {
			logger.Error("gRPC server failed", zap.Error(err))
		}
	}()

	return srv
}
