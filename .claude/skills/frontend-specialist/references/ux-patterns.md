# UX Patterns Reference

> Templates and checklists for the UX thinking that always precedes implementation.

---

## Persona Template

Complete one persona card per primary user type before designing any screen.
Extract from `docs/prd/` — specifically §3 User Stories and the Primary User section.

```
┌─────────────────────────────────────────────────────────────────┐
│  PERSONA CARD                                                   │
├─────────────────────────────────────────────────────────────────┤
│  Name:          [Fictional name that humanizes the role]        │
│  Role:          [Job title / context in their day]              │
│  Age range:     [e.g. 28–42]                                    │
│  Tech level:    ○ Non-technical  ○ Moderate  ○ Developer        │
│  Device:        ○ Desktop  ○ Mobile  ○ Both                     │
├─────────────────────────────────────────────────────────────────┤
│  Primary goal:  [What they want to accomplish in this product]  │
│  Secondary goal:[What else they care about]                     │
├─────────────────────────────────────────────────────────────────┤
│  Pain today:    [What frustrates them in the current workflow]  │
│  Fear:          [What they're afraid might go wrong]            │
├─────────────────────────────────────────────────────────────────┤
│  Key scenario:  [The #1 thing they do most often in this UI]    │
│  Success looks: [What "done" feels like for this person]        │
├─────────────────────────────────────────────────────────────────┤
│  Quote:         "[A realistic thing this person would say]"     │
└─────────────────────────────────────────────────────────────────┘
```

### Example — Internal Admin User

```
┌─────────────────────────────────────────────────────────────────┐
│  PERSONA CARD                                                   │
├─────────────────────────────────────────────────────────────────┤
│  Name:          Rafael (Dev Admin)                              │
│  Role:          Backend developer who owns go-base-project      │
│  Age range:     25–35                                           │
│  Tech level:    ● Developer                                     │
│  Device:        ● Desktop                                       │
├─────────────────────────────────────────────────────────────────┤
│  Primary goal:  Inspect and manipulate data without Postman     │
│  Secondary goal:Trigger background jobs and verify they ran     │
├─────────────────────────────────────────────────────────────────┤
│  Pain today:    Must memorize curl commands or use Postman      │
│  Fear:          Accidentally runs process twice on same item    │
├─────────────────────────────────────────────────────────────────┤
│  Key scenario:  Create test data → trigger process → verify     │
│  Success looks: Can do a full cycle in under 30 seconds         │
├─────────────────────────────────────────────────────────────────┤
│  Quote:         "I just need a quick UI to poke the API"        │
└─────────────────────────────────────────────────────────────────┘
```

---

## User Flow Template

Map the primary journey before designing screens. Maximum 7 steps.
If a flow has more than 7 steps, it's two flows.

```
Flow:     [Feature name]
Actor:    [Persona name]
Trigger:  [What causes the user to start this flow]

STEPS
──────────────────────────────────────────────────────────
1.  [Entry point / page they land on]
    └─ System shows: [what they see]

2.  [First action they take]
    └─ System does: [response / state change]
    └─ If error:   [what happens — user recovers here]

3.  [Core interaction]
    └─ System does: [main operation]

4.  [Confirmation / success state]
    └─ System shows: [feedback — toast, redirect, update]

5.  [Where they go next]
──────────────────────────────────────────────────────────
SUCCESS:  [What "done" looks like — data, UI state]
ABORT:    [How user exits gracefully at any point]
```

### Example — Create and Process Dummy

```
Flow:     Create and Process Dummy Item
Actor:    Rafael (Dev Admin)
Trigger:  Rafael needs to test a new background job handler

STEPS
──────────────────────────────────────────────────────────
1.  Lands on / (admin dashboard)
    └─ System shows: stat bar, create form, dummies list

2.  Types text into "Texto" field, clicks "+ Criar"
    └─ System does: POST /dummy → re-fetches list
    └─ If error:   Toast error with message, form stays filled

3.  Sees new row in table with "▶ Processar" button
    └─ System shows: list updated with new item

4.  Clicks "▶ Processar" on the new row
    └─ System does: POST /dummy/process?id=N → re-fetches list
    └─ If error:   Toast error, button re-enables

5.  Success toast appears, row reflects updated state
──────────────────────────────────────────────────────────
SUCCESS:  Item in DB with processed state; toast shown
ABORT:    User can navigate away at any point; no partial state
```

---

## Screen Inventory

Before building, list every screen / state the UI needs:

| Screen | Route | Primary action | Empty state | Error state | Loading state |
|--------|-------|----------------|-------------|-------------|---------------|
| Dashboard | / | Create dummy | "Nenhum item" message | Toast error | Spinner on table |
| Detail (if needed) | /dummy/:id | Process | — | 404 message | Skeleton |

---

## Information Architecture

For multi-page apps, define the navigation structure:

```
[App]
├── / — Dashboard (list + create)
├── /dummy/:id — Detail view
└── /settings — Configuration (future)
```

Navigation decisions:
- **Flat nav** (top bar): ≤ 5 sections, all equally important
- **Sidebar nav**: 5–15 sections, hierarchical, needs collapse on mobile
- **No nav**: Single-feature tools, admin panels

---

## Accessibility Checklist

Run through every page before shipping:

### Keyboard Navigation
- [ ] All interactive elements reachable by `Tab` key in logical order
- [ ] Focus is always visible — no `outline: none` without a visible replacement
- [ ] `Shift+Tab` moves focus backwards
- [ ] Modals trap focus within and return focus to trigger on close
- [ ] Dropdown menus closeable with `Escape`
- [ ] Tables navigable with arrow keys (if data grid)

### Semantic HTML
- [ ] Page has a single `<h1>` that describes the current view
- [ ] Heading hierarchy is logical (`h1` → `h2` → `h3`, no skipping)
- [ ] Navigation is in a `<nav>` element with `aria-label`
- [ ] Main content is in a `<main>` element
- [ ] Forms use `<label>` for every input (explicit association via `for`/`id`)
- [ ] Tables use `<th scope="col|row">` for headers
- [ ] Lists use `<ul>/<ol>` for collections

### ARIA
- [ ] Dynamic regions have `aria-live="polite"` (or `assertive` for urgent alerts)
- [ ] Loading states use `aria-busy="true"` on the container
- [ ] Error messages use `role="alert"` or are linked via `aria-describedby`
- [ ] Icon-only buttons have `aria-label`
- [ ] Disabled elements have `aria-disabled="true"` if using `div`/`span`

### Visual
- [ ] Color contrast ≥ 4.5:1 for normal text, ≥ 3:1 for large text (WCAG AA)
- [ ] Color is not the only way to convey status (always pair with icon or text)
- [ ] Touch targets ≥ 44×44px on mobile
- [ ] Text scales up to 200% without breaking layout

### Testing Tools
```bash
# Automated accessibility audit
npx @axe-core/cli http://localhost:8080

# Lighthouse accessibility score
npx lighthouse http://localhost:8080 --only-categories=accessibility
```

---

## Feedback Pattern Guide

Every async operation needs all four states:

```
State         What to show
────────────────────────────────────────────────────────
IDLE          Normal button / form, no indicator
LOADING       Button disabled, spinner icon, text "Salvando…"
SUCCESS       Toast notification (auto-dismiss 3.5s), UI updated
ERROR         Toast notification (persists until dismissed OR auto 5s)
              Error text near the form field if it's a validation error
```

### Toast Tone Guide
| Situation | Type | Example message (pt-BR) |
|---|---|---|
| Operation succeeded | success | "Dummy criado com sucesso!" |
| Operation failed (API) | error | "Erro ao criar: [mensagem humanizada]" |
| Background action queued | info | "Processamento iniciado." |
| Warn about irreversible | warning | "Esta ação não pode ser desfeita." |

---

## Responsive Breakpoints

```css
/* Mobile first */
/* base (≥0px):   single column, stacked layout */
/* sm  (≥640px):  2-column forms, compact tables */
/* md  (≥768px):  sidebar appears if app uses one */
/* lg  (≥1024px): max-width container, comfortable density */
/* xl  (≥1280px): wider max-width, more breathing room */
```

For the go-base-project admin panels, **desktop-first is acceptable** — internal tools
are overwhelmingly used on desktop. Still check at 768px for tablet use.

---

## Stack Decision Cheatsheet

```
Feature complexity?
│
├── Simple CRUD admin / internal tool / dashboard?
│   └── Vanilla HTML/JS embedded in Go binary ← DEFAULT for this project
│       • Zero build step
│       • Ships inside the Go binary (`go:embed`)
│       • No npm, no node_modules in production
│
├── Rich UI with many components / routes / client state?
│   ├── Team uses React already / prefers hooks / smaller bundle?
│   │   └── React + Vite
│   └── Enterprise / Angular CLI conventions / larger team?
│       └── Angular
│
└── Mixed (Go serves HTML, JS adds interactivity)?
    └── htmx + minimal vanilla JS (progressive enhancement)
        • Render HTML from Go templates
        • htmx for AJAX without writing a full SPA
```

---

## Component Naming Convention

Consistent naming prevents confusion across stack modes:

| Concept | Vanilla class | React component | Angular component |
|---|---|---|---|
| Page shell | `.page` | `XxxPage.tsx` | `xxx-page.component.ts` |
| Feature section | `.section--xxx` | `XxxSection.tsx` | `xxx-section.component.ts` |
| Reusable UI | `.card`, `.btn` | `ui/Card.tsx` | `shared/card.component.ts` |
| Domain-specific | `.dummy-card` | `domain/DummyCard.tsx` | `features/dummy/dummy-card.component.ts` |
| Form | `.form--xxx` | `XxxForm.tsx` | `xxx-form.component.ts` |
