# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Running the application locally
```bash
go run cmd/bruce/main.go      # HTTP API server (port 8080)
```

### Infrastructure (Docker)
```bash
docker compose up -d          # Start Redis + Bruce containers
docker compose down           # Stop containers, keep volumes
docker compose logs -f        # Tail logs
```

### Tests
```bash
go test ./...                 # Run all tests
go test -race ./...           # Run with race detector
go test -cover ./...          # Coverage report
```

### Quality
```bash
go build ./...                # Must compile with zero errors
go vet ./...                  # Static analysis
gofmt -l .                    # Check formatting (empty = clean)
```

### Configuration
Copy `config.example.yml` to `config.yml` and fill in credentials. The app reads `config.yml`
from the working directory by default, or from the path in the `CONFIG_PATH` env var.

---

## Architecture

Bruce is a **single-binary Go application** — one entrypoint, no FX, manual dependency
injection in `main.go`. All application code lives under `internal/`.

```
bruce/
├── cmd/bruce/main.go          # Sole entrypoint — wires all packages, graceful shutdown
├── internal/
│   ├── config/                # Viper config loader + Config struct
│   ├── database/              # SQLite connection (WAL mode) + schema.sql DDL
│   ├── domain/                # Plain Go domain structs (Session, Message, ConfigEntry)
│   ├── repository/            # database/sql CRUD implementations
│   ├── ai/                    # LLMService interface + Anthropic implementation
│   ├── worker/                # Asynq task payloads, processor, dispatcher
│   ├── connectors/
│   │   ├── whatsapp/          # WhatsApp connector (uses whatsmeow)
│   │   └── discord/           # Discord connector
│   └── api/
│       ├── router.go          # Gorilla Mux setup
│       └── handlers/          # HTTP handlers (health, sessions, messages, config)
├── web/public/                # Static web assets (index.html, style.css, app.js)
├── docker-compose.yml         # redis:7-alpine + bruce (no Postgres, no Prometheus)
├── config.example.yml
└── Dockerfile                 # CGO-enabled build (required for mattn/go-sqlite3)
```

### Dependency flow

```
internal/api/handlers → internal/repository → internal/domain
                      → internal/ai
                      → internal/worker
internal/worker/processor → internal/repository
                          → internal/ai
```

### Tech Stack
- **Router**: `gorilla/mux`
- **Database**: SQLite via `database/sql` + `mattn/go-sqlite3` (WAL mode, MaxOpenConns=1)
- **Async tasks**: `github.com/hibiken/asynq` (Redis-backed)
- **Config**: `viper` — keys: `server.port`, `redis.address`, `sqlite.dsn`, `claude.*`, `connectors.*`, `ui.*`
- **Tests**: `testify` — repository tests use SQLite in-memory (`:memory:`)

### Key architectural rules
- **No FX** — all wiring is explicit in `cmd/bruce/main.go`
- **No GORM** — repositories use `database/sql` directly
- **No PostgreSQL** — SQLite is the sole database
- `internal/` packages depend only inward — handlers never import repositories directly,
  they go through interfaces defined in the handler package
- `mattn/go-sqlite3` requires CGO — always build with `CGO_ENABLED=1`

### `config.yml` schema
```yaml
server:
  port: 8080

redis:
  address: "redis:6379"
  max_retries: 3

sqlite:
  dsn: "./data/bruce.db"

claude:
  api_key: ""
  model: "claude-opus-4-6"
  max_tokens: 1024
  context_window: 15

connectors:
  whatsapp:
    enabled: false
    device_store_dsn: "./data/whatsapp.db"
  discord:
    enabled: false
    bot_token: ""

ui:
  default_system_prompt: "You are Bruce, a personal AI assistant."
```

### Graceful shutdown order
1. HTTP server (`Shutdown` with 10s timeout) — stops accepting new requests
2. Asynq server (`Shutdown`) — finishes in-flight tasks
3. SQLite DB (`Close` via defer) — safe after no writers remain

### `/health` endpoint
```
GET /health
  Auth: none
  Returns: 200 { "status": "ok", "uptime_seconds": N }
```

---

## Agents & Skills (SDD Pipeline)

This project ships with 6 Claude Code agents and 6 skills that encode a full **Spec-Driven
Development (SDD)** pipeline. They are automatically loaded when you open Claude Code in
this directory.

### Pipeline

```
business-investor-validator → product-manager-prd → software-architect → go-backend-dev → frontend-specialist → qa-specialist
```

| Stage | Agent / Skill | Purpose |
|---|---|---|
| 1 | `business-investor-validator` | Validate the idea with an investor-grade scorecard |
| 2 | `product-manager-prd` | Generate a full PRD from the validated idea |
| 3 | `software-architect` | Convert the PRD into a technical blueprint |
| 4 | `go-backend-dev` | Implement Go packages under `internal/` |
| 5 | `frontend-specialist` | Build UI pages against the Go API (port 8080) |
| 6 | `qa-specialist` | Generate E2E BDD tests from specs in `docs/specs/` |

### How to invoke

- **Skills** — type `/` in Claude Code to see the slash-command list (e.g. `/go-backend-dev`, `/qa-specialist`).
- **Agents** — ask Claude to "use the `go-backend-dev` agent" and it will load with full architectural context.

### Example flow

```
1. /business-investor-validator   ← describe your idea
2. /product-manager-prd           ← turn validated idea into PRD
3. /software-architect            ← convert PRD into technical plan
4. /go-backend-dev                ← implement internal/ packages from the plan
5. /frontend-specialist           ← build UI against the API
6. /qa-specialist                 ← generate BDD tests for the spec
```
