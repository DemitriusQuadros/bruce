package modules

import (
	"go.uber.org/fx"

	"bruce/internal/metrics"
)

var MetricsModule = fx.Module("metrics",
	fx.Provide(func() *metrics.MetricsCollector {
		return metrics.NewMetricsCollector([]metrics.MetricConfig{
			{
				Name:       "http_requests_total",
				Help:       "Total of http requets received",
				Type:       metrics.Counter,
				LabelNames: []string{"path", "method"},
			},
			{
				Name:       "http_request_duration_seconds",
				Help:       "HTTP Requests duration",
				Type:       metrics.Histogram,
				LabelNames: []string{"path"},
				Buckets:    []float64{0.1, 0.3, 1.2, 5.0},
			},
		})
	}),
)
