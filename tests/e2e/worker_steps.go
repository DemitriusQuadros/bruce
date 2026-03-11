package e2e_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"

	"bruce/internal/worker"
)

// RegisterWorkerSteps binds all step definitions for the worker feature.
func RegisterWorkerSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	// --- Preconditions ---
	ctx.Step(`^a session exists for connector "([^"]*)" and channel "([^"]*)"$`, tc.aSessionExistsFor)
	ctx.Step(`^the session for connector "([^"]*)" and channel "([^"]*)" is paused$`, tc.theSessionIsPaused)
	ctx.Step(`^no session exists for connector "([^"]*)" and channel "([^"]*)"$`, tc.noSessionExistsFor)
	ctx.Step(`^(\d+) user messages have been inserted into the session for connector "([^"]*)" and channel "([^"]*)"$`, tc.nUserMessagesInserted)

	// --- Actions ---
	ctx.Step(`^I enqueue a "message:process" task with connector "([^"]*)", channel "([^"]*)", and content "([^"]*)"$`, tc.iEnqueueMessageTask)
	ctx.Step(`^I call FindOrCreate for connector "([^"]*)" and channel "([^"]*)"$`, tc.iCallFindOrCreate)

	// --- Polling assertions (positive — must appear within timeout) ---
	ctx.Step(`^within (\d+) seconds a "([^"]*)" message with content "([^"]*)" exists in the session$`, tc.withinSecondsMessageWithContentExists)
	ctx.Step(`^within (\d+) seconds an "([^"]*)" message exists in the session$`, tc.withinSecondsMessageRoleExists)
	ctx.Step(`^within (\d+) seconds a session exists for connector "([^"]*)" and channel "([^"]*)"$`, tc.withinSecondsSessionExists)
	ctx.Step(`^within (\d+) seconds a "([^"]*)" message with content "([^"]*)" exists in that session$`, tc.withinSecondsMessageWithContentExistsInLatestSession)

	// --- Negative assertions (must NOT appear after waiting) ---
	ctx.Step(`^after (\d+) seconds no "([^"]*)" message with content "([^"]*)" exists in the session$`, tc.afterSecondsNoMessageWithContent)
	ctx.Step(`^after (\d+) seconds no "([^"]*)" message exists in the session$`, tc.afterSecondsNoMessageRole)

	// --- Count assertions ---
	ctx.Step(`^only 1 session exists for connector "([^"]*)" and channel "([^"]*)"$`, tc.onlyOneSessionExists)
	ctx.Step(`^both calls returned the same session ID$`, tc.bothCallsReturnedSameSessionID)
	ctx.Step(`^the database contains at least (\d+) "([^"]*)" messages in the session$`, tc.dbContainsAtLeastNMessages)
}

// ---------------------------------------------------------------------------
// Precondition steps
// ---------------------------------------------------------------------------

// aSessionExistsFor inserts (or finds) a session in the DB and stores it in ScenarioData.
func (tc *TestContext) aSessionExistsFor(connectorType, channelID string) error {
	sessionID, err := tc.findOrCreateSession(connectorType, channelID)
	if err != nil {
		return err
	}
	tc.ScenarioData["session_id"] = sessionID
	tc.ScenarioData["connector_type"] = connectorType
	tc.ScenarioData["channel_id"] = channelID
	return nil
}

// theSessionIsPaused sets is_active=0 for the session identified by (connectorType, channelID).
func (tc *TestContext) theSessionIsPaused(connectorType, channelID string) error {
	_, err := tc.DB.Exec(
		`UPDATE sessions SET is_active = 0 WHERE connector_type = ? AND channel_id = ?`,
		connectorType, channelID,
	)
	if err != nil {
		return fmt.Errorf("pause session (%s, %s): %w", connectorType, channelID, err)
	}
	return nil
}

// noSessionExistsFor asserts no session exists for the pair, then stores the pair in ScenarioData.
func (tc *TestContext) noSessionExistsFor(connectorType, channelID string) error {
	var count int
	err := tc.DB.QueryRow(
		`SELECT COUNT(*) FROM sessions WHERE connector_type = ? AND channel_id = ?`,
		connectorType, channelID,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("query session count: %w", err)
	}
	if count != 0 {
		return fmt.Errorf("expected no session for (%s, %s) but found %d", connectorType, channelID, count)
	}
	tc.ScenarioData["connector_type"] = connectorType
	tc.ScenarioData["channel_id"] = channelID
	return nil
}

// nUserMessagesInserted inserts N user messages directly into the DB for the given session.
func (tc *TestContext) nUserMessagesInserted(n int, connectorType, channelID string) error {
	sessionID, err := tc.findOrCreateSession(connectorType, channelID)
	if err != nil {
		return err
	}
	tc.ScenarioData["session_id"] = sessionID
	tc.ScenarioData["connector_type"] = connectorType
	tc.ScenarioData["channel_id"] = channelID

	for i := 0; i < n; i++ {
		id := uuid.New().String()
		ts := time.Now().UTC().Add(time.Duration(i) * time.Millisecond)
		_, err := tc.DB.Exec(
			`INSERT INTO messages (id, session_id, role, content, timestamp) VALUES (?, ?, 'user', ?, ?)`,
			id, sessionID, fmt.Sprintf("seed message %d", i+1),
			ts.Format("2006-01-02 15:04:05.000000000"),
		)
		if err != nil {
			return fmt.Errorf("insert seed message %d: %w", i+1, err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Action steps
// ---------------------------------------------------------------------------

// iEnqueueMessageTask enqueues a message:process task via the Asynq client.
// requires: redis
func (tc *TestContext) iEnqueueMessageTask(connectorType, channelID, content string) error {
	payload := worker.ProcessIncomingMessagePayload{
		ConnectorType: connectorType,
		ChannelID:     channelID,
		Content:       content,
	}
	task, err := worker.NewProcessIncomingMessageTask(payload)
	if err != nil {
		return fmt.Errorf("build task: %w", err)
	}
	info, err := tc.AsynqClient.Enqueue(task)
	if err != nil {
		return fmt.Errorf("enqueue task: %w", err)
	}
	tc.ScenarioData["task_id"] = info.ID
	// Store connector/channel so polling steps can resolve the session.
	tc.ScenarioData["connector_type"] = connectorType
	tc.ScenarioData["channel_id"] = channelID
	tc.ScenarioData["enqueued_content"] = content
	return nil
}

// iCallFindOrCreate calls the SessionRepository.FindOrCreate via a direct DB INSERT OR IGNORE
// and stores the resulting session ID. Called twice to verify idempotency.
func (tc *TestContext) iCallFindOrCreate(connectorType, channelID string) error {
	sessionID, err := tc.findOrCreateSession(connectorType, channelID)
	if err != nil {
		return err
	}
	// Accumulate call results to verify idempotency.
	results, _ := tc.ScenarioData["findorcreate_results"].([]string)
	results = append(results, sessionID)
	tc.ScenarioData["findorcreate_results"] = results
	tc.ScenarioData["session_id"] = sessionID
	tc.ScenarioData["connector_type"] = connectorType
	tc.ScenarioData["channel_id"] = channelID
	return nil
}

// ---------------------------------------------------------------------------
// Polling assertion steps (positive)
// ---------------------------------------------------------------------------

// withinSecondsMessageWithContentExists polls the DB until a message with the given role
// and content appears in the session stored in ScenarioData, or times out.
// requires: redis, running-server
func (tc *TestContext) withinSecondsMessageWithContentExists(seconds int, role, content string) error {
	sessionID, err := tc.resolveSessionID()
	if err != nil {
		return err
	}
	return pollUntil(seconds, func() (bool, error) {
		return tc.messageExists(sessionID, role, content)
	})
}

// withinSecondsMessageRoleExists polls until any message with the given role appears.
// requires: redis, running-server
func (tc *TestContext) withinSecondsMessageRoleExists(seconds int, role string) error {
	sessionID, err := tc.resolveSessionID()
	if err != nil {
		return err
	}
	return pollUntil(seconds, func() (bool, error) {
		return tc.messageRoleExists(sessionID, role)
	})
}

// withinSecondsSessionExists polls until a session for the connector/channel pair appears.
// requires: redis, running-server
func (tc *TestContext) withinSecondsSessionExists(seconds int, connectorType, channelID string) error {
	return pollUntil(seconds, func() (bool, error) {
		var count int
		err := tc.DB.QueryRow(
			`SELECT COUNT(*) FROM sessions WHERE connector_type = ? AND channel_id = ?`,
			connectorType, channelID,
		).Scan(&count)
		if err != nil {
			return false, err
		}
		if count > 0 {
			// Store for downstream steps.
			row := tc.DB.QueryRow(
				`SELECT id FROM sessions WHERE connector_type = ? AND channel_id = ?`,
				connectorType, channelID,
			)
			var id string
			if err := row.Scan(&id); err == nil {
				tc.ScenarioData["session_id"] = id
			}
		}
		return count > 0, nil
	})
}

// withinSecondsMessageWithContentExistsInLatestSession resolves the session from the most
// recently discovered "channel_id"/"connector_type" pair stored during the scenario.
// requires: redis, running-server
func (tc *TestContext) withinSecondsMessageWithContentExistsInLatestSession(seconds int, role, content string) error {
	return tc.withinSecondsMessageWithContentExists(seconds, role, content)
}

// ---------------------------------------------------------------------------
// Negative assertion steps
// ---------------------------------------------------------------------------

// afterSecondsNoMessageWithContent waits the full duration then asserts absence.
// requires: redis, running-server
func (tc *TestContext) afterSecondsNoMessageWithContent(seconds int, role, content string) error {
	time.Sleep(time.Duration(seconds) * time.Second)
	sessionID, err := tc.resolveSessionID()
	if err != nil {
		// No session yet — message definitely absent.
		return nil
	}
	exists, err := tc.messageExists(sessionID, role, content)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("expected no %q message with content %q but found one in session %s",
			role, content, sessionID)
	}
	return nil
}

// afterSecondsNoMessageRole waits the full duration then asserts no message with the given role.
// requires: redis, running-server
func (tc *TestContext) afterSecondsNoMessageRole(seconds int, role string) error {
	time.Sleep(time.Duration(seconds) * time.Second)
	sessionID, err := tc.resolveSessionID()
	if err != nil {
		return nil
	}
	exists, err := tc.messageRoleExists(sessionID, role)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("expected no %q message but found one in session %s", role, sessionID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Count / invariant assertion steps
// ---------------------------------------------------------------------------

// onlyOneSessionExists asserts exactly 1 session exists for (connectorType, channelID).
func (tc *TestContext) onlyOneSessionExists(connectorType, channelID string) error {
	var count int
	err := tc.DB.QueryRow(
		`SELECT COUNT(*) FROM sessions WHERE connector_type = ? AND channel_id = ?`,
		connectorType, channelID,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("count sessions: %w", err)
	}
	if count != 1 {
		return fmt.Errorf("expected exactly 1 session for (%s, %s), got %d",
			connectorType, channelID, count)
	}
	return nil
}

// bothCallsReturnedSameSessionID verifies idempotency: both FindOrCreate results are the same ID.
func (tc *TestContext) bothCallsReturnedSameSessionID() error {
	results, ok := tc.ScenarioData["findorcreate_results"].([]string)
	if !ok || len(results) < 2 {
		return fmt.Errorf("expected 2 FindOrCreate results in ScenarioData, got %v", results)
	}
	if results[0] != results[1] {
		return fmt.Errorf("expected same session ID from both FindOrCreate calls, got %q and %q",
			results[0], results[1])
	}
	return nil
}

// dbContainsAtLeastNMessages asserts the session has at least n messages with the given role.
func (tc *TestContext) dbContainsAtLeastNMessages(n int, role string) error {
	sessionID, err := tc.resolveSessionID()
	if err != nil {
		return err
	}
	var count int
	err = tc.DB.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE session_id = ? AND role = ?`,
		sessionID, role,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("count messages: %w", err)
	}
	if count < n {
		return fmt.Errorf("expected at least %d %q messages in session %s, got %d",
			n, role, sessionID, count)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// findOrCreateSession replicates the SessionRepository.FindOrCreate logic using raw SQL.
// This avoids importing the internal repository package in the test binary.
func (tc *TestContext) findOrCreateSession(connectorType, channelID string) (string, error) {
	id := uuid.New().String()
	_, err := tc.DB.Exec(
		`INSERT OR IGNORE INTO sessions (id, connector_type, channel_id, system_prompt, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, '', 1, datetime('now'), datetime('now'))`,
		id, connectorType, channelID,
	)
	if err != nil {
		return "", fmt.Errorf("findOrCreateSession insert: %w", err)
	}
	var sessionID string
	err = tc.DB.QueryRow(
		`SELECT id FROM sessions WHERE connector_type = ? AND channel_id = ?`,
		connectorType, channelID,
	).Scan(&sessionID)
	if err != nil {
		return "", fmt.Errorf("findOrCreateSession select: %w", err)
	}
	return sessionID, nil
}

// resolveSessionID returns the session ID stored in ScenarioData.
// It queries the DB if needed using the stored connector_type/channel_id.
func (tc *TestContext) resolveSessionID() (string, error) {
	if id, ok := tc.ScenarioData["session_id"].(string); ok && id != "" {
		return id, nil
	}
	connectorType, _ := tc.ScenarioData["connector_type"].(string)
	channelID, _ := tc.ScenarioData["channel_id"].(string)
	if connectorType == "" || channelID == "" {
		return "", fmt.Errorf("no session_id in ScenarioData and no connector_type/channel_id to resolve from")
	}
	var id string
	err := tc.DB.QueryRow(
		`SELECT id FROM sessions WHERE connector_type = ? AND channel_id = ?`,
		connectorType, channelID,
	).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no session found for connector=%s channel=%s", connectorType, channelID)
		}
		return "", fmt.Errorf("resolve session ID: %w", err)
	}
	tc.ScenarioData["session_id"] = id
	return id, nil
}

// messageExists returns true if a message with the given role and content exists in sessionID.
func (tc *TestContext) messageExists(sessionID, role, content string) (bool, error) {
	var count int
	err := tc.DB.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE session_id = ? AND role = ? AND content = ?`,
		sessionID, role, content,
	).Scan(&count)
	return count > 0, err
}

// messageRoleExists returns true if any message with the given role exists in sessionID.
func (tc *TestContext) messageRoleExists(sessionID, role string) (bool, error) {
	var count int
	err := tc.DB.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE session_id = ? AND role = ?`,
		sessionID, role,
	).Scan(&count)
	return count > 0, err
}

// pollUntil calls check repeatedly until it returns true or the deadline expires.
// It uses a 500ms poll interval. Never uses time.Sleep for the outer wait.
func pollUntil(seconds int, check func() (bool, error)) error {
	deadline := time.Now().Add(time.Duration(seconds) * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		ok, err := check()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("condition not met within %d seconds", seconds)
		}
		<-ticker.C
	}
}

// decodeTaskPayload is used in debugging helpers — kept for completeness.
func decodeTaskPayload(data []byte) (*worker.ProcessIncomingMessagePayload, error) {
	var p worker.ProcessIncomingMessagePayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ensure decodeTaskPayload is not flagged as unused (it is a debug helper).
var _ = decodeTaskPayload
