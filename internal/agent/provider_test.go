package agent_test

import (
	"fmt"
	"testing"

	"github.com/fireflg/ago-musthave-metrics-tpl/internal/agent"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/stretchr/testify/assert"
)

func TestNextPollCount(t *testing.T) {
	p := &agent.Provider{}

	assert.Equal(t, 1.0, p.NextPollCount())
	assert.Equal(t, 2.0, p.NextPollCount())
	assert.Equal(t, 3.0, p.NextPollCount())
}

func TestCollectRuntimeMemStats(t *testing.T) {
	p := &agent.Provider{}
	m := p.CollectRuntimeMemStats()

	_, ok := m["RandomValue"]
	assert.True(t, ok)

	for _, name := range agent.MemStatFields {
		val, exists := m[name]
		assert.True(t, exists, "field %s should exist", name)
		assert.IsType(t, float64(0), val)
	}
}

func TestCollectGopsUtilMetrics(t *testing.T) {
	p := &agent.Provider{}
	m, err := p.CollectGopsUtilMetrics()
	assert.NoError(t, err)

	assert.Contains(t, m, "TotalMemory")
	assert.Contains(t, m, "FreeMemory")

	numCPU, _ := cpu.Counts(true)
	for i := 1; i <= numCPU; i++ {
		assert.Contains(t, m, fmt.Sprintf("CPUutilization%d", i))
	}
}
