package agent

import "runtime"

type MetricsProvider interface {
	Poll() runtime.MemStats
}

type Provider struct{}

func (p *Provider) Poll() runtime.MemStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return memStats
}
