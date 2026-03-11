---
description: Implement E2E tests using BDD patterns (Gherkin + godog) by reading spec files (docs/specs/)
---

# QA Specialist Workflow

Use this workflow to write, generate, review, or run E2E scenarios via **Gherkin** (`.feature` files) and **godog** (`go`) step definitions for the go-base-project backend. The source of truth are Markdown specs stored in `docs/specs/`.

## E2E Testing Stack
- `github.com/cucumber/godog`
- `net/http` for API assertions
- `gorm.io/gorm` for database verification (state)
- Feature files saved in `tests/e2e/features/<domain>.feature`
- Godog definitions in `tests/e2e/steps/<domain>_steps.go`

## Step 1 — Read the Specification
Extract instructions from `docs/specs/<feature>.md`:
- **User Stories**: Drives scenario names and Actor given states.
- **API Contracts**: Drives `When I send a Request` and `Then the Response status is <code>` tests.
- **Normal Flow**: Maps to Happy Path.
- **Error Cases**: Drive Error Scenarios.
- **Edge Cases**: Explicit scenario tests per boundary.
- **Invariants**: Steps checked in `Background` or at the end of every scenario.

## Step 2 — Generate Gherkin `.feature` Files
Structure:
```gherkin
Feature: [Spec Title]
  As a [role] I want to [action] So that [outcome]

  Background:
    Given the API is running
    And the database is clean

  # Happy Path
  @smoke @REQ-001
  Scenario: [Action is successful]
    When I send a POST ...
    Then the response status is 201
    And the database contains [element]

  # Error Path
  @error @REQ-002
  Scenario: [Error case failure condition]
```
Ensure each scenario maps to `@REQ-NNN` tags from spec docs.

## Step 3 — Generate `godog` Go Steps
Create standard Gherkin step receivers inside `tests/e2e/steps/<domain>_steps.go`. Register via `Register<Domain>Steps(ctx, tc *TestContext)`. 
Ensure your database validations correctly reference `TestContext.DB.Model(&entities.<Model>{}).Where(...).Count(&count)`.

## Shared Context & Assertions
- The `TestContext` struct passes the `DB`, `HTTPClient`, `LastResponse`, and `LastBody`.
- **Database Wipe Loop**: `Background` must clear tables (truncate) in reverse foreign-key order.

## Pitfalls & Principles
- **DO NOT** use Mocks in E2E validation. Hit the real API (`localhost:8080`).
- **DO NOT** rely on `time.Sleep()` for async behavior tests. Poll the Database table `TestContext.DB.Model...` inside a timeout loop (Deadline).
- Always verify both the HTTP payload output and the Physical DB Table output matching states after an action.
