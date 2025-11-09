package agent

import "time"

type Intervals struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
}
type Metrics map[string]float64

var memStatFields = []string{
	"Alloc", "BuckHashSys", "Frees", "GCCPUFraction",
	"HeapAlloc", "HeapIdle", "HeapInuse", "HeapReleased",
	"HeapObjects", "HeapSys", "LastGC", "Lookups",
	"MCacheInuse", "MCacheSys", "MSpanInuse", "Mallocs",
	"NextGC", "NumForcedGC", "NumGC", "OtherSys",
	"PauseTotalNs", "StackInuse", "Sys", "TotalAlloc",
}
