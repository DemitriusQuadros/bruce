# Spec 30: Local Database Connector [BACKEND]

## Overview

Implement `db_query` tool that executes read-only SQL queries against user-configured local databases (SQLite, PostgreSQL, MySQL). Tool accepts SQL string, validates for safety (only SELECT allowed, no DDL), parameterizes input, and returns result set as JSON. Supports time-range queries for analytics (e.g., "users created in last 7 days"). Phase 2 read-only; Phase 3+ can support write with explicit confirmation.

## Phase

**Phase 2** (Weeks 7–14)

## Prerequisites

- Tool registry exists (Spec 12)
- Config system supports database connection strings
- Driver packages for SQLite, PostgreSQL, MySQL are vendored

## Deliverables

**Files to Create:**
- `internal/tools/database/db.go` — Database query tool implementation
- `internal/tools/database/validator.go` — SQL safety validation (SELECT-only, no DDL)
- `internal/tools/database/parser.go` — Query result formatter (JSON)

**Files to Modify:**
- `internal/tools/registry.go` — register database tool at startup
- `cmd/bruce/main.go` — instantiate database tool with connection pool
- `internal/config/config.go` — add `tools.database.connections` (map of name to DSN)
- `config.example.yml` — document database connection string format

## Acceptance Criteria

- [ ] Tool `db_query` accepts: `database` (friendly name from config), `query` (SQL string), `params` (optional JSON values for placeholders)
- [ ] Tool validates query: only SELECT, FROM, WHERE, JOIN, GROUP BY, ORDER BY, LIMIT allowed; no INSERT, UPDATE, DELETE, DROP, CREATE, ALTER
- [ ] Tool rejects parameterless queries to prevent SQL injection (or uses regex allowlist of safe patterns)
- [ ] Query results are returned as JSON array of objects (column: value pairs)
- [ ] Large result sets are truncated (max 1000 rows, 100KB JSON) to prevent memory explosion
- [ ] Tool handles connection errors, SQL syntax errors, permission denied gracefully
- [ ] Query timeout is 30s (database-specific, e.g., `SET STATEMENT_TIMEOUT` for PostgreSQL)
- [ ] Latency: <2s p95 for typical queries (mocked)
- [ ] Log all executed queries (sanitized) to tool_executions table for audit

## API / Component Contract

**Config**:
```yaml
tools:
  database:
    connections:
      production:
        dsn: "user=postgres password=... host=localhost dbname=prod"
        driver: "postgres"
      analytics:
        dsn: "./data/analytics.db"
        driver: "sqlite"
```

**Tool Schema**:
```json
{
	"name": "db_query",
	"description": "Execute a read-only SQL query against a local database",
	"input_schema": {
		"type": "object",
		"properties": {
			"database": {
				"type": "string",
				"description": "Database name from config (e.g., 'production', 'analytics')"
			},
			"query": {
				"type": "string",
				"description": "SQL SELECT query"
			},
			"params": {
				"type": "array",
				"description": "Optional query parameter values (for ? or $1 placeholders)",
				"items": {}
			}
		},
		"required": ["database", "query"]
	}
}
```

**SQL Validation** (pseudo-code):
```go
func ValidateQuery(sql string) error {
	// Tokenize SQL
	// Check first keyword is SELECT
	// Reject: INSERT, UPDATE, DELETE, DROP, CREATE, ALTER, TRUNCATE
	// Reject: `INTO`, `VALUES` (unless in WHERE clause for pattern matching)
	// Allow: SELECT, FROM, WHERE, JOIN, GROUP BY, HAVING, ORDER BY, LIMIT, OFFSET
	// Return error if invalid
}
```

## Out of Scope

- Write queries (INSERT, UPDATE, DELETE — Phase 3+ with approval)
- Stored procedures / functions
- Schema introspection (list tables, columns)
- Query optimization suggestions
- Query history / saved queries
