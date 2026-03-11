# godog Setup & Installation Guide

## 1. Add godog to go.mod

```bash
go get github.com/cucumber/godog@latest
go get github.com/go-resty/resty/v2@latest   # optional: nicer HTTP client
go mod tidy
```

Required additions to `go.mod`:
```
require (
    github.com/cucumber/godog v0.14.1
    github.com/go-resty/resty/v2 v2.12.0   // optional
)
```

---

## 2. Create the tests/e2e directory structure

```bash
mkdir -p tests/e2e/features
mkdir -p tests/e2e/steps
touch tests/e2e/suite_test.go
touch tests/e2e/steps/context.go
touch tests/e2e/steps/common_steps.go
```

---

## 3. First Run

Start the API server in one terminal:
```bash
make up                        # Start Docker infra (Redis, Postgres, Prometheus)
go run cmd/api/main.go         # Start API on port 8080
```

Run E2E tests in another:
```bash
cd tests/e2e
go test -v ./...
```

Or from the project root:
```bash
go test -v ./tests/e2e/...
```

---

## 4. Makefile targets (add to project Makefile)

```makefile
.PHONY: test-e2e test-e2e-smoke test-e2e-regression

test-e2e:
	go test -v ./tests/e2e/...

test-e2e-smoke:
	go test -v -args -godog.tags="@smoke" ./tests/e2e/...

test-e2e-regression:
	go test -v -args -godog.tags="@regression" ./tests/e2e/...

test-e2e-report:
	go test -v -args -godog.format=junit ./tests/e2e/... > test-results.xml
```

---

## 5. Environment Variables

```bash
E2E_BASE_URL=http://localhost:8080   # Override default API URL
CONFIG_PATH=./config.yml             # Already used by the project
```

For CI/CD, set `E2E_BASE_URL` to point at the deployed environment.

---

## 6. Validating Setup

After setup, run this check:
```bash
go build ./...          # Must compile
go vet ./tests/e2e/...  # Must pass
go test -v -run TestE2E ./tests/e2e/...  # Must execute (even if scenarios fail)
```

A successful first run with no `.feature` files looks like:
```
=== RUN   TestE2E
0 scenarios
0 steps
0s
--- PASS: TestE2E (0.00s)
```

---

## 7. godog Version Compatibility

This skill targets `godog v0.14.x`. Key API notes:
- `godog.ScenarioContext` is the step registration type (not `Suite`)
- `ctx.Step(regex, func)` registers steps
- `ctx.Before` / `ctx.After` for scenario hooks
- `godog.DocString` for multi-line step arguments
- `godog.Table` for data tables

If using an older version (v0.12.x), the `ScenarioContext` API is the same but
`godog.TestSuite` may differ slightly — check the migration guide at
https://github.com/cucumber/godog/blob/main/CHANGELOG.md

---

## 8. Debugging Failing Steps

If a step is undefined, godog prints the suggested snippet:
```
You can implement step definitions for undefined steps with these snippets:

func iSendAGETPOSTPUTPATCHDELETERequestTo(arg1, arg2 string) error {
    return godog.ErrPending
}
```

Copy, implement, and register. Never leave `godog.ErrPending` in production tests.

---

## 9. Parallel Execution Warning

Do NOT run E2E tests in parallel (`t.Parallel()`) — they share a database and will
corrupt each other's state. Run sequentially (default godog behavior).

For CI speed, split by tag:
```bash
# Job 1
go test -args -godog.tags="@smoke" ./tests/e2e/...
# Job 2 (after job 1 passes)
go test -args -godog.tags="@regression" ./tests/e2e/...
```
