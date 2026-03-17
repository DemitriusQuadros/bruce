# Spec 33: Tools Log UI [FRONTEND]

## Overview

Add a "Tools" subtab inside the Logs view that displays a detailed log of all tool executions. Show tool name, parameters (preview), result (success/error), latency, and timestamp. Sortable by date, tool, or status. Filter by tool name or status (success/failed/pending). Updates dynamically as tools are executed during active chat. Helps users debug and understand what actions Claude took.

## Phase

**Phase 2** (Weeks 7–14)

## Prerequisites

- Logs tab exists (`web/public/js/modules/logs.js`)
- Tool execution logging exists in backend (Spec 11, 12)
- API endpoint to fetch tool execution history

## Deliverables

**Files to Modify:**
- `web/public/js/modules/logs.js` — add tools subtab and logic
- `web/public/css/components/logs.css` — styling for tools log table

**Files to Create:**
- (Update existing logs module; no new files strictly required)

**Backend Changes** (Minimal):
- Add API endpoint `GET /api/v1/sessions/{id}/tool-executions` to fetch log entries

## Acceptance Criteria

- [ ] Logs tab has new "Tools" subtab alongside existing subtabs (Messages, etc.)
- [ ] Tools subtab displays table: Tool Name | Parameters | Result | Latency | Timestamp
- [ ] Parameters column shows JSON preview (truncated to 100 chars, expandable)
- [ ] Result column shows "Success", "Error: [message]", "Pending" with color coding
- [ ] Latency column shows time in ms (e.g., "234ms", "1.2s")
- [ ] Timestamp shows relative time (e.g., "2 minutes ago") with tooltip for exact time
- [ ] Table is sortable by any column (click header)
- [ ] Filter dropdown: by tool name, by status (all/success/failed)
- [ ] "Expand" link on each row shows full parameters and result JSON
- [ ] Subtab auto-updates as new tools execute (polls API every 2–3s during active session)
- [ ] Responsive: table is readable on mobile (horizontal scroll or collapsing columns)

## Component Contract

**API Endpoint**:
```
GET /api/v1/sessions/{id}/tool-executions?limit=100&offset=0
  Returns: {
    executions: [
      {
        id: "uuid",
        tool_name: "gmail_read",
        input: { action: "list", max_results: 5 },
        output: { messages: [...] },
        status: "success",
        latency_ms: 1234,
        created_at: "2026-03-12T15:30:00Z"
      },
      ...
    ],
    total: 42
  }
```

**`web/public/js/modules/logs.js`** additions:
```javascript
function initToolsSubtab() {
	// Register tools subtab
	// Fetch and render tool executions
	// Set up polling for updates
}

async function fetchToolExecutions(sessionId, limit, offset, filter) {
	// Fetch from API with optional filter
}

function renderToolsTable(executions) {
	// Render table with sortable headers
	// Format timestamps, latency, status
}
```

**HTML Structure**:
```html
<div id="tools-subtab">
	<div class="tools-toolbar">
		<label>Filter by tool:
			<select id="tool-filter">
				<option value="">All Tools</option>
				<option value="gmail_read">Gmail</option>
				<option value="calendar_read">Calendar</option>
				<!-- ... -->
			</select>
		</label>
		<label>Status:
			<select id="status-filter">
				<option value="">All</option>
				<option value="success">Success</option>
				<option value="failed">Failed</option>
			</select>
		</label>
	</div>
	<table id="tools-log">
		<thead>
			<tr>
				<th>Tool Name</th>
				<th>Parameters</th>
				<th>Result</th>
				<th>Latency</th>
				<th>Timestamp</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			<!-- Dynamically populated -->
		</tbody>
	</table>
</div>
```

## Out of Scope

- Tool execution replay/re-run
- Execution timeline visualization
- Integration with analytics/metrics dashboard
- Tool execution comparison across sessions
