# Go Base Project

This project serves as a foundational scaffolding and boilerplate codebase designed to precisely jumpstart future Go web/worker applications!

## Features included out-of-the-box:
- **Clean Architecture Pattern**: Layered isolation across `Entities`, `Repositories`, `Services`, `Use Cases`, and `Handlers`.
- **Dummy Reference Module**: A fully implemented sample module showcasing:
  - Database connectivity over `gorm`
  - Fully functional REST API mapping via `gorilla/mux`
  - Integration with asynchronous background queues using `hibiken/asynq` in a headless worker binary
  - Presentation rendering loop inside a terminal UI via `termui`
- **Application Orchestration**: Bootstrapping powered by `uber-go/fx` for clean, modular dependency injection mapping
- **Automated Instrumenting**: Built-in middleware wrapping metrics and traces ready to ingest into Prometheus