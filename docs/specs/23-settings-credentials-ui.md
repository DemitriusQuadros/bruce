# Spec 22: Settings & Credentials UI [FRONTEND]

## Overview

Extend the Settings tab in the web UI to manage OAuth connections, LLM provider selection, and API key configuration. Show a list of connected OAuth services (Google, GitHub, Notion, Trello) with disconnect buttons. Allow users to select the active LLM provider (Claude, Gemini) and configure provider-specific settings. Provide forms to manage API keys securely (masked input, store server-side).

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Settings tab exists (`web/public/js/modules/settings.js`)
- OAuth flow endpoints exist (Spec 13)
- Backend settings API exists (`GET /api/v1/config`, `POST /api/v1/config/update`)
- Multi-provider LLM abstraction in place (Spec 7)

## Deliverables

**Files to Create:**
- `web/public/js/modules/settings.js` (extend) — add credentials and provider sections
- `web/public/css/components/settings.css` (create/extend) — styling for credential cards

**Files to Modify:**
- `web/public/index.html` — ensure Settings tab is present
- `web/public/js/router.js` — handle `#settings` route (if not already)
- `internal/api/handlers/config.go` — add endpoints for OAuth status and provider selection
- `internal/config/config.go` — add provider selection field to Config struct

## Acceptance Criteria

- [ ] Settings tab displays sections: "OAuth Connections", "LLM Provider", "API Keys"
- [ ] OAuth Connections section lists: Google, GitHub, Notion, Trello with status (connected/disconnected)
- [ ] Each OAuth service has "Connect" or "Disconnect" button
- [ ] Clicking "Connect" redirects to `/auth/{provider}/start` endpoint
- [ ] Clicking "Disconnect" shows confirmation modal, then calls disconnect endpoint
- [ ] On successful disconnect, status updates immediately (no page reload)
- [ ] LLM Provider section shows radio buttons: "Claude (claude-opus-4-6)", "Gemini (gemini-pro)" with descriptions
- [ ] Selecting a new provider saves choice and shows confirmation toast
- [ ] API Keys section (if applicable) allows masking/unmasking; submits securely to backend
- [ ] All credential updates trigger toast notifications (success or error)
- [ ] Page persists settings on browser refresh (read from `/api/v1/config`)
- [ ] Mobile responsive (settings section stacks vertically)

## Component Contract

**`web/public/js/modules/settings.js`**:
```javascript
export function init() {
	// Register route, load settings
	setupOAuthConnections();
	setupProviderSelector();
	setupAPIKeySection();
}

async function setupOAuthConnections() {
	// GET /api/v1/auth/status
	// Display connected services
	// Add connect/disconnect handlers
}

async function setupProviderSelector() {
	// GET /api/v1/config
	// Show current provider
	// Handle radio button change
	// POST /api/v1/config/update with { llm_provider: "..." }
}

async function disconnectOAuth(provider) {
	// POST /api/v1/auth/{provider}/disconnect
}
```

**HTML Structure**:
```html
<div id="settings-tab">
	<h2>Settings & Credentials</h2>

	<section id="oauth-connections">
		<h3>OAuth Connections</h3>
		<div class="oauth-list">
			<div class="oauth-card">
				<h4>Google Workspace</h4>
				<p>For Gmail, Calendar, Docs, Drive</p>
				<button class="btn-disconnect">Disconnect</button>
			</div>
			<!-- Similar cards for GitHub, Notion, Trello -->
		</div>
	</section>

	<section id="llm-provider">
		<h3>LLM Provider</h3>
		<div class="radio-group">
			<label>
				<input type="radio" name="provider" value="claude" />
				<span>Claude (Anthropic)</span>
			</label>
			<label>
				<input type="radio" name="provider" value="gemini" />
				<span>Gemini (Google)</span>
			</label>
		</div>
	</section>

	<section id="api-keys">
		<h3>API Keys</h3>
		<form id="api-key-form">
			<label>Claude API Key</label>
			<input type="password" name="claude_api_key" placeholder="sk-..." />
			<button type="submit" class="btn-save">Save</button>
		</form>
	</section>
</div>
```

**Backend Endpoints**:
```
GET /api/v1/auth/status
  Returns: { google: { connected: bool }, github: { connected: bool }, ... }

POST /api/v1/auth/{provider}/disconnect
  Returns: { status: "disconnected" }

GET /api/v1/config
  Returns: { llm_provider: "claude", ... }

POST /api/v1/config/update
  Body: { llm_provider: "gemini", ... }
  Returns: { status: "updated" }
```

## Out of Scope

- OAuth token refresh UI details
- Two-factor authentication setup
- Rate limit configuration
- API key rotation policies
- Service-level settings per OAuth provider (only global config)

