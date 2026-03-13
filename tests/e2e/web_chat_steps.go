package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
)

// RegisterWebChatSteps binds all step definitions for the web chat feature.
func RegisterWebChatSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	// --- Preconditions ---
	ctx.Step(`^a web chat session exists$`, tc.aWebChatSessionExists)
	ctx.Step(`^a web chat session exists with title "([^"]*)"$`, tc.aWebChatSessionExistsWithTitle)
	ctx.Step(`^I store the chat session ID as "([^"]*)"$`, tc.iStoreTheChatSessionIDAs)
	ctx.Step(`^I switch to stored session "([^"]*)"$`, tc.iSwitchToStoredSession)

	// --- Actions ---
	ctx.Step(`^I send a chat message "([^"]*)"$`, tc.iSendAChatMessage)
	ctx.Step(`^I send a GET request to the chat session$`, tc.iSendGETToTheChatSession)
	ctx.Step(`^I send a DELETE request to the chat session$`, tc.iSendDELETEToTheChatSession)
	ctx.Step(`^I send a GET request to the chat session messages$`, tc.iSendGETToTheChatSessionMessages)
	ctx.Step(`^I send a GET request to the discord session via chat endpoint$`, tc.iSendGETToDiscordSessionViaChatEndpoint)
	ctx.Step(`^I send a DELETE request to the discord session via chat endpoint$`, tc.iSendDELETEToDiscordSessionViaChatEndpoint)

	// --- Assertions ---
	ctx.Step(`^the response body is a JSON array of length (\d+)$`, tc.theResponseBodyIsJSONArrayOfLength)
	ctx.Step(`^the response body is a JSON array$`, tc.theResponseBodyIsJSONArray)
	ctx.Step(`^every session in the response has connector_type "([^"]*)"$`, tc.everySessionHasConnectorType)
	ctx.Step(`^the assistant response role is "([^"]*)"$`, tc.theAssistantResponseRoleIs)
	ctx.Step(`^the assistant response content is not empty$`, tc.theAssistantResponseContentIsNotEmpty)
	ctx.Step(`^the session title is "([^"]*)"$`, tc.theSessionTitleIs)
	ctx.Step(`^the session title starts with "([^"]*)"$`, tc.theSessionTitleStartsWith)
	ctx.Step(`^the first session in the list has title "([^"]*)"$`, tc.theFirstSessionInTheListHasTitle)
	ctx.Step(`^the chat session has (\d+) messages$`, tc.theChatSessionHasNMessages)

	// --- DB assertions ---
	ctx.Step(`^the database contains a user message "([^"]*)" in the chat session$`, tc.theDBContainsUserMessageInChatSession)
	ctx.Step(`^the database contains an assistant message in the chat session$`, tc.theDBContainsAssistantMessageInChatSession)
	ctx.Step(`^the database has no messages for the deleted chat session$`, tc.theDBHasNoMessagesForDeletedChatSession)
}

// ---------------------------------------------------------------------------
// Precondition steps
// ---------------------------------------------------------------------------

func (tc *TestContext) aWebChatSessionExists() error {
	return tc.createWebChatSession("")
}

func (tc *TestContext) aWebChatSessionExistsWithTitle(title string) error {
	return tc.createWebChatSession(title)
}

func (tc *TestContext) createWebChatSession(title string) error {
	body := fmt.Sprintf(`{"title": "%s"}`, title)
	req, err := http.NewRequest("POST", tc.BaseURL+"/api/v1/chat/sessions", bytes.NewBufferString(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tc.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("create web chat session: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("expected 201 creating chat session, got %d: %s", resp.StatusCode, string(respBody))
	}

	var session map[string]interface{}
	if err := json.Unmarshal(respBody, &session); err != nil {
		return fmt.Errorf("parse chat session response: %w", err)
	}

	sessionID, ok := session["id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("chat session response missing id")
	}

	tc.ScenarioData["chat_session_id"] = sessionID
	tc.ScenarioData["session_id"] = sessionID
	return nil
}

func (tc *TestContext) iStoreTheChatSessionIDAs(key string) error {
	id, ok := tc.ScenarioData["chat_session_id"].(string)
	if !ok || id == "" {
		return fmt.Errorf("no chat_session_id in ScenarioData")
	}
	tc.ScenarioData["stored_session_"+key] = id
	return nil
}

func (tc *TestContext) iSwitchToStoredSession(key string) error {
	id, ok := tc.ScenarioData["stored_session_"+key].(string)
	if !ok || id == "" {
		return fmt.Errorf("no stored session with key %q", key)
	}
	tc.ScenarioData["chat_session_id"] = id
	tc.ScenarioData["session_id"] = id
	return nil
}

// ---------------------------------------------------------------------------
// Action steps
// ---------------------------------------------------------------------------

func (tc *TestContext) iSendAChatMessage(content string) error {
	sessionID, ok := tc.ScenarioData["chat_session_id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("no chat_session_id in ScenarioData")
	}

	body := fmt.Sprintf(`{"content": "%s"}`, content)
	req, err := http.NewRequest("POST",
		tc.BaseURL+"/api/v1/chat/sessions/"+sessionID+"/messages",
		bytes.NewBufferString(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	return tc.doRequest(req)
}

func (tc *TestContext) iSendGETToTheChatSession() error {
	sessionID, ok := tc.ScenarioData["chat_session_id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("no chat_session_id in ScenarioData")
	}
	return tc.iSendRequest("GET", "/api/v1/chat/sessions/"+sessionID)
}

func (tc *TestContext) iSendDELETEToTheChatSession() error {
	sessionID, ok := tc.ScenarioData["chat_session_id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("no chat_session_id in ScenarioData")
	}
	// Store for post-delete assertions.
	tc.ScenarioData["deleted_chat_session_id"] = sessionID
	return tc.iSendRequest("DELETE", "/api/v1/chat/sessions/"+sessionID)
}

func (tc *TestContext) iSendGETToTheChatSessionMessages() error {
	sessionID, ok := tc.ScenarioData["chat_session_id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("no chat_session_id in ScenarioData")
	}
	return tc.iSendRequest("GET", "/api/v1/chat/sessions/"+sessionID+"/messages")
}

func (tc *TestContext) iSendGETToDiscordSessionViaChatEndpoint() error {
	sessionID, ok := tc.ScenarioData["session_id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("no session_id in ScenarioData")
	}
	return tc.iSendRequest("GET", "/api/v1/chat/sessions/"+sessionID)
}

func (tc *TestContext) iSendDELETEToDiscordSessionViaChatEndpoint() error {
	sessionID, ok := tc.ScenarioData["session_id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("no session_id in ScenarioData")
	}
	return tc.iSendRequest("DELETE", "/api/v1/chat/sessions/"+sessionID)
}

// ---------------------------------------------------------------------------
// Assertion steps
// ---------------------------------------------------------------------------

func (tc *TestContext) theResponseBodyIsJSONArrayOfLength(expected int) error {
	var arr []interface{}
	if err := json.Unmarshal(tc.LastBody, &arr); err != nil {
		return fmt.Errorf("response is not a JSON array: %w — body: %s", err, string(tc.LastBody))
	}
	if len(arr) != expected {
		return fmt.Errorf("expected JSON array of length %d, got %d — body: %s",
			expected, len(arr), string(tc.LastBody))
	}
	return nil
}

func (tc *TestContext) theResponseBodyIsJSONArray() error {
	var arr []interface{}
	if err := json.Unmarshal(tc.LastBody, &arr); err != nil {
		return fmt.Errorf("response is not a JSON array: %w — body: %s", err, string(tc.LastBody))
	}
	return nil
}

func (tc *TestContext) everySessionHasConnectorType(expected string) error {
	var arr []map[string]interface{}
	if err := json.Unmarshal(tc.LastBody, &arr); err != nil {
		return fmt.Errorf("response is not a JSON array of objects: %w", err)
	}
	for i, obj := range arr {
		ct, _ := obj["connector_type"].(string)
		if ct != expected {
			return fmt.Errorf("session[%d] has connector_type %q, expected %q", i, ct, expected)
		}
	}
	return nil
}

func (tc *TestContext) theAssistantResponseRoleIs(expected string) error {
	var parsed map[string]interface{}
	if err := json.Unmarshal(tc.LastBody, &parsed); err != nil {
		return fmt.Errorf("response is not JSON: %w", err)
	}
	msg, ok := parsed["message"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("response missing 'message' object — body: %s", string(tc.LastBody))
	}
	role, _ := msg["role"].(string)
	if role != expected {
		return fmt.Errorf("expected message role %q, got %q", expected, role)
	}
	return nil
}

func (tc *TestContext) theAssistantResponseContentIsNotEmpty() error {
	var parsed map[string]interface{}
	if err := json.Unmarshal(tc.LastBody, &parsed); err != nil {
		return fmt.Errorf("response is not JSON: %w", err)
	}
	msg, ok := parsed["message"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("response missing 'message' object")
	}
	content, _ := msg["content"].(string)
	if content == "" {
		return fmt.Errorf("expected non-empty assistant content")
	}
	return nil
}

func (tc *TestContext) theSessionTitleIs(expected string) error {
	var parsed map[string]interface{}
	if err := json.Unmarshal(tc.LastBody, &parsed); err != nil {
		return fmt.Errorf("response is not JSON: %w", err)
	}
	session, ok := parsed["session"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("response missing 'session' object — body: %s", string(tc.LastBody))
	}
	title, _ := session["title"].(string)
	if title != expected {
		return fmt.Errorf("expected session title %q, got %q", expected, title)
	}
	return nil
}

func (tc *TestContext) theSessionTitleStartsWith(prefix string) error {
	var parsed map[string]interface{}
	if err := json.Unmarshal(tc.LastBody, &parsed); err != nil {
		return fmt.Errorf("response is not JSON: %w", err)
	}
	session, ok := parsed["session"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("response missing 'session' object — body: %s", string(tc.LastBody))
	}
	title, _ := session["title"].(string)
	if !strings.HasPrefix(title, prefix) {
		return fmt.Errorf("expected session title to start with %q, got %q", prefix, title)
	}
	return nil
}

func (tc *TestContext) theFirstSessionInTheListHasTitle(expected string) error {
	var arr []map[string]interface{}
	if err := json.Unmarshal(tc.LastBody, &arr); err != nil {
		return fmt.Errorf("response is not a JSON array: %w", err)
	}
	if len(arr) == 0 {
		return fmt.Errorf("expected non-empty session list")
	}
	title, _ := arr[0]["title"].(string)
	if title != expected {
		return fmt.Errorf("expected first session title %q, got %q", expected, title)
	}
	return nil
}

func (tc *TestContext) theChatSessionHasNMessages(expected int) error {
	var parsed map[string]interface{}
	if err := json.Unmarshal(tc.LastBody, &parsed); err != nil {
		return fmt.Errorf("response is not JSON: %w", err)
	}
	messages, ok := parsed["messages"].([]interface{})
	if !ok {
		return fmt.Errorf("response missing 'messages' array — body: %s", string(tc.LastBody))
	}
	if len(messages) != expected {
		return fmt.Errorf("expected %d messages, got %d", expected, len(messages))
	}
	return nil
}

// ---------------------------------------------------------------------------
// DB assertion steps
// ---------------------------------------------------------------------------

func (tc *TestContext) theDBContainsUserMessageInChatSession(content string) error {
	sessionID, ok := tc.ScenarioData["chat_session_id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("no chat_session_id in ScenarioData")
	}
	var count int
	err := tc.DB.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE session_id = ? AND role = 'user' AND content = ?`,
		sessionID, content,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("query user messages: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("expected user message %q in session %s but found none", content, sessionID)
	}
	return nil
}

func (tc *TestContext) theDBContainsAssistantMessageInChatSession() error {
	sessionID, ok := tc.ScenarioData["chat_session_id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("no chat_session_id in ScenarioData")
	}
	var count int
	err := tc.DB.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE session_id = ? AND role = 'assistant'`,
		sessionID,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("query assistant messages: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("expected assistant message in session %s but found none", sessionID)
	}
	return nil
}

func (tc *TestContext) theDBHasNoMessagesForDeletedChatSession() error {
	sessionID, ok := tc.ScenarioData["deleted_chat_session_id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("no deleted_chat_session_id in ScenarioData")
	}
	var count int
	err := tc.DB.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE session_id = ?`,
		sessionID,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("query messages for deleted session: %w", err)
	}
	if count != 0 {
		return fmt.Errorf("expected 0 messages for deleted session %s, got %d", sessionID, count)
	}
	return nil
}
