# Spec 32: Enhanced Confirmations UI [FRONTEND]

## Overview

Extend the Confirmations tab (Spec 22) with rich visualization and approval workflows. Show tool details (name, description, risk level), parameter validation, side-by-side diffs (if applicable), and suggested alternatives. Allow bulk approvals and conditional approvals ("approve for this session only"). Integrate with tool registry to display full tool metadata and help text.

## Phase

**Phase 2** (Weeks 7–14)

## Prerequisites

- Confirmation API exists (Spec 20)
- Confirmations tab exists (Spec 22)
- Tool registry provides rich metadata (Spec 12)

## Deliverables

**Files to Create:**
- `web/public/js/modules/confirmations_enhanced.js` — Enhanced confirmation logic
- `web/public/css/components/confirmation_detailed.css` — Rich detail card styling
- `web/public/js/components/tool_info.js` — Reusable tool info component

**Files to Modify:**
- `web/public/js/modules/confirmations.js` — integrate enhanced UI
- `internal/api/handlers/confirmations.go` — expose tool metadata in confirmation response

## Acceptance Criteria

- [ ] Confirmation cards display: tool icon/badge, name, risk level (low/medium/high)
- [ ] Clicking card expands to show: tool description, parameter details, expected output hints
- [ ] Parameter details show: name, type, description, current value (truncated with copy button)
- [ ] Risk indicators show colored badges: "Safe" (green), "Caution" (yellow), "High Risk" (red)
- [ ] High-risk tools (file_write, email_send) show warning banner with confirmation text
- [ ] "Approve All" button for batch approval of similar tools (with confirmation modal)
- [ ] "Approve Once" button to approve single execution; "Approve Always" to disable future prompts
- [ ] Suggested alternatives displayed (e.g., "Try calendar_read instead of email_search")
- [ ] Parameter diffs shown (old value → new value) for update-style operations
- [ ] Toast notifications on approval/denial with undo option (5s timeout)
- [ ] Confirmations older than 1 hour show expiry countdown timer
- [ ] Mobile responsive (cards stack, modals adapt)

## Component Contract

**`web/public/js/modules/confirmations_enhanced.js`**:
```javascript
export function init() {
	// Enhanced initialization with tool metadata
	setupToolCards();
	setupRiskIndicators();
	setupBulkApproval();
	setupToolSuggestions();
}

function renderToolCard(confirmation, toolMetadata) {
	// Display rich tool info card
	// Include icon, description, risk badge
	// Expandable details section
}

async function bulkApprove(toolNames, scope) {
	// scope: "once", "always", "session"
	// POST to approve multiple confirmations
}
```

**Confirmation Response (Enhanced)**:
```json
{
	"id": "conf_abc123",
	"session_id": "sess_xyz",
	"tool": {
		"name": "email_send",
		"description": "Send an email message",
		"risk_level": "high",
		"icon": "📧",
		"parameters": [
			{ "name": "to", "type": "array", "description": "Recipients", "value": ["user@example.com"] },
			{ "name": "subject", "type": "string", "value": "Meeting Notes" },
			{ "name": "body", "type": "string", "value": "..." }
		]
	},
	"suggested_alternatives": ["email_search"],
	"created_at": "2024-03-13T14:02:00Z",
	"expires_at": "2024-03-13T15:02:00Z"
}
```

**HTML Structure**:
```html
<div class="confirmation-card expanded">
	<div class="card-header">
		<div class="tool-badge">
			<span class="icon">📧</span>
			<div class="tool-info">
				<h3>Email Send</h3>
				<span class="risk-badge risk-high">High Risk</span>
			</div>
		</div>
	</div>
	<div class="card-body">
		<p class="tool-description">Send an email message</p>
		<div class="parameters">
			<h4>Parameters</h4>
			<div class="param">
				<label>to:</label>
				<code>["user@example.com"]</code>
			</div>
			<div class="param">
				<label>subject:</label>
				<code>"Meeting Notes"</code>
			</div>
		</div>
		<div class="alternatives">
			<h4>Suggested Alternatives</h4>
			<p>Consider using email_search if you want to check existing emails first.</p>
		</div>
	</div>
	<div class="card-actions">
		<button class="btn btn-approve-once">Approve Once</button>
		<button class="btn btn-approve-always">Approve Always</button>
		<button class="btn btn-deny">Deny</button>
	</div>
</div>
```

## Out of Scope

- Approval workflows with multiple users/roles
- Conditional approval logic (e.g., "approve if recipient is in whitelist")
- Tool-specific approval rules (custom approval per tool)
- Approval audit trail or detailed logging
- Export confirmations history
- API key masking in parameter display (handle with care)

