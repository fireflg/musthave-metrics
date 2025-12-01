package agent

import (
	"context"
	"runtime"
	"time"

	"go.uber.org/zap"
)

type MetricsProvider interface {
	Poll() runtime.MemStats
}

type MetricsReporter interface {
	Report(ctx context.Context, metric string, value float64) error
}

type MetricsStorage interface {
	UpdateData(memStats runtime.MemStats)
}

type Agent struct {
	cfg      *Config
	provider MetricsProvider
	reporter MetricsReporter
	Storage  MetricsStorage
	logger   *zap.SugaredLogger
}

func NewAgent(cfg *Config, provider MetricsProvider, reporter MetricsReporter, logger *zap.SugaredLogger, storage MetricsStorage,
) *Agent {
	return &Agent{
		cfg:      cfg,
		provider: provider,
		reporter: reporter,
		logger:   logger,
		Storage:  storage,
	}
}

func (a *Agent) Start(ctx context.Context) error {
	pollTicker := time.NewTicker(time.Duration(a.cfg.PollInterval) * time.Second)
	reportTicker := time.NewTicker(time.Duration(a.cfg.ReportInterval) * time.Second)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	a.logger.Infof("Agent started")
	a.logger.Infof("Pool interval %v", a.cfg.PollInterval)
	a.logger.Infof("Reporting interval %v", a.cfg.ReportInterval)

	for {
		select {
		case <-pollTicker.C:
			a.Storage.UpdateData(a.provider.Poll())
			a.logger.Infof("Pool metric")

		case <-reportTicker.C:
			for metric, value := range a.Storage.(Metrics) {

				err := a.reporter.Report(ctx, metric, value)
				if err != nil {
					a.logger.Warnw("Failed to report metric", "metric", metric, "value", value, "error", err)
				}
				a.logger.Infow("Reported metric", "metric", metric, "value", value)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
