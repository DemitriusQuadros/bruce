package middleware

import (
	"context"
	"bruce/internal/configuration"
	"bruce/internal/metrics"
	"time"

	"github.com/hibiken/asynq"
)

const (
	total_task_execution_metric = "asyn_total_task_execution"
	total_task_duration_metric  = "asynq_total_task_duration"
)

func AsynqConfigMiddleware(
	h asynq.Handler,
	cfg *configuration.Configuration,
	collector *metrics.MetricsCollector,
) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
		start := time.Now()
		ctx = context.WithValue(ctx, configKey, cfg)
		defer func() {
			duration := time.Since(start).Seconds()

			collector.IncrementCounter(total_task_execution_metric, map[string]string{
				"task": t.Type(),
			})

			collector.ObserveHistogram(total_task_duration_metric, map[string]string{
				"task": t.Type(),
			}, duration)
		}()

		if err := h.ProcessTask(ctx, t); err != nil {
			return err
		}

		return nil
	})
}
