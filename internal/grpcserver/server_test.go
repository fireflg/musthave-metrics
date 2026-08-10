package grpcserver_test

import (
	"context"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/grpcserver"
	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	pb "github.com/fireflg/go-musthave-metrics-tpl/internal/proto"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/memory"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsServer_UpdateMetrics(t *testing.T) {
	svc := service.NewMetricsService(memory.NewMemoryRepository(), nil)
	srv := grpcserver.NewMetricsServer(svc)

	req := &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{
			{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 123.45},
			{Id: "PollCount", Type: pb.Metric_COUNTER, Delta: 7},
		},
	}

	_, err := srv.UpdateMetrics(context.Background(), req)
	require.NoError(t, err)

	gauge, err := svc.GetMetric("Alloc", models.Gauge)
	require.NoError(t, err)
	require.NotNil(t, gauge.Value)
	assert.Equal(t, 123.45, *gauge.Value)

	counter, err := svc.GetMetric("PollCount", models.Counter)
	require.NoError(t, err)
	require.NotNil(t, counter.Delta)
	assert.Equal(t, int64(7), *counter.Delta)
}

func TestMetricsServer_UpdateMetrics_EmptyBatch(t *testing.T) {
	svc := service.NewMetricsService(memory.NewMemoryRepository(), nil)
	srv := grpcserver.NewMetricsServer(svc)

	resp, err := srv.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
