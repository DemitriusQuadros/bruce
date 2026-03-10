package middleware

import (
	"context"
	"bruce/internal/configuration"
	"bruce/internal/metrics"
	"net/http"
	"time"
)

type key int

const (
	configKey             key = 0
	total_requests_metric     = "http_requests_total"
	total_duration_metric     = "http_request_duration_seconds"
)

func ConfigMiddleware(cfg *configuration.Configuration, collector *metrics.MetricsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := context.WithValue(r.Context(), configKey, cfg)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)

			duration := time.Since(start).Seconds()

			collector.IncrementCounter(total_requests_metric, map[string]string{
				"path":   r.URL.Path,
				"method": r.Method,
			})

			collector.ObserveHistogram(total_requests_metric, map[string]string{
				"path": r.URL.Path,
			}, duration)
		})
	}
}

func FromContext(ctx context.Context) *configuration.Configuration {
	cfg, ok := ctx.Value(configKey).(*configuration.Configuration)
	if !ok {
		return nil
	}
	return cfg
}
