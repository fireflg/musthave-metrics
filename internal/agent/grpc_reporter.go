package agent

import (
	"context"
	"fmt"

	pb "github.com/fireflg/go-musthave-metrics-tpl/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// realIPMetadataKey — ключ метаданных, в котором агент передаёт свой IP-адрес.
const realIPMetadataKey = "x-real-ip"

// GRPCReporter отправляет метрики на сервер батчами по протоколу gRPC.
type GRPCReporter struct {
	conn    *grpc.ClientConn
	client  pb.MetricsClient
	localIP *lazyIP
}

// Verify GRPCReporter implements MetricsReporter interface.
var _ MetricsReporter = (*GRPCReporter)(nil)

// NewGRPCReporter создает новый экземпляр GRPCReporter, подключаясь к
// gRPC-серверу по адресу serverAddr.
func NewGRPCReporter(serverAddr string) (*GRPCReporter, error) {
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc client: %w", err)
	}

	return &GRPCReporter{
		conn:    conn,
		client:  pb.NewMetricsClient(conn),
		localIP: newLazyIP(serverAddr),
	}, nil
}

// Close закрывает соединение с gRPC-сервером.
func (r *GRPCReporter) Close() error {
	return r.conn.Close()
}

// WaitServer ожидает готовности соединения с gRPC-сервером.
func (r *GRPCReporter) WaitServer(ctx context.Context) error {
	r.conn.Connect()

	for {
		state := r.conn.GetState()
		if state == connectivity.Ready {
			return nil
		}

		if !r.conn.WaitForStateChange(ctx, state) {
			return ctx.Err()
		}
	}
}

// Report отправляет батч метрик на сервер, передавая свой IP-адрес
// в метаданных x-real-ip.
func (r *GRPCReporter) Report(ctx context.Context, metrics Metrics) error {
	if localIP := r.localIP.get(); localIP != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, realIPMetadataKey, localIP)
	}

	if _, err := r.client.UpdateMetrics(ctx, &pb.UpdateMetricsRequest{Metrics: toProto(metrics)}); err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}

	return nil
}

// toProto конвертирует собранные метрики в protobuf-представление.
func toProto(metrics Metrics) []*pb.Metric {
	result := make([]*pb.Metric, 0, len(metrics))

	for id, value := range metrics {
		if id == "PollCount" {
			result = append(result, &pb.Metric{
				Id:    id,
				Type:  pb.Metric_COUNTER,
				Delta: int64(value),
			})
			continue
		}

		result = append(result, &pb.Metric{
			Id:    id,
			Type:  pb.Metric_GAUGE,
			Value: value,
		})
	}

	return result
}
