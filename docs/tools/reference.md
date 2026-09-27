# Built-in Tools Reference

This document provides a complete reference for all tools available to Bruce's autonomous agent loop.

---

## Tool Categories

- [1. Core & Productivity](#1-core--productivity)
  - [`artifact_save`](#artifact_save)
  - [`web_search`](#web_search)
  - [`http_request`](#http_request)
- [2. Proactive & Scheduling](#2-proactive--scheduling)
  - [`proactive_create`](#proactive_create)
  - [`proactive_list`](#proactive_list)
  - [`proactive_toggle`](#proactive_toggle)
  - [`proactive_delete`](#proactive_delete)
- [3. System & Filesystem](#3-system--filesystem)
  - [`bash_exec`](#bash_exec)
  - [`file_read`, `file_write`, `file_list`](#filesystem-tools)
  - [`git_local`](#git_local)
- [4. Integrations & Third-Party APIs](#4-integrations--third-party-apis)
  - [Google Workspace (`gmail`, `calendar`, `docs`)](#google-workspace-tools)
  - [`github`](#github)
  - [`notion`](#notion)
  - [`trello`](#trello)
  - [`n8n`](#n8n)

---

## 1. Core & Productivity

### `artifact_save`
Generates and persists a standalone HTML document saved on disk and served over HTTP.

- **Config**: `tools.artifacts.enabled: true`
- **Parameters**:
  - `title` (*string, required*): Human-readable title for the artifact.
  - `filename` (*string, required*): Target filename ending in `.html` (e.g. `dashboard.html`).
  - `content` (*string, required*): Complete standalone HTML document including `<!DOCTYPE html>`, `<html>`, `<head>`, `<style>`, and `<body>`.
- **Output**: Returns the public access path `/artifacts/<id>/<filename>`.

---

### `web_search`
Queries Google Custom Search API to retrieve live information and citations from the web.

- **Config**:
  ```yaml
  tools:
    web_search:
      enabled: true
      google_api_key: "AIzaSy..."
      google_cx: "0123456789:abcdef"
  ```
- **Parameters**:
  - `query` (*string, required*): Search query terms.
  - `num_results` (*integer, optional*): Number of results to return (1-10, default: 5).
- **Output**: Formatted list of titles, snippets, and source URLs.

---

### `http_request`
Makes arbitrary HTTP requests (GET, POST, PUT, DELETE) to external REST APIs or webhooks.

- **Config**: Enabled by default (`tools.http_client.enabled: true`).
- **Parameters**:
  - `url` (*string, required*): Full HTTP/HTTPS target URL.
  - `method` (*string, required*): HTTP method (`GET`, `POST`, `PUT`, `DELETE`).
  - `headers` (*object, optional*): Map of header key-value pairs.
  - `body` (*string, optional*): Request body content.
- **Output**: HTTP status code and response body text.

---

## 2. Proactive & Scheduling

### `proactive_create`
Registers a background schedule or ambient monitoring watch.

- **Config**: Built-in (requires scheduler poller).
- **Parameters**:
  - `title` (*string, required*): Descriptive name of the schedule or watch.
  - `type` (*string, required*): `'cron'` for time-based tasks or `'watch'` for periodic condition monitoring.
  - `schedule` (*string, required*):
    - For `cron`: Standard 5-token cron (`0 9 * * 1-5`) or relative offset (`+2m`, `in 5 minutes`, `daqui a 2 minutos`).
    - For `watch`: Polling interval in minutes (integer string, e.g. `'30'`, minimum 5).
  - `execution_mode` (*string, optional*):
    - `'message'`: Direct notification/reminder delivery (bypasses LLM, 0 tokens, 0 latency).
    - `'agent'`: Runs through the AI agent loop with tool execution.
  - `prompt_condition` (*string, required*):
    - For `execution_mode: 'message'`: The exact text to deliver.
    - For `execution_mode: 'agent'`: Instructions for the AI agent to compile.
    - For `watch`: The condition to monitor (e.g. `'emails from contractors'`).
  - `target_connector` (*string, optional*): `'discord'`, `'whatsapp'`, `'telegram'`, or `'web'`.
  - `target_channel_id` (*string, optional*): Destination channel ID or phone number.
  - `target_tools` (*array of strings, optional*): Limit tools for condition watch.
  - `timezone` (*string, optional*): IANA timezone (defaults to host/app timezone).

---

### `proactive_list`
Lists all proactive tasks, schedules, and watches associated with the current session or all sessions.

---

### `proactive_toggle`
Pauses or resumes a background proactive task.

- **Parameters**:
  - `task_id` (*string, required*): The task UUID.
  - `is_active` (*boolean, required*): `true` to resume, `false` to pause.

---

### `proactive_delete`
Permanently deletes a scheduled proactive task.

- **Parameters**:
  - `task_id` (*string, required*): The task UUID.

---

## 3. System & Filesystem

### `bash_exec`
Executes terminal commands inside a sandboxed working directory with a timeout safeguard.

- **Config**:
  ```yaml
  tools:
    bash:
      enabled: true
      working_dir: "/path/to/sandbox"
      timeout_seconds: 30
  ```
- **Parameters**:
  - `command` (*string, required*): Shell command string.
- **Output**: Combined stdout and stderr.

---

### Filesystem Tools
- `file_read(path)`: Reads content of a local file.
- `file_write(path, content)`: Writes content to a local file.
- `file_list(directory)`: Lists files and subdirectories.

---

### `git_local`
Performs safe local git operations on a repository on disk:
- `git_status`: Inspect working directory state.
- `git_log`: View recent commit history.
- `git_diff`: View unstaged or staged changes.
- `git_commit`: Create a commit with a message.

---

## 4. Integrations & Third-Party APIs

### Google Workspace Tools
Requires Google OAuth Client credentials (`google.oauth_client_id` & `google.oauth_client_secret`).
- **`gmail_search`**: Search inbox using standard Gmail search queries (e.g. `is:unread from:boss`).
- **`gmail_read`**: Read full email body and thread metadata.
- **`gmail_send`**: Compose and send outbound emails.
- **`calendar_list_events`**: Retrieve upcoming calendar events for a date range.
- **`calendar_create_event`**: Schedule a meeting or calendar event with title, start, and end time.
- **`docs_read` / `docs_create`**: Read or create Google Docs documents.

---

### `github`
Interact with GitHub issues, pull requests, and repositories.
- **Config**: `tools.github.api_token: "ghp_..."`
- `github_list_issues`: Fetch issues with state, labels, and assignees.
- `github_create_issue`: Open a new issue in a repository.
- `github_list_prs`: Inspect open or merged pull requests.
- `github_get_repo`: Retrieve repository statistics and metadata.

---

### `notion`
Query and update Notion databases and workspaces.
- **Config**: `tools.notion.api_key: "secret_..."`
- `notion_search`: Search pages and databases by title.
- `notion_get_page`: Retrieve page content blocks.
- `notion_create_page`: Create a new page or database entry.

---

### `trello`
Manage Kanban boards, lists, and cards.
- **Config**: `tools.trello.api_key` & `tools.trello.user_token`.
- `trello_list_boards`: List authorized boards.
- `trello_list_cards`: Retrieve cards from a board list.
- `trello_create_card`: Create a new card with title, description, and due date.

---

### `n8n`
Trigger n8n automation workflows and webhooks.
- **Config**: `tools.n8n.base_url` & `tools.n8n.api_key`.
- `n8n_trigger_webhook`: Send payload to an n8n webhook node.
- `n8n_execute_workflow`: Trigger workflow execution via n8n REST API.
