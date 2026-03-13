# Spec 23: Settings Credentials UI [FRONTEND]

## Overview

Extend the Settings tab in the web UI to add credential management for external services. Add "OAuth Connections" section with buttons to connect Google, GitHub (Phase 2), and other providers. Each service shows connection status (connected/not connected), last auth date, and "Disconnect" button. Clicking "Connect [Service]" opens OAuth flow (`GET /auth/{service}/start`). Implemented as modular section within existing Settings tab.

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Settings tab exists (`web/public/js/modules/settings.js`)
- OAuth flow API exists (Spec 13, Phase 1: Google only)
- Tab/module navigation system exists

## Deliverables

**Files to Modify:**
- `web/public/js/modules/settings.js` — add OAuth credentials section
- `web/public/css/components/form.css` (or new `settings.css`) — styling for credential cards

**Files to Create:**
- (Update existing settings module; no new files strictly required)

## Acceptance Criteria

- [ ] Settings tab displays "OAuth Connections" section
- [ ] For Google: shows "Not Connected" or "Connected (since MM/DD/YYYY)" with service icon/badge
- [ ] "Connect Google" button opens new window to `/auth/google/start` (OAuth flow)
- [ ] On successful OAuth redirect (after callback), parent window detects completion and updates status to "Connected"
- [ ] "Disconnect" button clears stored credentials (calls backend to delete from `oauth_tokens` table)
- [ ] Tool enable/disable toggles exist for each service (e.g., "Enable Gmail tool", "Enable Calendar tool")
- [ ] UI shows warning if service is enabled but not connected
- [ ] Connection status persists on page reload (backend state is source of truth)
- [ ] Error during OAuth redirects user back to Settings with error message
- [ ] Status badge uses color coding: green (connected), gray (not connected), yellow (connection warning)
- [ ] Responsive design: works on desktop and mobile viewport

## Component Contract

**HTML Structure** (added to Settings):
```html
<section id="oauth-connections">
	<h3>OAuth Connections</h3>
	<div class="connection-card" data-service="google">
		<div class="service-info">
			<img src="images/google-icon.svg" alt="Google">
			<div>
				<h4>Google (Gmail, Calendar, Docs)</h4>
				<p id="status-google">Not Connected</p>
			</div>
		</div>
		<div class="actions">
			<button id="btn-connect-google" class="btn-primary">Connect Google</button>
			<button id="btn-disconnect-google" class="btn-secondary" style="display:none;">Disconnect</button>
		</div>
	</div>
</section>

<section id="tool-settings">
	<h3>Enable/Disable Tools</h3>
	<label>
		<input type="checkbox" id="enable-gmail" checked>
		Gmail Tool (requires Google connection)
	</label>
	<label>
		<input type="checkbox" id="enable-calendar" checked>
		Calendar Tool (requires Google connection)
	</label>
	<!-- More tools -->
</section>
```

**`web/public/js/modules/settings.js`** additions:
```javascript
// Fetch current OAuth status on module init
async function fetchOAuthStatus()

// Open OAuth flow in new window
function connectOAuthService(service) {
	// window.open(`/auth/${service}/start`, '_blank', 'width=500,height=600')
}

// Detect OAuth callback completion (via message event or polling)
window.addEventListener('storage', (e) => {
	// Listen for OAuth completion signal from callback
	// Update connection status in UI
})

// POST to /api/v1/auth/disconnect to clear credentials
async function disconnectOAuth(service)

// Tool enable/disable via API (future enhancement)
async function setToolEnabled(toolName, enabled)
```

## Out of Scope

- GitHub OAuth (Phase 2, Spec 26)
- API key input fields (only OAuth for Phase 1)
- Token refresh UI (automatic server-side)
- Permission scopes selector (fixed scopes for each service)
- Credential history / rotation
