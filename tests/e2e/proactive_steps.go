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
	"github.com/google/uuid"

	"bruce/internal/ai"
	"bruce/internal/config"
	"bruce/internal/domain"
	"bruce/internal/repository"
	"bruce/internal/tools/proactive"
)

// RegisterProactiveSteps registers Gherkin step definitions for proactive intelligence,
// ambient watches, and scheduled reports.
func RegisterProactiveSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	// --- Preconditions ---
	ctx.Step(`^a proactive cron task exists with title "([^"]*)" and schedule "([^"]*)"$`, tc.aProactiveCronTaskExists)
	ctx.Step(`^an ambient watch task exists with title "([^"]*)" and interval "([^"]*)"$`, tc.anAmbientWatchTaskExists)
	ctx.Step(`^a proactive task exists for that session$`, tc.aProactiveTaskExistsForThatSession)
	ctx.Step(`^I pause the proactive task "([^"]*)"$`, tc.iPauseTheProactiveTask)

	// --- Actions ---
	ctx.Step(`^I send a GET request to the proactive task$`, tc.iSendGETToTheProactiveTask)
	ctx.Step(`^I send a PATCH request to the proactive task with body:$`, tc.iSendPATCHToTheProactiveTaskWithBody)
	ctx.Step(`^I send a DELETE request to the proactive task$`, tc.iSendDELETEToTheProactiveTask)
	ctx.Step(`^I trigger an immediate run for the proactive task$`, tc.iTriggerImmediateRunForProactiveTask)
	ctx.Step(`^I delete that session$`, tc.iDeleteThatSession)

	// --- Conversational Tool Execution ---
	ctx.Step(`^I execute the conversational tool "([^"]*)" with parameters:$`, tc.iExecuteConversationalToolWithParams)
	ctx.Step(`^I execute the conversational tool "([^"]*)"$`, tc.iExecuteConversationalTool)
	ctx.Step(`^I execute the conversational tool "([^"]*)" with task "([^"]*)" and action "([^"]*)"$`, tc.iExecuteConversationalToolWithTaskAndAction)
	ctx.Step(`^I execute the conversational tool "([^"]*)" with task "([^"]*)"$`, tc.iExecuteConversationalToolWithTask)

	// --- Assertions ---
	ctx.Step(`^the proactive task has status "([^"]*)"$`, tc.theProactiveTaskHasStatus)
	ctx.Step(`^the proactive task target connector is "([^"]*)" and channel is "([^"]*)"$`, tc.theProactiveTaskTargetIs)
	ctx.Step(`^the tool execution succeeds with confirmation$`, tc.theToolExecutionSucceedsWithConfirmation)
	ctx.Step(`^the tool output contains "([^"]*)"$`, tc.theToolOutputContains)
	ctx.Step(`^the database contains a proactive task titled "([^"]*)" with type "([^"]*)"$`, tc.theDBContainsProactiveTaskTitledWithType)
	ctx.Step(`^the database contains (\d+) proactive tasks$`, tc.theDBContainsNProactiveTasks)
	ctx.Step(`^the database has no proactive tasks for the deleted session$`, tc.theDBHasNoProactiveTasksForDeletedSession)
}

// ---------------------------------------------------------------------------
// Preconditions
// ---------------------------------------------------------------------------

func (tc *TestContext) aProactiveCronTaskExists(title, schedule string) error {
	sessionID, err := tc.ensureTestSession("web", "default-test-channel")
	if err != nil {
		return err
	}

	payload := fmt.Sprintf(`{
		"session_id": "%s",
		"connector_type": "web",
		"channel_id": "default-test-channel",
		"target_connector": "web",
		"target_channel_id": "default-test-channel",
		"title": "%s",
		"task_type": "cron",
		"schedule_expr": "%s",
		"prompt_condition": "Generate scheduled report for %s"
	}`, sessionID, title, schedule, title)

	return tc.createProactiveTaskFromJSON(payload)
}

func (tc *TestContext) anAmbientWatchTaskExists(title, interval string) error {
	sessionID, err := tc.ensureTestSession("web", "default-test-channel")
	if err != nil {
		return err
	}

	payload := fmt.Sprintf(`{
		"session_id": "%s",
		"connector_type": "web",
		"channel_id": "default-test-channel",
		"target_connector": "web",
		"target_channel_id": "default-test-channel",
		"title": "%s",
		"task_type": "watch",
		"schedule_expr": "%s",
		"prompt_condition": "Ambient condition for %s",
		"target_tools": ["gmail_search"]
	}`, sessionID, title, interval, title)

	return tc.createProactiveTaskFromJSON(payload)
}

func (tc *TestContext) aProactiveTaskExistsForThatSession() error {
	sessionID, ok := tc.ScenarioData["session_id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("no session_id in ScenarioData")
	}
	connector, _ := tc.ScenarioData["connector_type"].(string)
	if connector == "" {
		connector = "discord"
	}
	channel, _ := tc.ScenarioData["channel_id"].(string)
	if channel == "" {
		channel = "cascade-chan-1"
	}

	payload := fmt.Sprintf(`{
		"session_id": "%s",
		"connector_type": "%s",
		"channel_id": "%s",
		"target_connector": "%s",
		"target_channel_id": "%s",
		"title": "Cascaded Task",
		"task_type": "cron",
		"schedule_expr": "0 9 * * 1-5",
		"prompt_condition": "Report prompt"
	}`, sessionID, connector, channel, connector, channel)

	return tc.createProactiveTaskFromJSON(payload)
}

func (tc *TestContext) iPauseTheProactiveTask(title string) error {
	var id string
	err := tc.DB.QueryRow(`SELECT id FROM proactive_tasks WHERE title = ?`, title).Scan(&id)
	if err != nil {
		return fmt.Errorf("find proactive task by title %q: %w", title, err)
	}

	req, err := http.NewRequest(
		http.MethodPatch,
		tc.BaseURL+"/api/v1/proactive-tasks/"+id,
		bytes.NewBufferString(`{"is_active": false}`),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return tc.doRequest(req)
}

func (tc *TestContext) ensureTestSession(connector, channel string) (string, error) {
	row := tc.DB.QueryRow(`SELECT id FROM sessions WHERE connector_type = ? AND channel_id = ?`, connector, channel)
	var id string
	if err := row.Scan(&id); err == nil {
		return id, nil
	}

	newID := uuid.New().String()
	_, err := tc.DB.Exec(
		`INSERT INTO sessions (id, connector_type, channel_id, is_active) VALUES (?, ?, ?, 1)`,
		newID, connector, channel,
	)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	return newID, nil
}

func (tc *TestContext) createProactiveTaskFromJSON(bodyJSON string) error {
	req, err := http.NewRequest(http.MethodPost, tc.BaseURL+"/api/v1/proactive-tasks", bytes.NewBufferString(bodyJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tc.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST /api/v1/proactive-tasks: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("expected 201 creating proactive task, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var task map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &task); err != nil {
		return fmt.Errorf("unmarshal proactive task: %w", err)
	}

	taskID, _ := task["id"].(string)
	tc.ScenarioData["task_id"] = taskID
	tc.ScenarioData["proactive_task_id"] = taskID
	return nil
}

// ---------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------

func (tc *TestContext) iSendGETToTheProactiveTask() error {
	taskID := tc.resolveCurrentTaskID()
	if taskID == "" {
		return fmt.Errorf("no proactive task id in context")
	}
	return tc.iSendRequest(http.MethodGet, "/api/v1/proactive-tasks/"+taskID)
}

func (tc *TestContext) iSendPATCHToTheProactiveTaskWithBody(body *godog.DocString) error {
	taskID := tc.resolveCurrentTaskID()
	if taskID == "" {
		return fmt.Errorf("no proactive task id in context")
	}
	return tc.iSendRequestWithBody(http.MethodPatch, "/api/v1/proactive-tasks/"+taskID, body)
}

func (tc *TestContext) iSendDELETEToTheProactiveTask() error {
	taskID := tc.resolveCurrentTaskID()
	if taskID == "" {
		return fmt.Errorf("no proactive task id in context")
	}
	return tc.iSendRequest(http.MethodDelete, "/api/v1/proactive-tasks/"+taskID)
}

func (tc *TestContext) iTriggerImmediateRunForProactiveTask() error {
	taskID := tc.resolveCurrentTaskID()
	if taskID == "" {
		return fmt.Errorf("no proactive task id in context")
	}
	return tc.iSendRequest(http.MethodPost, "/api/v1/proactive-tasks/"+taskID+"/run")
}

func (tc *TestContext) iDeleteThatSession() error {
	sessionID, ok := tc.ScenarioData["session_id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("no session_id in ScenarioData")
	}
	tc.ScenarioData["deleted_session_id"] = sessionID
	_, err := tc.DB.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID)
	return err
}

func (tc *TestContext) resolveCurrentTaskID() string {
	if id, ok := tc.ScenarioData["proactive_task_id"].(string); ok && id != "" {
		return id
	}
	if id, ok := tc.ScenarioData["task_id"].(string); ok && id != "" {
		return id
	}

	// Try extracting from last response body if present
	if tc.LastBody != nil {
		var parsed map[string]interface{}
		if err := json.Unmarshal(tc.LastBody, &parsed); err == nil {
			if id, ok := parsed["id"].(string); ok && id != "" {
				tc.ScenarioData["proactive_task_id"] = id
				return id
			}
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Conversational Tool Execution
// ---------------------------------------------------------------------------

func (tc *TestContext) iExecuteConversationalToolWithParams(toolName string, docString *godog.DocString) error {
	var input map[string]interface{}
	if err := json.Unmarshal([]byte(docString.Content), &input); err != nil {
		return fmt.Errorf("invalid JSON tool input: %w", err)
	}

	sessionRepo := repository.NewSessionRepository(tc.DB)
	proactiveRepo := repository.NewProactiveTaskRepository(tc.DB)
	cfg := config.Load()

	// Build context with current session if available
	ctx := context.Background()
	sessionID, _ := tc.ScenarioData["session_id"].(string)
	connector, _ := tc.ScenarioData["connector_type"].(string)
	channel, _ := tc.ScenarioData["channel_id"].(string)
	if sessionID != "" && connector != "" && channel != "" {
		ctx = ai.WithSessionContext(ctx, sessionID, connector, channel)
	}

	switch toolName {
	case "proactive_create":
		tool := proactive.NewCreateTool(proactiveRepo, sessionRepo, cfg)
		output, err := tool.Execute(ctx, input)
		if err != nil {
			return fmt.Errorf("proactive_create failed: %w", err)
		}
		tc.ScenarioData["tool_output"] = fmt.Sprintf("%v", output)
		if title, ok := input["title"].(string); ok && title != "" {
			var id string
			if err := tc.DB.QueryRow(`SELECT id FROM proactive_tasks WHERE title = ? ORDER BY created_at DESC LIMIT 1`, title).Scan(&id); err == nil {
				tc.ScenarioData["proactive_task_id"] = id
				tc.ScenarioData["task_id"] = id
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported conversational tool: %s", toolName)
	}
}

func (tc *TestContext) iExecuteConversationalTool(toolName string) error {
	proactiveRepo := repository.NewProactiveTaskRepository(tc.DB)
	ctx := context.Background()

	switch toolName {
	case "proactive_list":
		tool := proactive.NewListTool(proactiveRepo)
		output, err := tool.Execute(ctx, map[string]interface{}{})
		if err != nil {
			return fmt.Errorf("proactive_list failed: %w", err)
		}
		tc.ScenarioData["tool_output"] = fmt.Sprintf("%v", output)
		return nil
	default:
		return fmt.Errorf("unsupported conversational tool: %s", toolName)
	}
}

func (tc *TestContext) iExecuteConversationalToolWithTaskAndAction(toolName, taskTitle, action string) error {
	proactiveRepo := repository.NewProactiveTaskRepository(tc.DB)
	ctx := context.Background()

	switch toolName {
	case "proactive_toggle":
		tool := proactive.NewToggleTool(proactiveRepo)
		output, err := tool.Execute(ctx, map[string]interface{}{
			"task_id_or_title": taskTitle,
			"action":           action,
		})
		if err != nil {
			return fmt.Errorf("proactive_toggle failed: %w", err)
		}
		tc.ScenarioData["tool_output"] = fmt.Sprintf("%v", output)
		return nil
	default:
		return fmt.Errorf("unsupported conversational tool: %s", toolName)
	}
}

func (tc *TestContext) iExecuteConversationalToolWithTask(toolName, taskTitle string) error {
	proactiveRepo := repository.NewProactiveTaskRepository(tc.DB)
	ctx := context.Background()

	switch toolName {
	case "proactive_delete":
		tool := proactive.NewDeleteTool(proactiveRepo)
		output, err := tool.Execute(ctx, map[string]interface{}{
			"task_id_or_title": taskTitle,
		})
		if err != nil {
			return fmt.Errorf("proactive_delete failed: %w", err)
		}
		tc.ScenarioData["tool_output"] = fmt.Sprintf("%v", output)
		return nil
	default:
		return fmt.Errorf("unsupported conversational tool: %s", toolName)
	}
}

// ---------------------------------------------------------------------------
// Assertions
// ---------------------------------------------------------------------------

func (tc *TestContext) theProactiveTaskHasStatus(expected string) error {
	taskID := tc.resolveCurrentTaskID()
	var isActive bool
	err := tc.DB.QueryRow(`SELECT is_active FROM proactive_tasks WHERE id = ?`, taskID).Scan(&isActive)
	if err != nil {
		return fmt.Errorf("query task %q status: %w", taskID, err)
	}

	switch strings.ToLower(expected) {
	case "active":
		if !isActive {
			return fmt.Errorf("expected task %q to be active, got paused", taskID)
		}
	case "paused":
		if isActive {
			return fmt.Errorf("expected task %q to be paused, got active", taskID)
		}
	default:
		return fmt.Errorf("unknown expected status %q (must be active or paused)", expected)
	}
	return nil
}

func (tc *TestContext) theProactiveTaskTargetIs(expectedConnector, expectedChannelID string) error {
	taskID := tc.resolveCurrentTaskID()
	var targetConnector, targetChannelID string
	err := tc.DB.QueryRow(
		`SELECT target_connector, target_channel_id FROM proactive_tasks WHERE id = ?`,
		taskID,
	).Scan(&targetConnector, &targetChannelID)
	if err != nil {
		return fmt.Errorf("query target for task %q: %w", taskID, err)
	}

	if targetConnector != expectedConnector {
		return fmt.Errorf("expected target_connector %q, got %q", expectedConnector, targetConnector)
	}
	if targetChannelID != expectedChannelID {
		return fmt.Errorf("expected target_channel_id %q, got %q", expectedChannelID, targetChannelID)
	}
	return nil
}

func (tc *TestContext) theToolExecutionSucceedsWithConfirmation() error {
	output, ok := tc.ScenarioData["tool_output"].(string)
	if !ok || strings.TrimSpace(output) == "" {
		return fmt.Errorf("expected non-empty tool output")
	}
	return nil
}

func (tc *TestContext) theToolOutputContains(substring string) error {
	output, ok := tc.ScenarioData["tool_output"].(string)
	if !ok {
		return fmt.Errorf("no tool_output in ScenarioData")
	}
	if !strings.Contains(output, substring) {
		return fmt.Errorf("expected tool output to contain %q, got: %s", substring, output)
	}
	return nil
}

func (tc *TestContext) theDBContainsProactiveTaskTitledWithType(title, taskType string) error {
	var count int
	var id, targetConn, targetChan string
	err := tc.DB.QueryRow(
		`SELECT id, target_connector, target_channel_id FROM proactive_tasks WHERE title = ? AND task_type = ?`,
		title, taskType,
	).Scan(&id, &targetConn, &targetChan)
	if err == nil {
		count = 1
		tc.ScenarioData["proactive_task_id"] = id
		tc.ScenarioData["task_id"] = id
	}

	if count == 0 {
		return fmt.Errorf("expected task titled %q with type %q in DB, found none", title, taskType)
	}
	return nil
}

func (tc *TestContext) theDBContainsNProactiveTasks(expected int) error {
	var count int
	err := tc.DB.QueryRow(`SELECT COUNT(*) FROM proactive_tasks`).Scan(&count)
	if err != nil {
		return fmt.Errorf("query proactive_tasks count: %w", err)
	}
	if count != expected {
		return fmt.Errorf("expected %d proactive tasks in database, got %d", expected, count)
	}
	return nil
}

func (tc *TestContext) theDBHasNoProactiveTasksForDeletedSession() error {
	sessionID, ok := tc.ScenarioData["deleted_session_id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("no deleted_session_id recorded")
	}

	var count int
	err := tc.DB.QueryRow(`SELECT COUNT(*) FROM proactive_tasks WHERE session_id = ?`, sessionID).Scan(&count)
	if err != nil {
		return fmt.Errorf("query proactive tasks for deleted session: %w", err)
	}
	if count != 0 {
		return fmt.Errorf("expected 0 proactive tasks for session %s, found %d", sessionID, count)
	}
	return nil
}

// compile check
var _ = domain.TaskTypeCron
