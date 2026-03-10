package main

import (
	"context"
	"fmt"
	"bruce/app/entities"
	dummy "bruce/app/handler/web/dummy"
	"bruce/cmd/api/modules"
	config "bruce/internal/configuration"
	"bruce/internal/handler"
	"bruce/internal/metrics"
	"bruce/internal/middleware"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/mux"	
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type Route interface {
	Handlers() []handler.Configuration
}

func main() {
	fx.New(
		modules.ConfigurationModule,
		modules.DbModule,
		modules.MetricsModule,
		modules.DummyModule,
		fx.Provide(
			NewHTTPServer,
			AsRoute(dummy.NewDummyHandler),
			fx.Annotate(
				NewServeMux,
				fx.ParamTags(`group:"routes"`),
			),
		),
		fx.Invoke(func(db *gorm.DB) {
			if err := Migrate(db); err != nil {
				log.Fatalf("failed to migrate database: %v", err)
			}
		}),
		fx.Invoke(func(*http.Server) {}),
	).Run()
}

func NewHTTPServer(
	lc fx.Lifecycle,
	router *mux.Router,
	cfg *config.Configuration,
	collector *metrics.MetricsCollector,
) *http.Server {
	wrappedMux := middleware.ConfigMiddleware(cfg, collector)(router)
	srv := &http.Server{Addr: ":8080", Handler: wrappedMux}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			fmt.Println("Starting HTTP server at", srv.Addr)
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

func NewServeMux(routes []Route) *mux.Router {
	router := mux.NewRouter()
	for _, route := range routes {
		for _, handler := range route.Handlers() {
			router.HandleFunc(handler.Pattern, handler.Action).Methods(handler.Method)
		}
	}

	router.Handle("/metrics", promhttp.Handler())
	return router
}

func AsRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Route)),
		fx.ResultTags(`group:"routes"`),
	)
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&entities.Dummy{},
	)
}
