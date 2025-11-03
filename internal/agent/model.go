package agent

import "time"

type Intervals struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
}
type Metrics map[string]float64
