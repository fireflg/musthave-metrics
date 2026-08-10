// Package agent предоставляет агент сбора метрик, который периодически
// собирает системные метрики и отправляет их на сервер.
//
// Агент собирает статистику использования памяти runtime, CPU,
// использование памяти и другие системные метрики, затем отправляет их
// на настроенный эндпоинт сервера.
package agent

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Agent собирает и отправляет системные метрики на удаленный сервер.
type Agent struct {
	cfg      *Config
	provider MetricsProvider
	reporter MetricsReporter
	logger   *zap.SugaredLogger
}

// NewAgent создает новый экземпляр Agent с заданной конфигурацией,
// провайдером метрик, репортером и логгером.
func NewAgent(cfg *Config, provider MetricsProvider, reporter MetricsReporter, logger *zap.SugaredLogger,
) *Agent {
	return &Agent{
		cfg:      cfg,
		provider: provider,
		reporter: reporter,
		logger:   logger,
	}
}

func (a *Agent) collectMetrics() []Metrics {
	var result []Metrics

	runtimeMetrics := a.provider.CollectRuntimeMemStats()
	runtimeMetrics["PollCount"] = a.provider.NextPollCount()
	result = append(result, runtimeMetrics)

	gopsutilMetrics, err := a.provider.CollectGopsUtilMetrics()
	if err == nil {
		result = append(result, gopsutilMetrics)
	}

	return result
}

func (a *Agent) runPoller(ctx context.Context, metricsCh chan Metrics, pollInterval time.Duration) {
	for {
		select {
		case <-ctx.Done():
			close(metricsCh)
			return
		default:
			metrics := a.collectMetrics()
			for _, metric := range metrics {
				metricsCh <- metric
			}

			if pollInterval > 0 {
				select {
				case <-time.After(pollInterval):
				case <-ctx.Done():
					close(metricsCh)
					return
				}
			}
		}
	}
}

func (a *Agent) metricsWorker(ctx context.Context, metricsCh <-chan Metrics) {
	for {
		select {
		case metric, ok := <-metricsCh:
			if !ok {
				return
			}
			a.logger.Info("Sending metrics", zap.Any("metric", metric))
			if err := a.reporter.Report(ctx, metric); err != nil {
				a.logger.Info("Sending metrics", zap.Any("metric", metric))
			}
		case <-ctx.Done():
			return
		}
	}
}

// Start запускает агент сбора метрик.
func (a *Agent) Start(ctx context.Context) error {
	metricsCh := make(chan Metrics, a.cfg.RateLimit*2)

	a.logger.Infof("Agent started")

	if err := a.reporter.WaitServer(ctx); err != nil {
		return err
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.runPoller(ctx, metricsCh, time.Duration(a.cfg.PollInterval)*time.Second)
	}()

	for i := 0; i < a.cfg.RateLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.metricsWorker(ctx, metricsCh)
		}()
	}

	<-ctx.Done()
	wg.Wait()
	return ctx.Err()
}
