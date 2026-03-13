Feature: Web Chat Interface
  As a developer using Bruce
  I want to interact with the LLM through a web chat interface
  So that I can test LLM integration locally without configuring external connectors

  Background:
    Given the database is clean
    And the API is running

  # --- Session CRUD: Happy Path ---

  @smoke @REQ-CHAT-001
  Scenario: Create a new web chat session
    When I send a POST request to "/api/v1/chat/sessions" with body:
      """
      {}
      """
    Then the response status is 201
    And the response body field "connector_type" is "web"
    And the response body field "is_active" is "true"
    And the response body field "title" is ""
    And the response body contains "id"

  @smoke @REQ-CHAT-001
  Scenario: Create a web chat session with a title
    When I send a POST request to "/api/v1/chat/sessions" with body:
      """
      {"title": "My Test Chat"}
      """
    Then the response status is 201
    And the response body field "title" is "My Test Chat"
    And the response body field "connector_type" is "web"

  @smoke @REQ-CHAT-002
  Scenario: List web chat sessions returns only web sessions
    Given a web chat session exists
    And a session exists for connector "discord" and channel "e2e-web-filter-001"
    When I send a GET request to "/api/v1/chat/sessions"
    Then the response status is 200
    And the response body is a JSON array of length 1
    And every session in the response has connector_type "web"

  @smoke @REQ-CHAT-003
  Scenario: Get a web chat session with messages
    Given a web chat session exists
    When I send a GET request to the chat session
    Then the response status is 200
    And the response body contains "session"
    And the response body contains "messages"

  @smoke @REQ-CHAT-004
  Scenario: Delete a web chat session
    Given a web chat session exists
    When I send a DELETE request to the chat session
    Then the response status is 204
    When I send a GET request to the chat session
    Then the response status is 404

  # --- Send Message: Happy Path ---

  @smoke @REQ-CHAT-005
  Scenario: Send a message and receive an assistant response with provider name
    Given a web chat session exists
    When I send a chat message "Hello, Bruce!"
    Then the response status is 200
    And the response body contains "message"
    And the response body contains "session"
    And the response body contains "provider"
    And the assistant response role is "assistant"
    And the assistant response content is not empty

  @smoke @REQ-CHAT-006
  Scenario: First message auto-generates session title
    Given a web chat session exists
    When I send a chat message "How do I deploy a Go application to production?"
    Then the response status is 200
    And the session title is "How do I deploy a Go application to production?"

  @smoke @REQ-CHAT-006
  Scenario: Title is truncated to 50 characters for long messages
    Given a web chat session exists
    When I send a chat message "This is a very long message that should be truncated to fifty characters because it exceeds the limit"
    Then the response status is 200
    And the session title starts with "This is a very long message that should be trunca"

  @smoke @REQ-CHAT-007
  Scenario: Messages are persisted in the database
    Given a web chat session exists
    When I send a chat message "Persistent message"
    Then the response status is 200
    And the database contains a user message "Persistent message" in the chat session
    And the database contains an assistant message in the chat session

  # --- Get Messages ---

  @regression @REQ-CHAT-008
  Scenario: Get messages for a web chat session
    Given a web chat session exists
    And I send a chat message "First message"
    When I send a GET request to the chat session messages
    Then the response status is 200
    And the response body is a JSON array of length 2

  # --- Error Cases ---

  @error-case @REQ-CHAT-009
  Scenario: Send message with empty content returns 400
    Given a web chat session exists
    When I send a chat message ""
    Then the response status is 400
    And the response body contains "content is required"

  @error-case @REQ-CHAT-010
  Scenario: Get non-existent chat session returns 404
    When I send a GET request to "/api/v1/chat/sessions/nonexistent-id-12345"
    Then the response status is 404

  @error-case @REQ-CHAT-011
  Scenario: Delete non-existent chat session returns 404
    When I send a DELETE request to "/api/v1/chat/sessions/nonexistent-id-12345"
    Then the response status is 404

  @error-case @REQ-CHAT-012
  Scenario: Access a non-web session through chat endpoints returns 404
    Given a session exists for connector "discord" and channel "e2e-web-xss-001"
    When I send a GET request to the discord session via chat endpoint
    Then the response status is 404

  @error-case @REQ-CHAT-013
  Scenario: Delete a non-web session through chat endpoints returns 404
    Given a session exists for connector "discord" and channel "e2e-web-del-001"
    When I send a DELETE request to the discord session via chat endpoint
    Then the response status is 404

  @error-case @REQ-CHAT-014
  Scenario: Send message to non-existent session returns 404
    When I send a POST request to "/api/v1/chat/sessions/nonexistent-id-12345/messages" with body:
      """
      {"content": "hello"}
      """
    Then the response status is 404

  # --- Edge Cases ---

  @edge-case @REQ-CHAT-015
  Scenario: Multiple chat sessions are independent
    Given a web chat session exists
    And I store the chat session ID as "session_1"
    And a web chat session exists
    And I store the chat session ID as "session_2"
    When I send a chat message "Message for session 2"
    Then the response status is 200
    When I switch to stored session "session_1"
    And I send a GET request to the chat session
    Then the response status is 200
    And the chat session has 0 messages

  @edge-case @REQ-CHAT-016
  Scenario: Second message does not overwrite existing title
    Given a web chat session exists
    When I send a chat message "First message sets title"
    And I send a chat message "Second message should not change title"
    Then the session title is "First message sets title"

  @edge-case @REQ-CHAT-017
  Scenario: List returns sessions ordered by updated_at DESC
    Given a web chat session exists with title "Older Chat"
    And a web chat session exists with title "Newer Chat"
    When I send a GET request to "/api/v1/chat/sessions"
    Then the response status is 200
    And the first session in the list has title "Newer Chat"

  # --- Invariants ---

  @invariant @REQ-CHAT-018
  Scenario: Deleting a session cascades to messages
    Given a web chat session exists
    And I send a chat message "Will be deleted"
    When I send a DELETE request to the chat session
    Then the response status is 204
    And the database has no messages for the deleted chat session

  @invariant @REQ-CHAT-019
  Scenario: General sessions endpoint still works with web sessions present
    Given a web chat session exists
    When I send a GET request to "/api/v1/sessions"
    Then the response status is 200
    And the response body is a JSON array
