@proactive
Feature: Proactive Intelligence, Ambient Watches & Scheduled Reports
  As a user interacting with Bruce via chat or web dashboard
  I want to create, inspect, pause, resume, trigger, and delete scheduled reports and ambient watches
  So that Bruce can autonomously monitor tools and send proactive intelligence to my messaging channels

  Background:
    Given the database is clean
    And the API is running

  # --- Happy Path: REST API CRUD ---

  @smoke @REQ-PROACTIVE-001
  Scenario: Create a scheduled report (cron) via REST API
    When I send a POST request to "/api/v1/proactive-tasks" with body:
      """
      {
        "title": "Daily Standup Briefing",
        "task_type": "cron",
        "schedule_expr": "0 9 * * 1-5",
        "prompt_condition": "Summarize today's calendar meetings and open PRs",
        "target_connector": "discord",
        "target_channel_id": "chan-standup-001"
      }
      """
    Then the response status is 201
    And the response body field "title" is "Daily Standup Briefing"
    And the response body field "task_type" is "cron"
    And the response body field "is_active" is "true"
    And the response body contains "id"
    And the database contains a proactive task titled "Daily Standup Briefing" with type "cron"

  @smoke @REQ-PROACTIVE-002
  Scenario: Create an ambient condition watch via REST API
    When I send a POST request to "/api/v1/proactive-tasks" with body:
      """
      {
        "title": "VIP Email Watcher",
        "task_type": "watch",
        "schedule_expr": "15",
        "prompt_condition": "emails from VIP client about contract",
        "target_connector": "whatsapp",
        "target_channel_id": "+5511999999999",
        "target_tools": ["gmail_search"]
      }
      """
    Then the response status is 201
    And the response body field "title" is "VIP Email Watcher"
    And the response body field "task_type" is "watch"
    And the response body field "schedule_expr" is "15"
    And the database contains a proactive task titled "VIP Email Watcher" with type "watch"

  @smoke @REQ-PROACTIVE-003
  Scenario: List proactive tasks returns active and paused tasks
    Given a proactive cron task exists with title "Morning Briefing" and schedule "0 9 * * 1-5"
    And an ambient watch task exists with title "VIP Email Watch" and interval "15"
    When I send a GET request to "/api/v1/proactive-tasks"
    Then the response status is 200
    And the response body is a JSON array of length 2
    And the response body contains "Morning Briefing"
    And the response body contains "VIP Email Watch"

  @smoke @REQ-PROACTIVE-004
  Scenario: Pause and resume a proactive task via PATCH
    Given a proactive cron task exists with title "Daily Sync" and schedule "0 10 * * 1-5"
    When I send a PATCH request to the proactive task with body:
      """
      {"is_active": false}
      """
    Then the response status is 200
    And the proactive task has status "paused"
    When I send a PATCH request to the proactive task with body:
      """
      {"is_active": true}
      """
    Then the response status is 200
    And the proactive task has status "active"

  @smoke @REQ-PROACTIVE-005
  Scenario: Delete a proactive task via REST API
    Given a proactive cron task exists with title "Temporary Task" and schedule "0 12 * * *"
    When I send a DELETE request to the proactive task
    Then the response status is 200
    And the response body contains "task deleted"
    When I send a GET request to the proactive task
    Then the response status is 404
    And the database contains 0 proactive tasks

  @smoke @REQ-PROACTIVE-006
  Scenario: Manually trigger an immediate run via run endpoint
    Given a proactive cron task exists with title "On-Demand Report" and schedule "0 9 * * 1-5"
    When I trigger an immediate run for the proactive task
    Then the response status is 202
    And the response body contains "execution enqueued"

  # --- Conversational AI Tool Execution ---

  @smoke @REQ-PROACTIVE-007
  Scenario: Conversational creation of scheduled cron via agent tool proactive_create
    Given a session exists for connector "discord" and channel "chan-discord-99"
    When I execute the conversational tool "proactive_create" with parameters:
      """
      {
        "title": "Daily Standup Summary",
        "type": "cron",
        "schedule": "0 9 * * 1-5",
        "prompt_condition": "Summarize GitHub PRs and Calendar events"
      }
      """
    Then the tool execution succeeds with confirmation
    And the database contains a proactive task titled "Daily Standup Summary" with type "cron"
    And the proactive task target connector is "discord" and channel is "chan-discord-99"

  @smoke @REQ-PROACTIVE-008
  Scenario: Conversational creation of ambient watch with WhatsApp context auto-detection
    Given a session exists for connector "whatsapp" and channel "+5511988887777"
    When I execute the conversational tool "proactive_create" with parameters:
      """
      {
        "title": "Urgent Email Watcher",
        "type": "watch",
        "schedule": "10",
        "prompt_condition": "emails from VIP client",
        "target_tools": ["gmail_search"]
      }
      """
    Then the tool execution succeeds with confirmation
    And the database contains a proactive task titled "Urgent Email Watcher" with type "watch"
    And the proactive task target connector is "whatsapp" and channel is "+5511988887777"

  @smoke @REQ-PROACTIVE-009
  Scenario: Cross-channel conversational task creation
    Given a session exists for connector "discord" and channel "chan-dev-general"
    When I execute the conversational tool "proactive_create" with parameters:
      """
      {
        "title": "Cross-channel Alert",
        "type": "cron",
        "schedule": "0 8 * * *",
        "prompt_condition": "Daily infrastructure health check",
        "target_connector": "whatsapp",
        "target_channel_id": "+5511999998888"
      }
      """
    Then the tool execution succeeds with confirmation
    And the proactive task target connector is "whatsapp" and channel is "+5511999998888"

  @smoke @REQ-PROACTIVE-010
  Scenario: Conversational tools lifecycle for list, toggle, and delete
    Given a proactive cron task exists with title "Lifecycle Briefing" and schedule "0 9 * * 1-5"
    When I execute the conversational tool "proactive_list"
    Then the tool execution succeeds with confirmation
    And the tool output contains "Lifecycle Briefing"
    When I execute the conversational tool "proactive_toggle" with task "Lifecycle Briefing" and action "pause"
    Then the tool execution succeeds with confirmation
    And the proactive task has status "paused"
    When I execute the conversational tool "proactive_delete" with task "Lifecycle Briefing"
    Then the tool execution succeeds with confirmation
    And the database contains 0 proactive tasks

  # --- Error Cases ---

  @error @REQ-PROACTIVE-011
  Scenario: Watch interval under 5 minutes is rejected
    When I send a POST request to "/api/v1/proactive-tasks" with body:
      """
      {
        "title": "Too Fast Watch",
        "task_type": "watch",
        "schedule_expr": "2",
        "prompt_condition": "check continuously"
      }
      """
    Then the response status is 400
    And the response body contains "cannot be less than 5 minutes"

  @error @REQ-PROACTIVE-012
  Scenario: Invalid cron expression is rejected
    When I send a POST request to "/api/v1/proactive-tasks" with body:
      """
      {
        "title": "Broken Cron",
        "task_type": "cron",
        "schedule_expr": "not-a-cron-expression",
        "prompt_condition": "run daily"
      }
      """
    Then the response status is 400
    And the response body contains "invalid cron expression"

  @error @REQ-PROACTIVE-013
  Scenario: Missing required title returns 400 Bad Request
    When I send a POST request to "/api/v1/proactive-tasks" with body:
      """
      {
        "task_type": "cron",
        "schedule_expr": "0 9 * * *",
        "prompt_condition": "run without title"
      }
      """
    Then the response status is 400
    And the response body contains "title is required"

  @error @REQ-PROACTIVE-014
  Scenario: Get non-existent proactive task returns 404 Not Found
    When I send a GET request to "/api/v1/proactive-tasks/non-existent-task-id"
    Then the response status is 404
    And the response body contains "task not found"

  @error @REQ-PROACTIVE-015
  Scenario: Trigger run on non-existent task returns 404 Not Found
    When I send a POST request to "/api/v1/proactive-tasks/non-existent-task-id/run"
    Then the response status is 404

  # --- Edge Cases ---

  @edge-case @REQ-PROACTIVE-016
  Scenario: Filter proactive tasks by is_active query param
    Given a proactive cron task exists with title "Active Task" and schedule "0 9 * * *"
    And a proactive cron task exists with title "Paused Task" and schedule "0 10 * * *"
    And I pause the proactive task "Paused Task"
    When I send a GET request to "/api/v1/proactive-tasks?is_active=true"
    Then the response status is 200
    And the response body is a JSON array of length 1
    And the response body contains "Active Task"

  @edge-case @REQ-PROACTIVE-017
  Scenario: Updating schedule expression via PATCH recalculates next_run_at
    Given an ambient watch task exists with title "Flexible Watch" and interval "15"
    When I send a PATCH request to the proactive task with body:
      """
      {"schedule_expr": "60"}
      """
    Then the response status is 200
    And the response body field "schedule_expr" is "60"

  # --- Invariants ---

  @invariant @REQ-PROACTIVE-018
  Scenario: Deleting a session cascades and deletes associated proactive tasks
    Given a session exists for connector "discord" and channel "cascade-chan-1"
    And a proactive task exists for that session
    When I delete that session
    Then the database has no proactive tasks for the deleted session
