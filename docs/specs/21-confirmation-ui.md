# Spec 22: Confirmation UI [FRONTEND]

## Overview

Add a new "Confirmations" tab in the web UI to display pending tool execution approvals. Show a list of pending actions with tool name, parameters preview, and approve/deny buttons. Tab polls `/api/v1/confirmations` endpoint every 2–3 seconds. On user action (approve/deny), submit to corresponding endpoint and update UI. Completed confirmations are archived or hidden automatically.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Confirmation API exists (Spec 20)
- Tab navigation system exists (`web/public/js/router.js`)
- Module structure exists (`web/public/js/modules/`)

## Deliverables

**Files to Create:**
- `web/public/js/modules/confirmations.js` — Confirmations tab module
- `web/public/css/components/confirmation.css` — styling for confirmation cards

**Files to Modify:**
- `web/public/index.html` — add "Confirmations" tab link
- `web/public/js/main.js` — import confirmations module
- `web/public/js/router.js` — add route handler for `#confirmations`
- `web/public/css/main.css` — import confirmation component styles

## Acceptance Criteria

- [ ] New tab "Confirmations" appears in navigation with icon/label
- [ ] Tab displays pending confirmations as cards with: tool name, input parameters (JSON preview, truncated to 200 chars), "Approve" button, "Deny" button
- [ ] Each card shows creation time (relative: "2 minutes ago")
- [ ] Module polls API every 2–3s; list updates without page reload
- [ ] Clicking "Approve" submits POST to `/api/v1/confirmations/{id}/approve`; button shows loading state
- [ ] Clicking "Deny" submits POST to `/api/v1/confirmations/{id}/deny`; button shows loading state
- [ ] On success, card disappears from list; toast notification shows "Action approved" or "Action denied"
- [ ] On API error (401, 500), toast shows error message
- [ ] Tab shows "No pending confirmations" when list is empty
- [ ] Approved/denied confirmations are removed from pending list immediately (no polling delay)
- [ ] If user navigates away and back, list reloads correctly

## Component Contract

**`web/public/js/modules/confirmations.js`**:
```javascript
export function init() {
	// Register route, set up polling
}

// Polls /api/v1/confirmations every 2–3 seconds
async function fetchConfirmations()

// POST /api/v1/confirmations/{id}/approve
async function approveConfirmation(id)

// POST /api/v1/confirmations/{id}/deny
async function denyConfirmation(id)

// Render list of confirmations
function renderConfirmations(data)
```

**HTML Structure**:
```html
<div id="confirmations-tab">
	<h2>Pending Actions</h2>
	<div id="confirmations-list">
		<!-- Dynamically populated with confirmation cards -->
	</div>
	<div id="empty-state" style="display:none;">
		No pending confirmations
	</div>
</div>

<!-- Confirmation card template -->
<div class="confirmation-card">
	<div class="tool-info">
		<h3>Tool: {tool_name}</h3>
		<pre class="tool-args">{tool_input_preview}</pre>
	</div>
	<div class="creation-time">{relative_time}</div>
	<div class="actions">
		<button class="btn-approve">Approve</button>
		<button class="btn-deny">Deny</button>
	</div>
</div>
```

## Out of Scope

- Confirmation history / archive tab (just remove from pending)
- Complex parameter visualization (JSON preview is sufficient)
- Auto-expiry countdown timer (silently expire server-side)
- Bulk approve/deny
