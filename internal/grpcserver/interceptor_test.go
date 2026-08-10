package grpcserver_test

import (
	"context"
	"net"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/grpcserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func mustParseCIDR(t *testing.T, cidr string) *net.IPNet {
	t.Helper()
	_, ipNet, err := net.ParseCIDR(cidr)
	require.NoError(t, err)
	return ipNet
}

func TestTrustedSubnetInterceptor(t *testing.T) {
	tests := []struct {
		name        string
		realIP      string
		setMetadata bool
		wantCalled  bool
	}{
		{name: "ip in subnet", realIP: "192.168.1.42", setMetadata: true, wantCalled: true},
		{name: "ip outside subnet", realIP: "10.0.0.1", setMetadata: true, wantCalled: false},
		{name: "invalid ip", realIP: "not-an-ip", setMetadata: true, wantCalled: false},
		{name: "missing metadata", setMetadata: false, wantCalled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := grpcserver.TrustedSubnetInterceptor(
				mustParseCIDR(t, "192.168.1.0/24"),
				zaptest.NewLogger(t).Sugar(),
			)

			ctx := context.Background()
			if tt.setMetadata {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(grpcserver.RealIPMetadataKey, tt.realIP))
			}

			called := false
			handler := func(ctx context.Context, req any) (any, error) {
				called = true
				return "ok", nil
			}

			resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/metrics.Metrics/UpdateMetrics"}, handler)

			assert.Equal(t, tt.wantCalled, called)
			if tt.wantCalled {
				require.NoError(t, err)
				assert.Equal(t, "ok", resp)
				return
			}

			require.Error(t, err)
			assert.Equal(t, codes.PermissionDenied, status.Code(err))
		})
	}
}
