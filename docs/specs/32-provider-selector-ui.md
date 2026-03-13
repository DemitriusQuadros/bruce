# Spec 32: Provider Selector UI [FRONTEND]

## Overview

Extend the Settings tab to add an LLM provider selector dropdown. Show currently active provider (Claude / Gemini) with a visual indicator. Clicking the dropdown allows user to change the default provider globally, or set a per-session override. Settings are persisted via API call to backend. Header also shows active provider as a small badge for visibility.

## Phase

**Phase 2** (Weeks 7–14)

## Prerequisites

- Settings tab exists (`web/public/js/modules/settings.js`)
- Gemini provider backend exists (Spec 31)
- Session model includes `provider_override` field

## Deliverables

**Files to Modify:**
- `web/public/js/modules/settings.js` — add provider selector section and logic
- `web/public/css/components/form.css` or new `settings.css` — styling for provider selector
- `web/public/js/main.js` or header component — add provider badge in header

**Files to Create:**
- (Update existing modules; no new files)

## Acceptance Criteria

- [ ] Settings tab has "LLM Provider" section with dropdown (Claude / Gemini)
- [ ] Current selection is highlighted/checked
- [ ] Changing provider submits API call to set global default or per-session override
- [ ] Header shows small badge "Claude" or "Gemini" in top-right corner, clickable to Settings
- [ ] Badge color-codes providers (blue for Claude, green for Gemini, etc.)
- [ ] Provider selection persists on page reload (backed by session in database)
- [ ] Provider change takes effect immediately on next message
- [ ] Per-session override checkbox or toggle allows choosing provider for current chat only
- [ ] UI shows note if global default is overridden per-session
- [ ] Responsive design works on mobile

## Component Contract

**HTML Structure** (added to Settings):
```html
<section id="llm-provider-settings">
	<h3>LLM Provider</h3>
	<label for="provider-select">Active Provider:</label>
	<select id="provider-select">
		<option value="claude">Claude Opus 4.6</option>
		<option value="gemini">Google Gemini 1.5 Pro</option>
	</select>
	<p id="provider-note" class="note"></p>

	<label>
		<input type="checkbox" id="session-override">
		Override for this session only
	</label>
</section>
```

**`web/public/js/modules/settings.js`** additions:
```javascript
// Fetch current provider on init
async function fetchCurrentProvider()

// POST /api/v1/sessions/{id}/provider to update session provider
async function setSessionProvider(provider, isOverride)

// Update header badge when provider changes
function updateProviderBadge(provider)
```

**API Endpoints**:
```
GET /api/v1/config
  Returns: { llm: { default_provider: "claude" } }

POST /api/v1/sessions/{id}/provider
  Body: { provider: "gemini", override: true }
  Returns: { provider: "gemini" }
```

## Out of Scope

- Per-tool provider selection
- Cost comparison display (e.g., "Claude: $X, Gemini: $Y per request")
- Provider capability matrix (which tools work with which provider)
- Auto-selection based on query complexity
