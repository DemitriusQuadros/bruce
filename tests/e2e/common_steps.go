package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
)

// RegisterCommonSteps registers lifecycle, HTTP, and assertion steps reused across all domains.
func RegisterCommonSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	// --- Lifecycle ---
	ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
		tc.ScenarioData = make(map[string]interface{})
		tc.LastResponse = nil
		tc.LastBody = nil
		return gctx, nil
	})

	ctx.Step(`^the database is clean$`, tc.theDatabaseIsClean)
	ctx.Step(`^the API is running$`, tc.theAPIIsRunning)

	// --- HTTP actions ---
	ctx.Step(`^I send a (GET|POST|PUT|PATCH|DELETE) request to "([^"]*)"$`, tc.iSendRequest)
	ctx.Step(`^I send a (GET|POST|PUT|PATCH|DELETE) request to "([^"]*)" with body:$`, tc.iSendRequestWithBody)

	// --- HTTP assertions ---
	ctx.Step(`^the response status is (\d+)$`, tc.theResponseStatusIs)
	ctx.Step(`^the response body contains "([^"]*)"$`, tc.theResponseBodyContains)
	ctx.Step(`^the response body field "([^"]*)" is "([^"]*)"$`, tc.theResponseBodyFieldIs)
}

// theDatabaseIsClean deletes all rows from all tables in reverse FK order.
// SQLite does not support TRUNCATE; DELETE FROM is used instead.
func (tc *TestContext) theDatabaseIsClean() error {
	// FK order: messages depends on sessions; config_entries is independent.
	tables := []string{"messages", "sessions", "config_entries"}
	for _, t := range tables {
		if _, err := tc.DB.Exec("DELETE FROM " + t); err != nil {
			return fmt.Errorf("clean table %q: %w", t, err)
		}
	}
	return nil
}

// theAPIIsRunning asserts that GET /health returns 200.
func (tc *TestContext) theAPIIsRunning() error {
	resp, err := tc.HTTPClient.Get(tc.BaseURL + "/health")
	if err != nil {
		return fmt.Errorf("API health check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected /health 200, got %d", resp.StatusCode)
	}
	return nil
}

// iSendRequest sends a request without a body.
func (tc *TestContext) iSendRequest(method, path string) error {
	req, err := http.NewRequestWithContext(context.Background(), method, tc.BaseURL+path, nil)
	if err != nil {
		return err
	}
	return tc.doRequest(req)
}

// iSendRequestWithBody sends a request with a JSON body from a DocString.
func (tc *TestContext) iSendRequestWithBody(method, path string, body *godog.DocString) error {
	req, err := http.NewRequestWithContext(
		context.Background(), method, tc.BaseURL+path,
		bytes.NewBufferString(body.Content),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return tc.doRequest(req)
}

func (tc *TestContext) doRequest(req *http.Request) error {
	resp, err := tc.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("http %s %s: %w", req.Method, req.URL, err)
	}
	tc.LastResponse = resp
	tc.LastBody, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	return err
}

// theResponseStatusIs asserts the HTTP status code of the last response.
func (tc *TestContext) theResponseStatusIs(expected int) error {
	if tc.LastResponse == nil {
		return fmt.Errorf("no HTTP response recorded")
	}
	if tc.LastResponse.StatusCode != expected {
		return fmt.Errorf("expected status %d, got %d — body: %s",
			expected, tc.LastResponse.StatusCode, string(tc.LastBody))
	}
	return nil
}

// theResponseBodyContains asserts that the raw response body contains a substring.
func (tc *TestContext) theResponseBodyContains(substring string) error {
	if tc.LastBody == nil {
		return fmt.Errorf("no response body recorded")
	}
	if !strings.Contains(string(tc.LastBody), substring) {
		return fmt.Errorf("response body does not contain %q — body: %s", substring, string(tc.LastBody))
	}
	return nil
}

// theResponseBodyFieldIs asserts a top-level JSON field equals an expected string value.
func (tc *TestContext) theResponseBodyFieldIs(field, expected string) error {
	var parsed map[string]interface{}
	if err := json.Unmarshal(tc.LastBody, &parsed); err != nil {
		return fmt.Errorf("response is not JSON: %w — body: %s", err, string(tc.LastBody))
	}
	actual, ok := parsed[field]
	if !ok {
		return fmt.Errorf("field %q not found in response body", field)
	}
	if fmt.Sprintf("%v", actual) != expected {
		return fmt.Errorf("expected field %q = %q, got %q", field, expected, fmt.Sprintf("%v", actual))
	}
	return nil
}
