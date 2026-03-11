Feature: Worker Processor Pipeline
  As a user of Bruce
  I want messages enqueued via Asynq to be processed end-to-end
  So that user messages are stored and Claude generates assistant responses persisted in the database

  Background:
    Given the database is clean

  # --- Happy Path ---

  @smoke @REQ-001
  Scenario: Enqueue a message and both user and assistant messages appear in the database
    # requires: redis, running-server
    Given a session exists for connector "discord" and channel "e2e-chan-001"
    When I enqueue a "message:process" task with connector "discord", channel "e2e-chan-001", and content "hello Bruce"
    Then within 30 seconds a "user" message with content "hello Bruce" exists in the session
    And within 30 seconds an "assistant" message exists in the session

  # --- Session auto-creation ---

  @smoke @REQ-002
  Scenario: First message for a new channel auto-creates a session
    # requires: redis, running-server
    When I enqueue a "message:process" task with connector "discord", channel "e2e-chan-new-001", and content "first message"
    Then within 30 seconds a session exists for connector "discord" and channel "e2e-chan-new-001"
    And within 30 seconds a "user" message with content "first message" exists in that session

  # --- Session reuse ---

  @regression @REQ-002
  Scenario: Second message in same channel reuses the existing session
    # requires: redis, running-server
    Given a session exists for connector "whatsapp" and channel "+5511900000001"
    When I enqueue a "message:process" task with connector "whatsapp", channel "+5511900000001", and content "first"
    Then within 30 seconds a "user" message with content "first" exists in the session
    When I enqueue a "message:process" task with connector "whatsapp", channel "+5511900000001", and content "second"
    Then within 30 seconds a "user" message with content "second" exists in the session
    And only 1 session exists for connector "whatsapp" and channel "+5511900000001"

  # --- Paused session ---

  @error-case @REQ-003
  Scenario: Message for a paused session is discarded — no assistant message written
    # requires: redis, running-server
    Given a session exists for connector "discord" and channel "e2e-paused-001"
    And the session for connector "discord" and channel "e2e-paused-001" is paused
    When I enqueue a "message:process" task with connector "discord", channel "e2e-paused-001", and content "ignored message"
    Then after 5 seconds no "user" message with content "ignored message" exists in the session
    And after 5 seconds no "assistant" message exists in the session

  # --- Session FindOrCreate idempotency ---

  @invariant @REQ-002
  Scenario: FindOrCreate is idempotent — calling it twice returns the same session
    Given no session exists for connector "whatsapp" and channel "+5511900000002"
    When I call FindOrCreate for connector "whatsapp" and channel "+5511900000002"
    And I call FindOrCreate for connector "whatsapp" and channel "+5511900000002"
    Then only 1 session exists for connector "whatsapp" and channel "+5511900000002"
    And both calls returned the same session ID

  # --- Context window ---

  @edge-case @REQ-004
  Scenario: Context window limits messages sent to Claude — DB retains all messages
    # requires: redis, running-server
    Given a session exists for connector "discord" and channel "e2e-ctx-001"
    And 20 user messages have been inserted into the session for connector "discord" and channel "e2e-ctx-001"
    When I enqueue a "message:process" task with connector "discord", channel "e2e-ctx-001", and content "context check"
    Then within 30 seconds an "assistant" message exists in the session
    And the database contains at least 21 "user" messages in the session
