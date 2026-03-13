# Spec 34: Enhanced Confirmations UI [FRONTEND]

## Overview

Extend the Confirmations tab (Spec 22) with richer, more informative action cards. Show full tool arguments with schema context, estimated impact summary (e.g., "This will create 1 GitHub issue"), time-to-auto-deny countdown timer (confirmation expires in N seconds), and visual warnings for high-impact actions (red badge for "creates", "deletes", "sends"). Helps users make more informed approval decisions.

## Phase

**Phase 2** (Weeks 7–14)

## Prerequisites

- Confirmation API exists (Spec 20)
- Confirmation UI exists (Spec 22)
- Tool registry with schemas is available

## Deliverables

**Files to Modify:**
- `web/public/js/modules/confirmations.js` — enhance rendering with rich context
- `web/public/css/components/confirmation.css` — add styling for impact badges, countdown, expanded details

**Backend Changes** (Minor):
- Extend confirmation API response to include: `tool_schema`, `estimated_impact_summary`, `expires_in_seconds`

## Acceptance Criteria

- [ ] Confirmation cards now show: full tool schema (input parameter descriptions), estimated impact message, and expiry countdown
- [ ] Input parameters are displayed with labels from schema (e.g., "Recipient email: user@example.com" instead of raw JSON)
- [ ] High-impact tools get colored badges: red for "creates" (GitHub issues, Notion pages), yellow for "moves" (Trello cards), etc.
- [ ] Countdown timer shows remaining time (e.g., "Expires in 45 seconds") and updates every second
- [ ] When countdown reaches zero, card auto-dismisses and action is denied
- [ ] "Expand details" link shows full JSON for advanced users
- [ ] Tool schema description (from registry) is shown to provide context (e.g., "This will send an email to 3 recipients")
- [ ] Visual warnings for destructive actions (e.g., "This cannot be undone")
- [ ] Responsive on mobile: cards stack, countdown is visible, buttons are touch-friendly
- [ ] Accessibility: ARIA labels for screen readers, keyboard navigation (Tab to navigate buttons)

## Component Contract

**Enhanced Confirmation Response**:
```json
{
	"id": "conf-123",
	"session_id": "sess-456",
	"tool_name": "github_create_issue",
	"tool_call": {
		"name": "github_create_issue",
		"input": {
			"owner": "anthropics",
			"repo": "claude-code",
			"title": "Bug: tool_use crashes on invalid input",
			"body": "..."
		}
	},
	"tool_schema": {
		"name": "github_create_issue",
		"description": "Create a new GitHub issue (requires approval)",
		"input_schema": {
			"properties": {
				"owner": { "type": "string", "description": "Repository owner" },
				"repo": { "type": "string", "description": "Repository name" },
				"title": { "type": "string", "description": "Issue title" },
				"body": { "type": "string", "description": "Issue body/description" }
			}
		}
	},
	"estimated_impact": "Creates 1 GitHub issue on anthropics/claude-code repository",
	"impact_level": "medium",
	"expires_in_seconds": 3600,
	"created_at": "2026-03-12T15:30:00Z"
}
```

**`web/public/js/modules/confirmations.js`** additions:
```javascript
function renderEnhancedCard(confirmation) {
	// Render with schema labels, impact badge, countdown timer
	// Format input params from schema context
	startCountdownTimer(confirmation.id, confirmation.expires_in_seconds)
}

function startCountdownTimer(confirmationId, secondsRemaining) {
	// Update countdown every 1s
	// Auto-deny when timer hits zero
	// Remove card from UI
}

// Build impact summary from tool type + input
function estimateImpactSummary(toolName, input, schema) {
	// Examples: "Creates 1 GitHub issue", "Moves Trello card to Done list"
}
```

**HTML Structure**:
```html
<div class="confirmation-card" data-confirmation-id="{id}">
	<!-- Impact badge -->
	<div class="impact-badge" data-level="medium">
		Creates Issue
	</div>

	<!-- Tool info -->
	<div class="tool-info">
		<h3>Tool: github_create_issue</h3>
		<p class="tool-description">{schema.description}</p>
	</div>

	<!-- Tool parameters with schema labels -->
	<div class="tool-params">
		<div class="param">
			<label>Repository owner:</label>
			<code>anthropics</code>
		</div>
		<div class="param">
			<label>Repository name:</label>
			<code>claude-code</code>
		</div>
		<div class="param">
			<label>Issue title:</label>
			<code>Bug: tool_use crashes on invalid input</code>
		</div>
	</div>

	<!-- Estimated impact -->
	<div class="impact-summary">
		✓ Creates 1 GitHub issue on anthropics/claude-code repository
	</div>

	<!-- Countdown timer -->
	<div class="expiry-countdown">
		Expires in <span id="timer-{id}">59</span> seconds
	</div>

	<!-- Actions -->
	<div class="actions">
		<button class="btn-approve">Approve</button>
		<button class="btn-deny">Deny</button>
		<a class="link-expand">View full JSON</a>
	</div>

	<!-- Expandable details -->
	<details class="details-full-json">
		<summary>Full Parameters (JSON)</summary>
		<pre>{tool_call_json}</pre>
	</details>
</div>
```

## Out of Scope

- AI-powered impact estimation (Claude analyzing the action to explain consequences)
- Undo mechanism after approval
- Approval history with details
- Multi-level approval workflows
