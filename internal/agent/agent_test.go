package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/fireflg/ago-musthave-metrics-tpl/internal/agent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// fakeProvider реализует MetricsProvider
type fakeProvider struct {
	pollCount int
}

func (f *fakeProvider) CollectRuntimeMemStats() agent.Metrics {
	f.pollCount++
	return agent.Metrics{
		"Alloc":     100.0,
		"HeapAlloc": 200.0,
	}
}

func (f *fakeProvider) NextPollCount() float64 {
	return float64(f.pollCount)
}

func (f *fakeProvider) CollectGopsUtilMetrics() (agent.Metrics, error) {
	return agent.Metrics{
		"CPU": 50.0,
	}, nil
}

// реализует MetricsReporter
type fakeReporter struct {
	Reported []agent.Metrics
}

func (r *fakeReporter) Report(ctx context.Context, m agent.Metrics) error {
	r.Reported = append(r.Reported, m)
	return nil
}

func (r *fakeReporter) WaitServer(ctx context.Context) error {
	return nil
}

func newTestLogger() *zap.SugaredLogger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ := cfg.Build()
	return logger.Sugar()
}

func TestAgent_Start(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	provider := &fakeProvider{}
	reporter := &fakeReporter{}
	logger := newTestLogger()
	cfg := &agent.Config{
		PollInterval: 1,
		RateLimit:    2,
	}
	a := agent.NewAgent(cfg, provider, reporter, logger)

	err := a.Start(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context deadline exceeded, got %v", err)
	}

	if len(reporter.Reported) == 0 {
		t.Fatalf("Expected some metrics to be reported, got 0")
	}
}
