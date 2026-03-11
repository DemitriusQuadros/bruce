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
- **SDD Pipeline**: 6 Claude Code agents and skills for Spec-Driven Development — from idea validation to production-ready tests

## Claude Code Agents (SDD Pipeline)

This repository is a self-contained **Spec-Driven Development (SDD)** starter kit. Opening it in [Claude Code](https://claude.ai/code) automatically loads 6 specialized agents and skills that guide you through the full development lifecycle.

### Pipeline

```
Idea → Validate → PRD → Architecture → Code → UI → Tests
```

```
business-investor-validator
         ↓
  product-manager-prd
         ↓
   software-architect
         ↓
     go-backend-dev
         ↓
  frontend-specialist
         ↓
     qa-specialist
```

| Agent / Skill | What it does |
|---|---|
| `business-investor-validator` | Scores your idea against investor criteria; surfaces risks and opportunities |
| `product-manager-prd` | Generates a structured PRD with user stories, acceptance criteria, and scope |
| `software-architect` | Converts the PRD into a technical blueprint: domain model, API contracts, infrastructure |
| `go-backend-dev` | Implements Go domains following the clean-architecture conventions of this project |
| `frontend-specialist` | Builds UI pages (vanilla JS or React) connected to the Go API on port 8080 |
| `qa-specialist` | Writes E2E BDD tests (Gherkin + godog) from spec files in `docs/specs/` |

### Quick start

1. Clone this repo
2. Open Claude Code: `claude` in the project directory
3. Type `/` — all 6 skills appear in the slash-command list
4. Start with `/business-investor-validator` and describe your idea
5. Follow the pipeline through to `/qa-specialist`

No extra setup needed — the agents and skills are bundled in `.claude/`.