package agent

import "runtime"

type Provider struct{}

func (p *Provider) Poll() runtime.MemStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return memStats
}
