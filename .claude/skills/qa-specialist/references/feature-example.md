# Feature Example — Dummy Domain (Complete E2E)

> This example maps to the Dummy domain endpoints read from `app/handler/web/dummy/handler.go`:
> POST /dummy, GET /dummy?id=, POST /dummy/process?id=, GET /dummy/all

---

## tests/e2e/features/dummy.feature

```gherkin
Feature: Dummy Domain API
  As a developer using the go-base-project scaffolding
  I want to manage dummy entities via the REST API
  So that I can verify the full stack works end-to-end

  Background:
    Given the API is running
    And the database is clean

  # ─── Happy Path ────────────────────────────────────────────────────────────

  @smoke @REQ-001
  Scenario: Create a dummy item successfully
    When I send a POST request to "/dummy" with body:
      """
      {"text": "hello world"}
      """
    Then the response status is 201
    And the database contains a dummy with text "hello world"

  @smoke @REQ-002
  Scenario: Retrieve a dummy item by ID
    Given a dummy item exists with text "fetch me"
    When I send a GET request to "/dummy" with query param "id" = "1"
    Then the response status is 200
    And the response body field "text" is "fetch me"
    And the response body field "id" is "1"

  @smoke @REQ-003
  Scenario: Retrieve all dummy items
    Given the following dummy items exist:
      | text        |
      | first item  |
      | second item |
      | third item  |
    When I send a GET request to "/dummy/all"
    Then the response status is 200
    And the response body is a list of 3 items

  @smoke @REQ-004
  Scenario: Process a dummy item
    Given a dummy item exists with text "process me"
    When I send a POST request to "/dummy/process" with query param "id" = "1"
    Then the response status is 200
    And the dummy with id 1 has been processed

  # ─── Error Cases ───────────────────────────────────────────────────────────

  @error-case @REQ-001
  Scenario: Reject creation with empty body
    When I send a POST request to "/dummy" with body:
      """
      {}
      """
    Then the response status is 400

  @error-case @REQ-001
  Scenario: Reject creation with malformed JSON
    When I send a POST request to "/dummy" with body:
      """
      not-valid-json
      """
    Then the response status is 400

  @error-case @REQ-002
  Scenario: Return 400 for non-numeric ID on GET
    When I send a GET request to "/dummy" with query param "id" = "abc"
    Then the response status is 400
    And the response body contains "invalid id"

  @error-case @REQ-004
  Scenario: Return 400 for non-numeric ID on process
    When I send a POST request to "/dummy/process" with query param "id" = "xyz"
    Then the response status is 400
    And the response body contains "invalid id"

  # ─── Edge Cases ────────────────────────────────────────────────────────────

  @edge-case @REQ-002
  Scenario: Return error for ID that does not exist
    When I send a GET request to "/dummy" with query param "id" = "99999"
    Then the response status is 500

  @edge-case @REQ-003
  Scenario: Return empty list when no dummy items exist
    When I send a GET request to "/dummy/all"
    Then the response status is 200
    And the response body is an empty list

  @edge-case @REQ-001
  Scenario Outline: Text field boundary values
    When I send a POST request to "/dummy" with body:
      """
      {"text": "<text>"}
      """
    Then the response status is <status>
    Examples:
      | text                                      | status |
      | a                                         | 201    |
      | this is a normal text value               | 201    |
      | 1234567890                                | 201    |

  # ─── Invariants ────────────────────────────────────────────────────────────

  @invariant @REQ-001
  Scenario: Created dummy text is exactly preserved in DB
    When I send a POST request to "/dummy" with body:
      """
      {"text": "exact text value"}
      """
    Then the response status is 201
    And the database contains a dummy with text "exact text value"

  @invariant @REQ-002
  Scenario: GET response always includes id and text fields
    Given a dummy item exists with text "invariant check"
    When I send a GET request to "/dummy" with query param "id" = "1"
    Then the response status is 200
    And the response body contains field "id"
    And the response body contains field "text"
```

---

## tests/e2e/steps/dummy_steps.go

```go
package steps

import (
    "encoding/json"
    "fmt"

    "go-base-project/app/entities"

    "github.com/cucumber/godog"
)

func RegisterDummySteps(ctx *godog.ScenarioContext, tc *TestContext) {
    // Preconditions
    ctx.Step(`^a dummy item exists with text "([^"]*)"$`, tc.aDummyItemExistsWith)
    ctx.Step(`^the following dummy items exist:$`, tc.theFollowingDummyItemsExist)

    // Assertions
    ctx.Step(`^the database contains a dummy with text "([^"]*)"$`, tc.theDBContainsDummyWithText)
    ctx.Step(`^the dummy with id (\d+) has been processed$`, tc.theDummyHasBeenProcessed)
    ctx.Step(`^the response body contains field "([^"]*)"$`, tc.theResponseBodyContainsField)
    ctx.Step(`^the response body is an empty list$`, tc.theResponseBodyIsEmptyList)
}

func (tc *TestContext) aDummyItemExistsWith(text string) error {
    dummy := entities.Dummy{Text: text}
    return tc.DB.Create(&dummy).Error
}

func (tc *TestContext) theFollowingDummyItemsExist(table *godog.Table) error {
    for i, row := range table.Rows {
        if i == 0 {
            continue // skip header
        }
        text := row.Cells[0].Value
        dummy := entities.Dummy{Text: text}
        if err := tc.DB.Create(&dummy).Error; err != nil {
            return err
        }
    }
    return nil
}

func (tc *TestContext) theDBContainsDummyWithText(text string) error {
    var count int64
    tc.DB.Model(&entities.Dummy{}).Where("text = ?", text).Count(&count)
    if count == 0 {
        return fmt.Errorf("expected dummy with text %q in DB but found none", text)
    }
    return nil
}

func (tc *TestContext) theDummyHasBeenProcessed(id int) error {
    // After processing, verify the service ran — adapt to actual business logic
    var dummy entities.Dummy
    if err := tc.DB.First(&dummy, id).Error; err != nil {
        return fmt.Errorf("dummy with id %d not found: %w", id, err)
    }
    // Add assertion on processed state based on actual domain logic
    return nil
}

func (tc *TestContext) theResponseBodyContainsField(field string) error {
    var body map[string]interface{}
    if err := json.Unmarshal(tc.LastBody, &body); err != nil {
        return fmt.Errorf("response is not a JSON object: %w", err)
    }
    if _, ok := body[field]; !ok {
        return fmt.Errorf("expected field %q in response body but got: %s", field, string(tc.LastBody))
    }
    return nil
}

func (tc *TestContext) theResponseBodyIsEmptyList() error {
    var items []interface{}
    if err := json.Unmarshal(tc.LastBody, &items); err != nil {
        // null response is also acceptable for empty list
        if string(tc.LastBody) == "null\n" || string(tc.LastBody) == "null" {
            return nil
        }
        return fmt.Errorf("response is not a list: %s", string(tc.LastBody))
    }
    if len(items) != 0 {
        return fmt.Errorf("expected empty list but got %d items", len(items))
    }
    return nil
}
```

---

## tests/e2e/context.go

```go
package steps

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "strings"
    "time"

    "go-base-project/app/entities"

    "github.com/cucumber/godog"
)

type TestContext struct {
    BaseURL      string
    HTTPClient   *http.Client
    DB           interface{ Exec(sql string, values ...interface{}) error }
    LastResponse *http.Response
    LastBody     []byte
    ScenarioData map[string]interface{}
}

func (tc *TestContext) theAPIIsRunning() error {
    _, err := tc.HTTPClient.Get(tc.BaseURL + "/dummy/all")
    if err != nil {
        return fmt.Errorf("API unreachable at %s: %w", tc.BaseURL, err)
    }
    return nil
}

func (tc *TestContext) theDatabaseIsClean() error {
    // Truncate tables — extend slice as new domains are added
    // Order matters if FK constraints exist
    tables := []string{"dummies"}
    for _, t := range tables {
        if err := tc.DB.Exec(
            fmt.Sprintf("DELETE FROM %s", t),
        ).Error; err != nil {
            return fmt.Errorf("failed to clean table %s: %w", t, err)
        }
    }
    return nil
}

func (tc *TestContext) iSendRequest(method, path string) error {
    return tc.iSendRequestWithBody(method, path, nil)
}

func (tc *TestContext) iSendRequestWithBody(method, path string, body *godog.DocString) error {
    url := tc.BaseURL + path
    var bodyReader io.Reader
    if body != nil {
        bodyReader = bytes.NewBufferString(body.Content)
    }

    req, err := http.NewRequest(method, url, bodyReader)
    if err != nil {
        return err
    }
    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }

    resp, err := tc.HTTPClient.Do(req)
    if err != nil {
        return err
    }
    tc.LastResponse = resp
    tc.LastBody, err = io.ReadAll(resp.Body)
    resp.Body.Close()
    return err
}

func (tc *TestContext) iSendRequestWithQueryParam(method, path, key, value string) error {
    sep := "?"
    if strings.Contains(path, "?") {
        sep = "&"
    }
    return tc.iSendRequest(method, path+sep+key+"="+value)
}

func (tc *TestContext) theResponseStatusIs(expected int) error {
    if tc.LastResponse.StatusCode != expected {
        return fmt.Errorf("expected status %d, got %d\nBody: %s",
            expected, tc.LastResponse.StatusCode, string(tc.LastBody))
    }
    return nil
}

func (tc *TestContext) theResponseBodyContains(substr string) error {
    if !strings.Contains(string(tc.LastBody), substr) {
        return fmt.Errorf("expected body to contain %q but got: %s", substr, string(tc.LastBody))
    }
    return nil
}

func (tc *TestContext) theResponseBodyIsListOf(expected int) error {
    var items []interface{}
    if err := json.Unmarshal(tc.LastBody, &items); err != nil {
        return fmt.Errorf("response is not a list: %w", err)
    }
    if len(items) != expected {
        return fmt.Errorf("expected list of %d items, got %d", expected, len(items))
    }
    return nil
}
```

---

## tests/e2e/suite_test.go

```go
package e2e_test

import (
    "fmt"
    "os"
    "testing"

    "github.com/cucumber/godog"
    "go-base-project/internal/configuration"
    "go-base-project/internal/db"
    "go-base-project/tests/e2e/steps"
)

func TestE2E(t *testing.T) {
    baseURL := os.Getenv("E2E_BASE_URL")
    if baseURL == "" {
        baseURL = "http://localhost:8080"
    }

    cfg, err := configuration.NewConfiguration()
    if err != nil {
        t.Fatalf("failed to load config: %v", err)
    }

    database, err := db.NewDatabase(cfg)
    if err != nil {
        t.Fatalf("failed to connect to DB: %v", err)
    }

    tc := steps.NewTestContext(baseURL, database)

    suite := godog.TestSuite{
        Name: "e2e",
        ScenarioInitializer: func(ctx *godog.ScenarioContext) {
            steps.RegisterCommonSteps(ctx, tc)
            steps.RegisterDummySteps(ctx, tc)
        },
        Options: &godog.Options{
            Format:        "pretty",
            Paths:         []string{"features"},
            TestingT:      t,
            StopOnFailure: false,
        },
    }

    if suite.Run() != 0 {
        t.Fatal("E2E suite had failures — check output above")
    }
}
```
