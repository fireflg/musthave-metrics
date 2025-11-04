package service

import (
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
	}{
		{
			name: "valid gauge metric - new entry",
			fields: fields{
				Metrics: []models.Metrics{},
			},
			args: args{
				metricType:  "gauge",
				metricName:  "CPUUsage",
				metricValue: "42.5",
			},
			wantErr: false,
		},
		{
			name: "valid counter metric - new entry",
			fields: fields{
				Metrics: []models.Metrics{},
			},
			args: args{
				metricType:  "counter",
				metricName:  "Requests",
				metricValue: "100",
			},
			wantErr: false,
		},
		{
			name: "invalid metric type",
			fields: fields{
				Metrics: []models.Metrics{},
			},
			args: args{
				metricType:  "unknown",
				metricName:  "BadMetric",
				metricValue: "5",
			},
			wantErr: true,
		},
		{
			name: "invalid metric value (not number)",
			fields: fields{
				Metrics: []models.Metrics{},
			},
			args: args{
				metricType:  "gauge",
				metricName:  "CPU",
				metricValue: "notanumber",
			},
			wantErr: true,
		},
		{
			name: "update existing metric",
			fields: fields{
				Metrics: func() []models.Metrics {
					val := 10.0
					delta := int64(0)
					return []models.Metrics{
						{
							ID:    "Memory",
							MType: "gauge",
							Value: &val,
							Delta: &delta,
						},
					}
				}(),
			},
			args: args{
				metricType:  "gauge",
				metricName:  "Memory",
				metricValue: "20",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MetricsStorage{
				Metrics: tt.fields.Metrics,
			}
			if err := m.SetMetric(tt.args.metricType, tt.args.metricName, tt.args.metricValue); (err != nil) != tt.wantErr {
				t.Errorf("SetMetric() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				found := false
				for _, met := range m.Metrics {
					if met.ID == tt.args.metricName {
						found = true
						if met.Value == nil {
							t.Errorf("metric %s Value is nil", met.ID)
						}
					}
				}
				if !found {
					t.Errorf("metric %s not added", tt.args.metricName)
				}
			}
		})
	}
}

func TestMetricsStorage_GetMetric(t *testing.T) {
	valGauge := 50.5
	valCounter := 10.0
	delta := int64(0)

	tests := []struct {
		name       string
		fields     []models.Metrics
		metricType string
		metricName string
		wantErr    bool
		wantValue  string
	}{
		{
			name: "get existing gauge metric",
			fields: []models.Metrics{
				{ID: "DiskUsage", MType: "gauge", Value: &valGauge, Delta: &delta},
			},
			metricType: "gauge",
			metricName: "DiskUsage",
			wantErr:    false,
			wantValue:  "50.5",
		},
		{
			name: "get existing counter metric (integer output)",
			fields: []models.Metrics{
				{ID: "Requests", MType: "counter", Value: &valCounter, Delta: &delta},
			},
			metricType: "counter",
			metricName: "Requests",
			wantErr:    false,
			wantValue:  "10",
		},
		{
			name: "metric not found",
			fields: []models.Metrics{
				{ID: "CPU", MType: "gauge", Value: &valGauge},
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
			m := &MetricsStorage{
				Metrics: tt.fields,
			}
			got, err := m.GetMetric(tt.metricType, tt.metricName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.wantValue {
				t.Errorf("GetMetric() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}
