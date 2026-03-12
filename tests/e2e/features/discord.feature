Feature: Discord Connector
  As a Bruce user
  I want to send and receive messages via Discord
  So that I can interact with my AI assistant through Discord DMs

  Background:
    Given the database is clean

  # --- Happy Path ---

  @smoke @REQ-001
  Scenario: User sends a DM to Bruce and receives a response
    # Integration: Discord user sends DM → connector enqueues → worker processes → sends response
    # Requires: Discord bot token configured, redis running, server running
    Given a session exists for connector "discord" and channel "dm-smoke-001"
    When I enqueue a "message:process" task with connector "discord", channel "dm-smoke-001", and content "hello"
    Then within 30 seconds a "user" message with content "hello" exists in the session
    And within 30 seconds an "assistant" message exists in the session

  # --- Message Chunking Tests ---

  @invariant @REQ-002
  Scenario: Discord connector chunks messages exceeding 1900 characters
    # Discord limit is 2000 chars, connector uses 1900 to be safe
    Given the discord dispatcher is initialized
    When I call the dispatcher with a 2500 character message
    Then the message is split into multiple Discord messages
    And each chunk is at most 1900 characters
    And all chunks concatenated equal the original message

  @edge-case @REQ-002
  Scenario: Message exactly at 1900 character limit is not chunked
    Given the discord dispatcher is initialized
    When I call the dispatcher with a 1900 character message
    Then the message is sent as a single Discord message

  @edge-case @REQ-002
  Scenario: Message exceeding 1900 chars breaks on word boundaries when possible
    Given the discord dispatcher is initialized
    When I call the dispatcher with a 2100 character message with word boundaries
    Then the message is split at a word boundary
    And no chunk ends mid-word

  @edge-case @REQ-002
  Scenario: Long words without spaces are hard-cut at the limit
    Given the discord dispatcher is initialized
    When I call the dispatcher with a 2500 character word without spaces
    Then the message is split into chunks of exactly 1900 characters
    And the last chunk is shorter than 1900 characters

  # --- Connector Initialization ---

  @smoke @REQ-003
  Scenario: Discord connector initializes with valid token
    Given a discord bot token is configured
    When the discord connector is initialized
    Then the connector is ready to process messages

  @error-case @REQ-003
  Scenario: Discord connector fails to initialize without a token
    Given no discord bot token is configured
    When the discord connector is initialized
    Then an error is returned

  # --- Channel Filtering ---

  @invariant @REQ-004
  Scenario: Discord connector only processes DM channels, not guild messages
    # Phase 1 scope: DMs only
    Given a session exists for connector "discord" and channel "guild-channel-001"
    When I enqueue a message from a guild channel (not DM)
    Then the message is ignored and no task is enqueued

  # --- Message Validation ---

  @error-case @REQ-005
  Scenario: Empty messages are ignored by connector before enqueueing
    # The Discord connector filters empty messages at event handler level
    # so they never reach the task queue or create messages in DB
    Given a session exists for connector "discord" and channel "dm-empty-001"
    When the discord connector receives an empty message from a Discord DM
    Then no task is enqueued
    And the database contains zero messages for the session

  @edge-case @REQ-005
  Scenario: Messages with only whitespace are ignored by connector before enqueueing
    # Whitespace-only messages are also filtered by the connector
    Given a session exists for connector "discord" and channel "dm-whitespace-001"
    When the discord connector receives a whitespace-only message from a Discord DM
    Then no task is enqueued
    And the database contains zero messages for the session

  # --- Bot Self-Message Filtering ---

  @invariant @REQ-006
  Scenario: Discord connector ignores messages from the bot itself
    Given the discord connector knows its own bot user ID
    When the discord connector receives a message from the bot
    Then the message is ignored and no task is enqueued

  # --- Intent Scoping ---

  @invariant @REQ-007
  Scenario: Discord connector requests only necessary gateway intents
    # Principle of least privilege
    Given a discord bot token is configured
    When the discord connector is initialized
    Then the connector only requests IntentsDirectMessages and IntentsDirectMessageReactions
    And no other intents are requested
