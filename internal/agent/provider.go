package agent

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"sync/atomic"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

// Provider собирает системные метрики и метрики runtime.
type Provider struct {
	count    int64
	reported int64
}

// MetricsProvider определяет интерфейс для сбора метрик.
type MetricsProvider interface {
	// CollectRuntimeMemStats возвращает статистику памяти runtime.
	CollectRuntimeMemStats() Metrics
	// NextPollCount возвращает следующее значение счетчика опросов.
	NextPollCount() float64
	// CollectGopsUtilMetrics возвращает системные метрики с помощью gopsutil.
	CollectGopsUtilMetrics() (Metrics, error)
}

// Metrics — карта имен метрик к их значениям float64.
type Metrics map[string]float64

// NextPollCount возвращает прирост счётчика опросов с прошлого вызова.
// Сервер сам суммирует delta, поэтому накопленное значение отправлять нельзя.
func (p *Provider) NextPollCount() float64 {
	total := atomic.AddInt64(&p.count, 1)
	prev := atomic.SwapInt64(&p.reported, total)
	return float64(total - prev)
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
