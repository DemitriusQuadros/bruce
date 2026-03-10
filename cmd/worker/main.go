package main

import (
	"context"
	taskHandler "bruce/app/handler/tasks/dummy"
	worker "bruce/app/workers/dummy"
	"bruce/cmd/worker/modules"
	config "bruce/internal/configuration"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

var Registry = prometheus.NewRegistry()

type RedisConfiguration struct {
	Addr string
}

func NewRedisClient(cfg *config.Configuration) *asynq.RedisClientOpt {
	return &asynq.RedisClientOpt{
		Addr: cfg.Redis.Addr,
	}
}

func NewAsynqServer(client *asynq.RedisClientOpt) *asynq.Server {
	return asynq.NewServer(
		*client,
		asynq.Config{
			Concurrency: 10,
		},
	)
}

func RegisterHandlers(
	lc fx.Lifecycle,
	server *asynq.Server,
	cfg *config.Configuration,
	processor *taskHandler.DummyProcessor,
) {
	StartMetricsServer(cfg)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			mux := asynq.NewServeMux()

			mux.HandleFunc(worker.DummyTask, processor.HandleDummyTask)

			go server.Run(mux)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			server.Shutdown()
			return nil
		},
	})
}

func StartMetricsServer(cfg *config.Configuration) {
	h := asynqmon.New(asynqmon.Options{
		RootPath:          "/tasks/monitoring",
		RedisConnOpt:      asynq.RedisClientOpt{Addr: cfg.Redis.Addr},
		PrometheusAddress: cfg.Prometheus.Address,
	})

	r := mux.NewRouter()
	r.PathPrefix(h.RootPath()).Handler(h)

	srv := &http.Server{
		Handler: r,
		Addr:    ":9191",
	}
	go func() {
		srv.ListenAndServe()
	}()
}

func main() {
	app := fx.New(
		modules.ConfigurationModule,
		modules.DbModule,
		modules.MetricsModule,
		modules.CacheModule,
		modules.DummyModule,
		fx.Provide(
			NewRedisClient,
			NewAsynqServer,
		),
		fx.Invoke(RegisterHandlers),
	)

	app.Run()
}
