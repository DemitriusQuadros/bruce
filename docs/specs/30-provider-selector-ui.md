# Spec 30: Provider Selector UI [FRONTEND]

## Overview

Add a provider selector dropdown/toggle in the web UI to switch between active LLM providers (Claude, Gemini, etc.). Placed in the header or settings tab. When a provider is selected, the UI saves the choice (localStorage + backend config), and all subsequent API calls use that provider. Displays provider name, model version, and brief description. Part of the multi-provider LLM abstraction (Spec 7).

## Phase

**Phase 1** (Weeks 1–6)

## Prerequisites

- Multi-provider LLM abstraction exists (Spec 7)
- Settings API exists (Spec 22)
- Header/navigation component exists

## Deliverables

**Files to Create/Modify:**
- `web/public/js/modules/provider_selector.js` — Provider selector logic and UI
- `web/public/css/components/provider_selector.css` — Styling for dropdown/toggle
- `web/public/js/main.js` — import provider selector module

**Files to Modify:**
- `web/public/index.html` — add provider selector to header
- `web/public/js/store.js` — add provider state management
- `internal/api/handlers/config.go` — expose current provider + available providers

## Acceptance Criteria

- [ ] Header displays current provider (e.g., "Claude Opus 4.6")
- [ ] Clicking provider name opens dropdown listing available providers
- [ ] Each provider option shows: name, model version, status (available/unavailable)
- [ ] Selecting a provider calls `POST /api/v1/config/update` with new provider
- [ ] On success, UI updates header and shows confirmation toast
- [ ] Current provider selection is saved to localStorage for persistence
- [ ] On page load, UI restores previous provider selection
- [ ] Provider selector works across all tabs (selection is global)
- [ ] Disabled/unavailable providers show grayed out with tooltip explanation
- [ ] Mobile responsive (dropdown adapts to screen size)

## Component Contract

**`web/public/js/modules/provider_selector.js`**:
```javascript
export function init() {
	// Load available providers from /api/v1/config
	// Restore provider from localStorage
	// Setup click handlers for dropdown
	// Subscribe to store changes (if provider changes elsewhere)
}

async function loadAvailableProviders() {
	// GET /api/v1/config/providers
	// Returns: [{ id, name, model, status, description }]
}

async function selectProvider(providerId) {
	// POST /api/v1/config/update { llm_provider: providerId }
	// Update localStorage
	// Update store
	// Show toast
}
```

**HTML Structure**:
```html
<div id="provider-selector" class="header-control">
	<button class="provider-button">
		<span class="provider-name">Claude Opus 4.6</span>
		<span class="dropdown-icon">▼</span>
	</button>
	<div class="provider-dropdown" style="display: none;">
		<div class="provider-option" data-provider="claude">
			<h4>Claude Opus 4.6</h4>
			<p>Anthropic's latest large model</p>
		</div>
		<div class="provider-option" data-provider="gemini">
			<h4>Gemini Pro</h4>
			<p>Google's large language model</p>
		</div>
		<!-- More providers as they're added -->
	</div>
</div>
```

**Backend Endpoints**:
```
GET /api/v1/config/providers
  Returns: [
	  { id: "claude", name: "Claude", model: "claude-opus-4-6", status: "available", description: "..." },
	  { id: "gemini", name: "Gemini", model: "gemini-pro", status: "available", description: "..." }
  ]

POST /api/v1/config/update
  Body: { llm_provider: "gemini" }
  Returns: { status: "updated", current_provider: "gemini" }
```

## Out of Scope

- Provider-specific UI customization (same UI for all)
- Provider capability comparison matrix
- Automatic provider failover
- Cost tracking per provider
- Usage statistics per provider

