package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MetricType represents the type of metric being collected.
type MetricType string

const (
	CounterType   MetricType = "COUNTER"
	GaugeType     MetricType = "GAUGE"
	HistogramType MetricType = "HISTOGRAM"
)

// MetricSnapshot represents a single metric data point persisted to the database.
type MetricSnapshot struct {
	ID                     string            `json:"id"`
	Timestamp              time.Time         `json:"timestamp"`
	MetricName             string            `json:"metric_name"`
	MetricType             MetricType        `json:"metric_type"`
	Value                  float64           `json:"value"`
	Labels                 map[string]string `json:"labels"`
	AggregationWindowSecs  int               `json:"aggregation_window_seconds"`
}

// counterKey uniquely identifies a counter based on name and labels.
type counterKey struct {
	name   string
	labels string // JSON-serialized map
}

// histogramSamples holds all samples for a single histogram.
type histogramSamples struct {
	samples []float64
	mu      sync.Mutex
}

// Collector manages in-process metrics collection with periodic flush to database.
type Collector struct {
	db          *sql.DB
	counters    map[counterKey]float64
	histograms map[counterKey]*histogramSamples
	mu          sync.RWMutex
	enabled     bool
	enabledMu   sync.RWMutex
}

// NewCollector creates a new Collector with the given database connection.
func NewCollector(db *sql.DB) *Collector {
	return &Collector{
		db:          db,
		counters:    make(map[counterKey]float64),
		histograms: make(map[counterKey]*histogramSamples),
		enabled:     true,
	}
}

// SetEnabled allows toggling metrics collection on/off.
func (c *Collector) SetEnabled(enabled bool) {
	c.enabledMu.Lock()
	defer c.enabledMu.Unlock()
	c.enabled = enabled
}

// IsEnabled returns whether metrics collection is currently enabled.
func (c *Collector) IsEnabled() bool {
	c.enabledMu.RLock()
	defer c.enabledMu.RUnlock()
	return c.enabled
}

// IncrementCounter increments a counter by 1 with optional labels.
func (c *Collector) IncrementCounter(name string, labels map[string]string) {
	c.RecordCounter(name, 1.0, labels)
}

// RecordCounter adds a value to a counter with optional labels.
func (c *Collector) RecordCounter(name string, value float64, labels map[string]string) {
	if !c.IsEnabled() {
		return
	}

	key := c.makeKey(name, labels)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.counters[key] += value
}

// RecordHistogram records a sample to a histogram with optional labels.
func (c *Collector) RecordHistogram(name string, value float64, labels map[string]string) {
	if !c.IsEnabled() {
		return
	}

	key := c.makeKey(name, labels)

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.histograms[key]; !ok {
		c.histograms[key] = &histogramSamples{}
	}

	c.histograms[key].mu.Lock()
	c.histograms[key].samples = append(c.histograms[key].samples, value)
	c.histograms[key].mu.Unlock()
}

// Flush persists all in-process metrics to the database and resets counters.
func (c *Collector) Flush(ctx context.Context) error {
	if !c.IsEnabled() {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Prepare all snapshots to insert
	snapshots := make([]MetricSnapshot, 0)

	// Process counters
	for key, value := range c.counters {
		labels := c.parseLabelsFromKey(key)
		snapshot := MetricSnapshot{
			ID:                    uuid.New().String(),
			Timestamp:             time.Now().UTC(),
			MetricName:            key.name,
			MetricType:            CounterType,
			Value:                 value,
			Labels:                labels,
			AggregationWindowSecs: 60,
		}
		snapshots = append(snapshots, snapshot)
	}

	// Process histograms (generate p50, p95, p99 snapshots)
	for key, hs := range c.histograms {
		hs.mu.Lock()
		if len(hs.samples) > 0 {
			labels := c.parseLabelsFromKey(key)

			// Calculate percentiles
			p50, p95, p99 := calculatePercentiles(hs.samples)

			for _, p := range []struct {
				name  string
				value float64
			}{
				{"p50", p50},
				{"p95", p95},
				{"p99", p99},
			} {
				labelCopy := make(map[string]string)
				for k, v := range labels {
					labelCopy[k] = v
				}
				labelCopy["percentile"] = p.name

				snapshot := MetricSnapshot{
					ID:                    uuid.New().String(),
					Timestamp:             time.Now().UTC(),
					MetricName:            key.name,
					MetricType:            HistogramType,
					Value:                 p.value,
					Labels:                labelCopy,
					AggregationWindowSecs: 60,
				}
				snapshots = append(snapshots, snapshot)
			}
		}
		hs.mu.Unlock()
	}

	// Insert all snapshots into database
	for _, snapshot := range snapshots {
		labelsJSON, _ := json.Marshal(snapshot.Labels)
		_, err := c.db.ExecContext(ctx,
			`INSERT INTO metric_snapshots
			 (id, timestamp, metric_name, metric_type, value, labels, aggregation_window_seconds)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			snapshot.ID,
			snapshot.Timestamp,
			snapshot.MetricName,
			snapshot.MetricType,
			snapshot.Value,
			string(labelsJSON),
			snapshot.AggregationWindowSecs,
		)
		if err != nil {
			log.Printf("WARNING: failed to insert metric snapshot: %v", err)
		}
	}

	// Reset counters and histograms
	c.counters = make(map[counterKey]float64)
	c.histograms = make(map[counterKey]*histogramSamples)

	return nil
}

// makeKey creates a unique key from metric name and labels.
func (c *Collector) makeKey(name string, labels map[string]string) counterKey {
	labelsJSON, _ := json.Marshal(labels)
	return counterKey{
		name:   name,
		labels: string(labelsJSON),
	}
}

// parseLabelsFromKey extracts the labels map from a counterKey.
func (c *Collector) parseLabelsFromKey(key counterKey) map[string]string {
	var labels map[string]string
	json.Unmarshal([]byte(key.labels), &labels)
	if labels == nil {
		labels = make(map[string]string)
	}
	return labels
}

// calculatePercentiles computes p50, p95, and p99 from samples.
func calculatePercentiles(samples []float64) (p50, p95, p99 float64) {
	if len(samples) == 0 {
		return 0, 0, 0
	}

	// Sort samples
	sorted := make([]float64, len(samples))
	copy(sorted, samples)
	sort.Float64s(sorted)

	// Calculate percentiles using linear interpolation
	p50 = percentile(sorted, 50)
	p95 = percentile(sorted, 95)
	p99 = percentile(sorted, 99)

	return
}

// percentile computes the nth percentile from sorted samples.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	index := (p / 100) * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
