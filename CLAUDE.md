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
├── web/public/                # Static web assets (modular ES modules)
│   ├── index.html             # SPA entrypoint
│   ├── css/                   # Modular CSS
│   │   ├── main.css           # @import entrypoint
│   │   ├── tokens.css         # Design tokens (colors, spacing, fonts)
│   │   ├── base.css           # Global base styles (body, scrollbar)
│   │   ├── layout.css         # Header, main, tab sections
│   │   └── components/        # Component-scoped CSS
│   │       ├── button.css, card.css, badge.css, form.css
│   │       ├── connector.css, session.css, logs.css
│   │       └── toast.css, spinner.css
│   ├── js/                    # Modular JavaScript (ES modules)
│   │   ├── main.js            # Bootstrap application
│   │   ├── api.js             # HTTP client (fetch wrapper)
│   │   ├── store.js           # Reactive state (CustomEvent-based)
│   │   ├── router.js          # Hash-based routing
│   │   ├── toast.js           # Toast notifications
│   │   └── modules/           # Feature modules
│   │       ├── connectors.js  # Connectors tab
│   │       ├── sessions.js    # Sessions tab
│   │       ├── logs.js        # Logs tab (subscribes to sessions changes)
│   │       └── settings.js    # Settings tab
│   └── bruce-logo.png
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

## Frontend Architecture

The frontend is a **modular vanilla ES6 application** served directly from the Go binary via `web/embed.go`. No build step, no bundler — native browser ES modules with CSS `@import`.

### Key Principles
- **Zero external dependencies** — vanilla JS, native fetch API, CSS custom properties
- **Module decoupling** — each tab (connectors, sessions, logs, settings) is an independent module
- **Reactive store** — global state via `CustomEvent` (no circular imports between modules)
- **Hash-based routing** — `#connectors`, `#sessions`, `#logs`, `#settings` — tab state persists on refresh
- **CSS tokens** — all colors, spacing, typography in `css/tokens.css`, used throughout components

### Module Initialization Flow
1. `index.html` loads `js/main.js` with `type="module"`
2. `main.js` imports all modules (connectors, sessions, logs, settings)
3. Each module calls `init()` — registers with router, subscribes to store changes, sets up event listeners
4. `router.start()` wires hash navigation and activates initial tab

### Store Pattern (Fixes Coupling)
```javascript
// Store provides reactive state without circular imports
setState({ sessions: data });           // Update state
subscribe('sessions', callback);        // Listen for changes via CustomEvent
const value = getState('sessions');     // Read current value
```

**Key fix**: Sessions module no longer directly calls `populateLogsDropdown()`. Instead:
- Sessions calls `setState({ sessions: data })`
- Logs module has `subscribe('sessions', populateLogsDropdown)`
- Both modules import from store; no direct imports between them

### CSS Architecture
- `tokens.css` — CSS custom properties (colors, spacing, radius, fonts, transitions, shadows)
- `base.css` — global resets (*, html, body, scrollbar)
- `layout.css` — header, main, tab sections, responsive grid
- `components/*.css` — isolated component styles (button, card, badge, form, connector, session, logs, toast, spinner)
- `main.css` — single `@import` entrypoint that chains all files

### Frontend Testing
E2E tests use **Playwright** (no unit test framework needed for vanilla JS):

```bash
make test-frontend    # Installs npm deps + runs tests
# Or manually:
npm install && npx playwright test
```

Test file: `tests/ui/navigation.spec.ts` covers:
- Hash routing navigation
- Tab state persistence on refresh
- Sessions → Logs decoupling (dropdown populated without visiting Sessions tab)
- Form rendering and interaction

### Development Workflow
1. Edit `.css` or `.js` files in `web/public/`
2. Refresh browser at `http://localhost:8080`
3. No build step — changes are live (Go binary auto-serves from disk in dev)
4. Run `make test-frontend` to verify E2E tests pass

### Deployment
The `Dockerfile` builds a single binary with all frontend assets embedded via `//go:embed public`. Deploy just the binary — no separate static file server needed.

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
