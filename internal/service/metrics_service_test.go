package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
)

func TestMetricsStorage_SetMetric(t *testing.T) {
	type fields struct {
		Metrics []models.Metrics
	}
	type args struct {
		metricType  string
		metricName  string
		metricValue string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		wantVal string
	}{
		{
			name:    "valid gauge metric - new entry",
			fields:  fields{Metrics: []models.Metrics{}},
			args:    args{metricType: "gauge", metricName: "CPUUsage", metricValue: "42.5"},
			wantErr: false,
			wantVal: "42.5",
		},
		{
			name:    "valid counter metric - new entry",
			fields:  fields{Metrics: []models.Metrics{}},
			args:    args{metricType: "counter", metricName: "Requests", metricValue: "100"},
			wantErr: false,
			wantVal: "100",
		},
		{
			name:    "invalid metric type",
			fields:  fields{Metrics: []models.Metrics{}},
			args:    args{metricType: "unknown", metricName: "BadMetric", metricValue: "5"},
			wantErr: true,
		},
		{
			name: "update existing gauge metric",
			fields: fields{
				Metrics: []models.Metrics{
					{ID: "Memory", MType: "gauge", Value: float64Ptr(10.0)},
				},
			},
			args:    args{metricType: "gauge", metricName: "Memory", metricValue: "20"},
			wantErr: false,
			wantVal: "20",
		},
		{
			name: "update existing counter metric",
			fields: fields{
				Metrics: []models.Metrics{
					{ID: "Requests", MType: "counter", Value: float64Ptr(5.0), Delta: int64Ptr(5)},
				},
			},
			args:    args{metricType: "counter", metricName: "Requests", metricValue: "7"},
			wantErr: false,
			wantVal: "12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricsMap := make(map[string]models.Metrics)
			for _, f := range tt.fields.Metrics {
				metricsMap[f.ID] = f
			}
			m := &MetricsStorage{Metrics: metricsMap}

			val, err := strconv.ParseFloat(tt.args.metricValue, 64)
			if err != nil && !tt.wantErr {
				t.Fatalf("cannot parse metricValue: %v", err)
			}
			err = m.SetMetric(tt.args.metricType, tt.args.metricName, val)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetMetric() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				met := m.Metrics[tt.args.metricName]
				if met.Value == nil {
					t.Fatalf("metric.Value == nil")
				}
				got := strconv.FormatFloat(*met.Value, 'f', -1, 64)
				if got != tt.wantVal {
					t.Errorf("value = %v, want %v", got, tt.wantVal)
				}
			}
		})
	}
}

func TestMetricsStorage_GetMetric(t *testing.T) {
	valGauge := 50.5
	valCounter := 10.0
	delta := int64(10)

	tests := []struct {
		name       string
		fields     []models.Metrics
		metricType string
		metricName string
		wantErr    bool
		wantValue  float64
	}{
		{
			name: "get existing gauge metric",
			fields: []models.Metrics{
				{ID: "DiskUsage", MType: "gauge", Value: float64Ptr(valGauge)},
			},
			metricType: "gauge",
			metricName: "DiskUsage",
			wantErr:    false,
			wantValue:  50.5,
		},
		{
			name: "get existing counter metric",
			fields: []models.Metrics{
				{ID: "Requests", MType: "counter", Value: float64Ptr(valCounter), Delta: &delta},
			},
			metricType: "counter",
			metricName: "Requests",
			wantErr:    false,
			wantValue:  10,
		},
		{
			name: "metric not found",
			fields: []models.Metrics{
				{ID: "CPU", MType: "gauge", Value: float64Ptr(valGauge)},
			},
			metricType: "gauge",
			metricName: "Memory",
			wantErr:    true,
		},
		{
			name:       "invalid metric type",
			fields:     []models.Metrics{},
			metricType: "badtype",
			metricName: "Any",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricsMap := make(map[string]models.Metrics)
			for _, f := range tt.fields {
				metricsMap[f.ID] = f
			}
			m := &MetricsStorage{Metrics: metricsMap}

			got, err := m.GetMetric(tt.metricType, tt.metricName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantValue {
				t.Errorf("GetMetric() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func TestMetricsStorage_DecodeAndSetMetric(t *testing.T) {
	m := &MetricsStorage{Metrics: make(map[string]models.Metrics)}

	body := `{
        "id": "CPU",
        "type": "gauge",
        "value": 55.5
    }`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	err := m.DecodeAndSetMetric(req)
	if err != nil {
		t.Fatalf("DecodeAndSetMetric() error = %v", err)
	}

	metric, ok := m.Metrics["CPU"]
	if !ok {
		t.Fatalf("metric CPU not stored")
	}
	if metric.Value == nil || *metric.Value != 55.5 {
		t.Fatalf("expected value 55.5, got %v", metric.Value)
	}
	if metric.MType != "gauge" {
		t.Fatalf("expected type gauge, got %s", metric.MType)
	}
}

func float64Ptr(v float64) *float64 { return &v }
func int64Ptr(v int64) *int64       { return &v }
