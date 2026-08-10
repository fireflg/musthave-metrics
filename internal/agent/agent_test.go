package agent_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// fakeProvider реализует MetricsProvider
type fakeProvider struct {
	mu        sync.Mutex
	pollCount int
}

func (f *fakeProvider) CollectRuntimeMemStats() agent.Metrics {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pollCount++
	return agent.Metrics{
		"Alloc":     100.0,
		"HeapAlloc": 200.0,
	}
}

func (f *fakeProvider) NextPollCount() float64 {
	return 1
}

func (f *fakeProvider) CollectGopsUtilMetrics() (agent.Metrics, error) {
	return agent.Metrics{
		"CPU": 50.0,
	}, nil
}

func (f *fakeProvider) polls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.pollCount
}

type fakeReporter struct {
	mu       sync.Mutex
	reported []agent.Metrics
}

func (r *fakeReporter) Report(ctx context.Context, m agent.Metrics) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reported = append(r.reported, m)
	return nil
}

func (r *fakeReporter) WaitServer(ctx context.Context) error {
	return nil
}

func (r *fakeReporter) Reported() []agent.Metrics {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]agent.Metrics(nil), r.reported...)
}

func newTestLogger() *zap.SugaredLogger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ := cfg.Build()
	return logger.Sugar()
}

func TestAgent_Start(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	provider := &fakeProvider{}
	reporter := &fakeReporter{}
	cfg := &agent.Config{
		PollInterval:   1,
		ReportInterval: 1,
		RateLimit:      2,
	}
	a := agent.NewAgent(cfg, provider, reporter, newTestLogger())

	require.NoError(t, a.Start(ctx))
	assert.NotEmpty(t, reporter.Reported(), "метрики должны быть отправлены")
}

func TestAgent_RespectsReportInterval(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	provider := &fakeProvider{}
	reporter := &fakeReporter{}
	cfg := &agent.Config{
		PollInterval:   1,
		ReportInterval: 10,
		RateLimit:      1,
	}
	a := agent.NewAgent(cfg, provider, reporter, newTestLogger())

	require.NoError(t, a.Start(ctx))

	assert.GreaterOrEqual(t, provider.polls(), 1, "опрос должен был выполниться")
	assert.Empty(t, reporter.Reported(), "до истечения ReportInterval отправок быть не должно")
}

func TestAgent_RateLimitZero(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	reporter := &fakeReporter{}
	cfg := &agent.Config{PollInterval: 1, ReportInterval: 1, RateLimit: 0}
	a := agent.NewAgent(cfg, &fakeProvider{}, reporter, newTestLogger())

	require.NoError(t, a.Start(ctx))
	assert.NotEmpty(t, reporter.Reported())
}

type errReporter struct {
	fakeReporter
	err error
}

func (r *errReporter) WaitServer(ctx context.Context) error { return r.err }

func TestAgent_WaitServerError(t *testing.T) {
	reporter := &errReporter{err: assert.AnError}
	cfg := &agent.Config{PollInterval: 1, ReportInterval: 1, RateLimit: 1}
	a := agent.NewAgent(cfg, &fakeProvider{}, reporter, newTestLogger())

	assert.ErrorIs(t, a.Start(context.Background()), assert.AnError)
}

func TestProvider_PollCountIsIncrement(t *testing.T) {
	p := &agent.Provider{}

	assert.Equal(t, 1.0, p.NextPollCount())
	assert.Equal(t, 1.0, p.NextPollCount())
	assert.Equal(t, 1.0, p.NextPollCount())
}
