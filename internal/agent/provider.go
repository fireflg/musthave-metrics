package agent

import (
	"fmt"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"math/rand"
	"reflect"
	"runtime"
	"sync/atomic"
)

type Provider struct {
	count int64
}

type MetricsProvider interface {
	CollectRuntimeMemStats() Metrics
	NextPollCount() float64
	CollectGopsUtilMetrics() (Metrics, error)
}

type Metrics map[string]float64

func (p *Provider) NextPollCount() float64 {
	atomic.AddInt64(&p.count, 1)
	return float64(p.count)
}

func (p *Provider) CollectRuntimeMemStats() Metrics {
	var memStats runtime.MemStats

	runtime.ReadMemStats(&memStats)

	m := make(Metrics)

	v := reflect.ValueOf(memStats)

	for _, name := range MemStatFields {
		field := v.FieldByName(name)
		if field.IsValid() {
			switch field.Kind() {
			case reflect.Uint64:
				m[name] = float64(field.Uint())
			case reflect.Float64:
				m[name] = field.Float()
			case reflect.Int64:
				m[name] = float64(field.Int())
			default:
				m[name] = 0
			}
		} else {
			m[name] = 0
		}
	}

	m["RandomValue"] = rand.ExpFloat64()

	return m
}

func (p *Provider) CollectGopsUtilMetrics() (Metrics, error) {
	m := make(Metrics)
	v, err := mem.VirtualMemory()
	if err != nil {
		return m, err
	}
	percents, err := cpu.Percent(0, true)
	if err != nil {
		return m, err
	}
	m["TotalMemory"] = float64(v.Total)
	m["FreeMemory"] = float64(v.Free)
	for i, p := range percents {
		m[fmt.Sprintf("CPUutilization%d", i+1)] = p
	}
	return m, nil
}
