---
description: Implement new domains/modules, write Go backend REST APIs, Asynq workers, and architecture wiring for go-base-project
---

# Go Backend Developer Workflow

Use this workflow to implement new backend functionality following the specific `go-base-project` architecture.

## Project Architecture

The `go-base-project` follows **Clean Architecture** with a strict inward dependency flow:
`handler -> usecase -> repository -> DB/GORM`
`handler -> usecase -> service`

### Required Tech Stack
- **Router**: `gorilla/mux`
- **ORM**: `gorm.io/gorm` (PostgreSQL driver)
- **Dependency Injection**: `go.uber.org/fx`
- **Async tasks**: `github.com/hibiken/asynq`
- **Config**: `viper`
- **Metrics**: Prometheus via `internal/metrics`

## Domain Implementation Checklist

When creating a new domain, strictly implement these layers **in this order**:

1. **Entity** (`app/entities/<domain>.go`): Define the GORM model structs (`gorm.Model`).
2. **Repository** (`app/repository/<domain>/repository.go`): Implement data access via GORM. Include `New<Domain>Repository(db *gorm.DB)`.
3. **Service** (Optional) (`app/services/<domain>/service.go`): Isolate external API logic.
4. **UseCase** (`app/usecase/<domain>/usecase.go`): Define interfaces for Repo and Service. Build business logic here depending only on these interfaces, **never concrete types**. Include `New<Domain>UseCase(repo, svc)`.
5. **HTTP Handler** (`app/handler/web/<domain>/handler.go`): Define the API. Depend precisely on a local UseCase interface. Translate HTTP formats into `usecase` inputs. Return HTTP responses via `customerror.CustomError`.
6. **Task Handler** (`app/handler/tasks/<domain>/handler.go`): Implement Asynq payload unmarshaling terminating via the business logic wrapper (UseCase).
7. **Worker** (`app/workers/<domain>/worker.go`): Asynq client wrapper to enqueue payloads.
8. **FX API Module** (`cmd/api/modules/<domain>.go`): Wire dependencies. Explicitly wrap dependencies matching interfaces via func wrappers.
   - E.g: `func(s repository.DummyRepository) usecase.DummyRepository { return s }`
9. **FX Worker Module** (`cmd/worker/modules/<domain>.go`): As API Module, adding Cache logic.
10. **Register**: Add module to `fx.New(...)` arrays in `cmd/api/main.go`. Register route endpoints.
11. **Testing**: `usecase_test.go` (mock deps w/ testify/mock). `repository_test.go` (SQLite in-memory).

## QA Checklist & Pitfalls

- **NEVER** import a concrete type from another layer across a hard boundary (usecase must import interface locally defined).
- **NEVER** omit the interface adapter functions in FX, or startup will crash.
- **NEVER** execute logic containing HTTP timeouts/delays inside the Web request flow (offload to Asynq `workers`).
- Use `db.Select()` over `select *` when mapping arrays of GORM responses to reduce memory.
- Enforce pagination on list endpoints. No unbounded arrays.
