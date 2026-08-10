package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/proto"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/memory"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func newTestService() service.MetricsService {
	return service.NewMetricsService(memory.NewMemoryRepository(), nil)
}

func freeAddr(t *testing.T) string {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := lis.Addr().String()
	require.NoError(t, lis.Close())
	return addr
}

func dial(t *testing.T, addr string) proto.MetricsClient {
	t.Helper()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return proto.NewMetricsClient(conn)
}

func TestStartGRPCServer_DisabledWithoutAddr(t *testing.T) {
	assert.Nil(t, startGRPCServer("", newTestService(), nil, zap.NewNop()))
}

func TestStartGRPCServer_AcceptsBatch(t *testing.T) {
	svc := newTestService()
	addr := freeAddr(t)

	srv := startGRPCServer(addr, svc, nil, zap.NewNop())
	require.NotNil(t, srv)
	defer srv.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := dial(t, addr).UpdateMetrics(ctx, &proto.UpdateMetricsRequest{
		Metrics: []*proto.Metric{{Id: "g", Type: proto.Metric_GAUGE, Value: 4.2}},
	})
	require.NoError(t, err)

	got, err := svc.GetMetric("g", "gauge")
	require.NoError(t, err)
	assert.Equal(t, 4.2, *got.Value)
}

func TestStartGRPCServer_TrustedSubnetInterceptorIsWired(t *testing.T) {
	_, subnet, err := net.ParseCIDR("10.0.0.0/8")
	require.NoError(t, err)

	addr := freeAddr(t)
	srv := startGRPCServer(addr, newTestService(), subnet, zap.NewNop())
	require.NotNil(t, srv)
	defer srv.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := dial(t, addr)

	_, err = client.UpdateMetrics(ctx, &proto.UpdateMetricsRequest{})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	allowed := metadata.AppendToOutgoingContext(ctx, "x-real-ip", "10.1.2.3")
	_, err = client.UpdateMetrics(allowed, &proto.UpdateMetricsRequest{})
	assert.NoError(t, err)
}
