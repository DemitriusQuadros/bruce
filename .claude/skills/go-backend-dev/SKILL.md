---
name: go-backend-dev
description: >
  Senior Go backend developer agent for the go-base-project architecture. Use this skill
  whenever you need to implement new domains/modules, write or modify Go backend code, create
  REST API endpoints, add async task workers, write tests, fix bugs, refactor code, optimize
  performance, or review Go code in this project.
agent: go-backend-dev
delegation: true
---

# Go Backend Developer Skill

**This skill is a delegation point to the `go-backend-dev` agent.**

When you invoke this skill (e.g., `/go-backend-dev`), it launches a specialized agent that
implements Go backend development tasks for this project with deep knowledge of the project's
architecture, naming conventions, and patterns.

## When to use

- Implement new domains, modules, or endpoints
- Write or modify Go backend code
- Create REST API endpoints
- Add async task workers
- Write tests, fix bugs, refactor code
- Optimize performance or review Go code

## How to invoke

Type `/go-backend-dev` followed by your request:

```
/go-backend-dev add a new CRUD endpoint for users
/go-backend-dev create the payment service module
/go-backend-dev write E2E tests for the order workflow
```

The agent will launch with full access to your codebase and execute the necessary changes,
enforcing the project's specific architecture, naming, and wiring conventions.

---

## Documentation

For detailed information about the project architecture, reference the agent's knowledge base.
The agent has been trained on the full `go-base-project` structure including Clean Architecture
patterns, FX dependency injection, GORM models, Asynq task patterns, and testing conventions.

All code changes follow:
- **Inward dependency flow**: handlers → usecases → repositories → database
- **Clean Architecture**: entities, repositories, services, usecases, handlers
- **Testing patterns**: table-driven tests with testify/mock and SQLite in-memory
- **Async patterns**: Asynq task enqueuers in `workers/`, processors in `handler/tasks/`
