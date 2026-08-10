package agent_test

import (
	"context"
	"net"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/agent"
	pb "github.com/fireflg/go-musthave-metrics-tpl/internal/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// captureServer записывает последний полученный батч метрик и метаданные запроса.
type captureServer struct {
	pb.UnimplementedMetricsServer
	metrics []*pb.Metric
	md      metadata.MD
}

func (s *captureServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	s.metrics = req.GetMetrics()
	s.md, _ = metadata.FromIncomingContext(ctx)
	return &pb.UpdateMetricsResponse{}, nil
}

// startTestGRPCServer поднимает gRPC-сервер на свободном порту и возвращает
// его адрес вместе с обработчиком, накапливающим полученные запросы.
func startTestGRPCServer(t *testing.T) (string, *captureServer) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	capture := &captureServer{}
	srv := grpc.NewServer()
	pb.RegisterMetricsServer(srv, capture)

	go func() { _ = srv.Serve(listener) }()
	t.Cleanup(srv.Stop)

	return listener.Addr().String(), capture
}

func TestGRPCReporter_Report(t *testing.T) {
	addr, capture := startTestGRPCServer(t)

	reporter, err := agent.NewGRPCReporter(addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reporter.Close() })

	err = reporter.Report(context.Background(), agent.Metrics{"Alloc": 1.5, "PollCount": 3})
	require.NoError(t, err)

	require.Len(t, capture.metrics, 2)

	byID := map[string]*pb.Metric{}
	for _, m := range capture.metrics {
		byID[m.GetId()] = m
	}

	require.Contains(t, byID, "Alloc")
	assert.Equal(t, pb.Metric_GAUGE, byID["Alloc"].GetType())
	assert.Equal(t, 1.5, byID["Alloc"].GetValue())

	require.Contains(t, byID, "PollCount")
	assert.Equal(t, pb.Metric_COUNTER, byID["PollCount"].GetType())
	assert.Equal(t, int64(3), byID["PollCount"].GetDelta())
}

func TestGRPCReporter_Report_SendsRealIPMetadata(t *testing.T) {
	addr, capture := startTestGRPCServer(t)

	reporter, err := agent.NewGRPCReporter(addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reporter.Close() })

	require.NoError(t, reporter.Report(context.Background(), agent.Metrics{"Alloc": 1.0}))

	realIP := capture.md.Get("x-real-ip")
	require.Len(t, realIP, 1)
	assert.NotNil(t, net.ParseIP(realIP[0]), "x-real-ip must contain a valid IP address")
}

func TestGRPCReporter_WaitServer(t *testing.T) {
	addr, _ := startTestGRPCServer(t)

	reporter, err := agent.NewGRPCReporter(addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reporter.Close() })

	assert.NoError(t, reporter.WaitServer(context.Background()))
}
