// Package grpcserver предоставляет gRPC-транспорт для приёма метрик от агента.
//
// Пакет реализует сервис Metrics из api/metrics.proto: метод UpdateMetrics
// принимает батч метрик и сохраняет их через тот же слой сервиса, что и
// HTTP-обработчики.
package grpcserver

import (
	"context"

	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	pb "github.com/fireflg/go-musthave-metrics-tpl/internal/proto"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MetricsServer реализует gRPC-сервис Metrics.
type MetricsServer struct {
	pb.UnimplementedMetricsServer
	service service.MetricsService
}

// NewMetricsServer создает новый экземпляр MetricsServer.
func NewMetricsServer(service service.MetricsService) *MetricsServer {
	return &MetricsServer{service: service}
}

// UpdateMetrics принимает батч метрик и сохраняет его в хранилище.
func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	metrics := make([]models.Metrics, 0, len(req.GetMetrics()))
	for _, m := range req.GetMetrics() {
		metrics = append(metrics, toModel(m))
	}

	if err := s.service.SetMetricBatch(ctx, metrics); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UpdateMetricsResponse{}, nil
}

// toModel конвертирует метрику из protobuf-представления во внутреннюю модель.
func toModel(m *pb.Metric) models.Metrics {
	metric := models.Metrics{ID: m.GetId()}

	switch m.GetType() {
	case pb.Metric_COUNTER:
		delta := m.GetDelta()
		metric.MType = models.Counter
		metric.Delta = &delta
	default:
		value := m.GetValue()
		metric.MType = models.Gauge
		metric.Value = &value
	}

	return metric
}
