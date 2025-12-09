package agent

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Agent struct {
	cfg      *Config
	provider MetricsProvider
	reporter MetricsReporter
	storage  MetricsStorage
	logger   *zap.SugaredLogger
}

func NewAgent(cfg *Config, provider MetricsProvider, reporter MetricsReporter, logger *zap.SugaredLogger, storage MetricsStorage,
) *Agent {
	return &Agent{
		cfg:      cfg,
		provider: provider,
		reporter: reporter,
		logger:   logger,
		storage:  storage,
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
			a.storage.UpdateData(a.provider.Poll())
			a.logger.Infof("Pool metric")

		case <-reportTicker.C:
			for metric, value := range a.storage.(Metrics) {

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
