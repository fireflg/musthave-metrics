package agent

import (
	"math/rand"
	"reflect"
	"runtime"
)

type Metrics map[string]float64

type MetricsStorage interface {
	UpdateData(memStats runtime.MemStats)
}

func (m Metrics) UpdateData(memStats runtime.MemStats) {
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
	m["PollCount"]++
}
