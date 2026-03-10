# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Running the applications locally
```bash
go run cmd/api/main.go        # HTTP API server (port 8080)
go run cmd/worker/main.go     # Asynq task worker + monitoring UI (port 9191)
go run cmd/console/main.go    # Terminal UI dashboard
```

Or via Makefile:
```bash
make run-api-local
make run-worker-local
make run-console
```

### Infrastructure (Docker)
```bash
make up       # Build and start all containers (Redis, PostgreSQL, Prometheus, Grafana)
make down     # Stop containers, keep volumes
make stop     # Stop containers without removing them
make restart  # Restart containers
make logs     # Tail logs
make clean    # Remove containers, volumes, images
```

### Tests
```bash
go test ./...                                    # Run all tests
go test ./app/usecase/dummy/...                  # Run tests in a specific package
go test -run TestDummyUseCase_CreateDummy ./app/usecase/dummy/...  # Run a single test
```

### Configuration
Copy `config.example.yml` to `config.yml` and fill in credentials. The app reads `config.yml` from the working directory by default, or from the path in `CONFIG_PATH` env var.

## Architecture

This is a Go scaffolding project with three separate entry points sharing common `app/` and `internal/` packages. The `Dummy` domain serves as a working reference implementation to copy when adding new domains.

### Entry Points (`cmd/`)
- **api** — REST API server using `gorilla/mux`, port 8080. Registers routes, runs DB migrations via GORM `AutoMigrate` on startup, and exposes a `/metrics` Prometheus endpoint.
- **worker** — Asynq async task processor. Registers task handlers and serves the Asynqmon monitoring UI at port 9191 (`/tasks/monitoring`).
- **console** — Terminal dashboard using `termui`. Wires dependencies manually via `cmd/console/dependencies/` (no FX) and renders pages from `cmd/console/pages/`.

Each `cmd/api` and `cmd/worker` entry point defines its own `modules/` directory with FX dependency injection modules (`ConfigurationModule`, `DbModule`, `MetricsModule`, `DummyModule`). The worker also includes `CacheModule`.

### Application Layer (`app/`)

Follows clean architecture — dependencies flow inward:

```
handler → usecase → repository → (entities/DB)
                 → services    → (external/domain logic)
```

- **`app/entities/`** — GORM-mapped domain types. Example: `Dummy{ID int64, Text string}`.
- **`app/repository/`** — GORM data access. Each domain has its own package (e.g., `dummy/`).
- **`app/usecase/`** — Business logic. Each usecase defines its own interfaces for its dependencies (`DummyRepository`, `DummyService`), enabling mock-based testing without infrastructure.
- **`app/handler/web/`** — HTTP handlers implementing the `Route` interface (`Handlers() []handler.Configuration`). Each handler takes a `UseCase` interface, not a concrete type. DTOs live alongside the handler.
- **`app/handler/tasks/`** — Asynq task handlers. Each processor unmarshals a payload and delegates to a usecase method.
- **`app/services/`** — Domain services containing business logic or external calls that don't belong in a usecase. Example: `DummyService.ProcessDummy`.
- **`app/workers/`** — Asynq client wrappers that serialize a payload and enqueue a named task. Example: `DummyWorker.EnqueueDummyTask`.

### FX Module Wiring (`cmd/*/modules/dummy.go`)

Each domain module wires all layers using `fx.Provide` and explicit interface adapters:

```go
var DummyModule = fx.Module("dummy",
    fx.Provide(
        repository.NewDummyRepository,
        service.NewDummyService,
        usecase.NewDummyUseCase,
        worker.NewDummyWorker,
        taskHandler.NewDummyProcessor,
        webHandler.NewDummyHandler,
        func(s repository.DummyRepository) usecase.DummyRepository { return s },
        func(s service.DummyService) usecase.DummyService { return s },
        func(s *usecase.DummyUseCase) taskHandler.DummyUseCase { return s },
        func(s *usecase.DummyUseCase) webHandler.UseCase { return s },
    ),
)
```

New domains follow the same pattern: create module file, add to `fx.New(...)` in `main.go`, register routes and task handlers.

### Internal / Infrastructure (`internal/`)
- **`configuration/`** — Viper-based config loader. Reads `config.yml` with keys: `DB.HOST`, `DB.PORT`, `DB.USER`, `DB.PASSWORD`, `DB.DBNAME`, `DB.SSLMODE`, `REDIS.ADDR`, `PROMETHEUS.ADDRESS`.
- **`db/`** — GORM PostgreSQL connection factory (`NewDatabase`).
- **`memcache/`** — Thread-safe in-memory key-value store.
- **`customerror/`** — `CustomError{Code, Message}` — errors carry HTTP status codes so handlers can respond correctly.
- **`metrics/`** — Prometheus counter/gauge wrapper (`MetricsCollector`).
- **`middleware/`** — HTTP middleware (`ConfigMiddleware`) and Asynq middleware that inject config and metrics into requests.
- **`handler/`** — Shared `handler.Configuration` struct (`Pattern`, `Action`, `Method`) used by all web handlers.

### Testing Patterns
- Mocks are defined inline in `_test.go` files using `testify/mock` — no separate `mocks/` directory.
- Repository tests use SQLite in-memory (`gorm.io/driver/sqlite`) to avoid needing a real Postgres instance.
- Usecase and task handler tests mock all interfaces — no DB, Redis, or external services required.

### Monitoring
- Prometheus scrapes the API (`/metrics`) and worker. Grafana dashboards are in `docs/grafana/`.
- Asynqmon UI available at `http://localhost:9191/tasks/monitoring` when the worker is running.
