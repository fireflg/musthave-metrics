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

// Интервалы по умолчанию, если в конфигурации задано неположительное значение.
const (
	defaultPollInterval   = 2 * time.Second
	defaultReportInterval = 10 * time.Second
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

type metricsBuffer struct {
	mu    sync.Mutex
	items []Metrics
}

func (b *metricsBuffer) add(items ...Metrics) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.items = append(b.items, items...)
}

func (b *metricsBuffer) drain() []Metrics {
	b.mu.Lock()
	defer b.mu.Unlock()

	items := b.items
	b.items = nil
	return items
}

func (a *Agent) runPoller(ctx context.Context, buf *metricsBuffer, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		buf.add(a.collectMetrics()...)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *Agent) runReporter(ctx context.Context, buf *metricsBuffer, metricsCh chan<- Metrics, reportInterval time.Duration) {
	ticker := time.NewTicker(reportInterval)
	defer ticker.Stop()
	defer close(metricsCh)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		for _, metric := range buf.drain() {
			select {
			case metricsCh <- metric:
			case <-ctx.Done():
				return
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
			a.logger.Debugw("Sending metrics", "count", len(metric))
			if err := a.reporter.Report(ctx, metric); err != nil {
				a.logger.Errorw("Failed to send metrics", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// Start запускает агент сбора метрик и возвращает управление после отмены ctx.
func (a *Agent) Start(ctx context.Context) error {
	pollInterval := intervalOrDefault(a.cfg.PollInterval, defaultPollInterval)
	reportInterval := intervalOrDefault(a.cfg.ReportInterval, defaultReportInterval)

	rateLimit := a.cfg.RateLimit
	if rateLimit < 1 {
		rateLimit = 1
	}

	metricsCh := make(chan Metrics, rateLimit*2)
	buf := &metricsBuffer{}

	a.logger.Infow("Agent started",
		"poll_interval", pollInterval,
		"report_interval", reportInterval,
		"rate_limit", rateLimit,
	)

	if err := a.reporter.WaitServer(ctx); err != nil {
		return err
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.runPoller(ctx, buf, pollInterval)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.runReporter(ctx, buf, metricsCh, reportInterval)
	}()

	for i := 0; i < rateLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.metricsWorker(ctx, metricsCh)
		}()
	}

	<-ctx.Done()
	wg.Wait()

	return nil
}

func intervalOrDefault(seconds int, fallback time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
