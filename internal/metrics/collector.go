package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type MetricType string

const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "histogram"
	Summary   MetricType = "summary"
)

type MetricConfig struct {
	Name       string
	Help       string
	Type       MetricType
	LabelNames []string
	Buckets    []float64
}

type MetricsCollector struct {
	metrics map[string]prometheus.Collector
}

func NewMetricsCollector(cfgs []MetricConfig) *MetricsCollector {
	collector := &MetricsCollector{
		metrics: make(map[string]prometheus.Collector),
	}

	for _, cfg := range cfgs {
		switch cfg.Type {
		case Counter:
			collector.metrics[cfg.Name] = prometheus.NewCounterVec(
				prometheus.CounterOpts{Name: cfg.Name, Help: cfg.Help},
				cfg.LabelNames,
			)
		case Gauge:
			collector.metrics[cfg.Name] = prometheus.NewGaugeVec(
				prometheus.GaugeOpts{Name: cfg.Name, Help: cfg.Help},
				cfg.LabelNames,
			)
		case Histogram:
			collector.metrics[cfg.Name] = prometheus.NewHistogramVec(
				prometheus.HistogramOpts{Name: cfg.Name, Help: cfg.Help, Buckets: cfg.Buckets},
				cfg.LabelNames,
			)
		case Summary:
			collector.metrics[cfg.Name] = prometheus.NewSummaryVec(
				prometheus.SummaryOpts{Name: cfg.Name, Help: cfg.Help},
				cfg.LabelNames,
			)
		}
	}

	for _, metric := range collector.metrics {
		prometheus.MustRegister(metric)
	}

	return collector
}

func (c *MetricsCollector) IncrementCounter(name string, labels map[string]string) {
	if metric, ok := c.metrics[name].(*prometheus.CounterVec); ok {
		metric.With(labels).Inc()
	}
}

func (c *MetricsCollector) ObserveHistogram(name string, labels map[string]string, value float64) {
	if metric, ok := c.metrics[name].(*prometheus.HistogramVec); ok {
		metric.With(labels).Observe(value)
	}
}

func (c *MetricsCollector) SetGauge(name string, labels map[string]string, value float64) {
	if metric, ok := c.metrics[name].(*prometheus.GaugeVec); ok {
		metric.With(labels).Set(value)
	}
}
