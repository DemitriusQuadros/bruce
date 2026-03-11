---
description: Design and implement frontend user interfaces (Vanilla JS, React, Angular) following UX design principles
---

# Frontend Specialist Workflow

Use this workflow to design pages, components, or full frontend applications. As a frontend UX specialist, you prioritize UX design and strict technical implementations fitting the Go binary embedding philosophy (or standalone SPA requests).

## Step 0 — UX First (Mandatory)
Before writing any code, execute this UX thinking:
1. **Persona Definition**: Name, Role, Goal, Pain, Tech level, Key scenario.
2. **User Flow**: `[Entry] -> [Action] -> [Interaction] -> [Success] -> [Next]`
3. **UX Principles**: Apply Clarity, Progressive disclosure, Feedback loops, Error recovery, and WCAG AA Accessibility.

## Stack Decision Framework
Determine the right stack based on context:
- **Mode 1: Vanilla HTML/JS/CSS (DEFAULT for go-base-project)**
  - For UIs served directly from the Go binary via embedded files. No build step.
- **Mode 2: React (Vite SPA)**
  - Provided when explicitly requested or standalone SPA complexity implies it.
- **Mode 3: Angular**
  - For enterprise-scale SPA requests.

## Workflow Instructions

### If Mode 1 (Vanilla JS Go-Embedded):
- Define **Design Tokens** as CSS custom properties (`:root`). Use BEM for classes.
- Create simple `api.js` utilizing vanilla `fetch` wrappers.
- Inject visible Loading States (disabling buttons) and visual error humanization.
- Add components like vanilla JS Modals, Toasts.

### If Mode 2 (React Vite):
- Structure code with `/hooks`, `/components/ui`, `/components/domain`, `/api`.
- Use `TanStack Query` for data loading and state handling.
- Build Vitest component tests and Playwright E2E tests.

### UX Implementation Patterns required in code:
- **Loading states:** Every async call must visibly lock the UI trigger.
- **Error mapping:** Never expose raw `{"error":"..."}` strings. Map API errors to human language.
- **Accessibility:** Ensure all inputs have `<label>`, buttons are focusable, keyboard traps are avoided, and `aria-live` regions handle async notifications.

## Testing Standards
- In Vanilla JS, write Playwright E2E tests targetting `data-testid` DOM elements.
- Ensure E2E tests mock/handle authentication (if not public) and assert on final visual DOM states (like `.toast--visible`).
