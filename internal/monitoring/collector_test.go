package monitoring

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollector_IncrementCounter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	collector := NewCollector(db)

	// Increment counter
	collector.IncrementCounter("test_counter", map[string]string{"status": "200"})
	collector.IncrementCounter("test_counter", map[string]string{"status": "200"})
	collector.IncrementCounter("test_counter", map[string]string{"status": "404"})

	// Flush metrics
	ctx := context.Background()
	err := collector.Flush(ctx)
	require.NoError(t, err)

	// Verify metrics were persisted
	var count int
	row := db.QueryRow(`SELECT COUNT(*) FROM metric_snapshots WHERE metric_name = 'test_counter'`)
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, 2, count) // 2 distinct labels (200, 404)

	// Verify values
	var value float64
	row = db.QueryRow(`SELECT value FROM metric_snapshots WHERE metric_name = 'test_counter' AND labels LIKE '%200%'`)
	require.NoError(t, row.Scan(&value))
	assert.Equal(t, 2.0, value)
}

func TestCollector_RecordHistogram(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	collector := NewCollector(db)

	// Record histogram samples
	for i := 0; i < 100; i++ {
		collector.RecordHistogram("request_duration", float64(i*10), map[string]string{"route": "/api"})
	}

	// Flush metrics
	ctx := context.Background()
	err := collector.Flush(ctx)
	require.NoError(t, err)

	// Verify histogram percentiles were persisted
	var count int
	row := db.QueryRow(`SELECT COUNT(*) FROM metric_snapshots WHERE metric_name = 'request_duration'`)
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, 3, count) // p50, p95, p99

	// Verify p50 value is approximately correct
	var p50Value float64
	row = db.QueryRow(`SELECT value FROM metric_snapshots
					   WHERE metric_name = 'request_duration' AND labels LIKE '%p50%'`)
	require.NoError(t, row.Scan(&p50Value))
	// For values 0, 10, 20, ..., 990, p50 should be around 495
	assert.Greater(t, p50Value, 400.0)
	assert.Less(t, p50Value, 600.0)
}

func TestCollector_Disabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	collector := NewCollector(db)
	collector.SetEnabled(false)

	// Increment counter while disabled
	collector.IncrementCounter("test_counter", map[string]string{})

	// Flush metrics
	ctx := context.Background()
	err := collector.Flush(ctx)
	require.NoError(t, err)

	// Verify no metrics were persisted
	var count int
	row := db.QueryRow(`SELECT COUNT(*) FROM metric_snapshots`)
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, 0, count)
}

func TestCollector_FlushResetsCounters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	collector := NewCollector(db)

	// Add metrics and flush
	collector.IncrementCounter("test_counter", map[string]string{})
	ctx := context.Background()
	err := collector.Flush(ctx)
	require.NoError(t, err)

	// Verify we have 1 row
	var count int
	row := db.QueryRow(`SELECT COUNT(*) FROM metric_snapshots`)
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, 1, count)

	// Flush again without adding metrics
	err = collector.Flush(ctx)
	require.NoError(t, err)

	// Count should still be 1 (no new metrics added)
	row = db.QueryRow(`SELECT COUNT(*) FROM metric_snapshots`)
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, 1, count)
}

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create metric_snapshots table
	_, err = db.Exec(`
		CREATE TABLE metric_snapshots (
			id TEXT PRIMARY KEY,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			metric_name TEXT NOT NULL,
			metric_type TEXT NOT NULL,
			value REAL NOT NULL,
			labels TEXT,
			aggregation_window_seconds INTEGER DEFAULT 60
		)
	`)
	require.NoError(t, err)

	return db
}
