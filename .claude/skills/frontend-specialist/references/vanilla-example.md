# Vanilla HTML/JS Example — Dummy Domain Admin Page

> Complete working example: a single-page admin panel for the Dummy domain,
> embedded in the Go binary and served at `GET /`. Zero build step. Zero npm.
> Connects to the real API at `http://localhost:8080`.

---

## UX Thinking (Step 0) — Done First

### Persona
```
Persona:    Dev Admin (internal user)
Role:       Developer or technical admin managing dummy data
Goal:       Create, view, and process dummy items quickly
Pain:       Currently must use curl / Postman for every operation
Tech level: Developer
Device:     Desktop
Key scenario: Create a batch of dummy items, then trigger processing on specific ones
```

### User Flow
```
Land on page → See list of dummies (or empty state)
    ↓
Fill form → Submit → See new item appear in list with success toast
    ↓
Click "Process" on a row → See status update + success toast
    ↓
Refresh page → State persists (data from API, not memory)
```

### UX Decisions
- Inline table with "Process" action per row — no modal needed for this simple action
- Toast for all async feedback — non-blocking, auto-dismissing
- Empty state illustration + CTA when list is empty — never show a blank screen
- Form stays visible above the list — creation is the primary action

---

## Directory Layout

```
web/
├── static/
│   ├── css/main.css
│   └── js/
│       ├── api.js
│       └── dummy.js
└── templates/
    └── index.html
```

---

## web/static/css/main.css

```css
/* ── Design Tokens ──────────────────────────────────────────────── */
:root {
  --c-bg:           #0f1117;
  --c-surface:      #1a1d27;
  --c-surface-alt:  #22263a;
  --c-border:       #2e3349;
  --c-text:         #e2e8f0;
  --c-text-muted:   #7c8db5;
  --c-primary:      #5b8af5;
  --c-primary-hover:#7aa3ff;
  --c-danger:       #f56565;
  --c-success:      #48bb78;
  --c-warning:      #ecc94b;

  --radius:    8px;
  --radius-lg: 14px;
  --shadow:    0 4px 24px rgba(0,0,0,.35);

  --font: 'Inter', 'Segoe UI', system-ui, sans-serif;
  --mono: 'JetBrains Mono', 'Fira Code', monospace;
}

/* ── Reset ──────────────────────────────────────────────────────── */
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body {
  font-family: var(--font);
  background: var(--c-bg);
  color: var(--c-text);
  min-height: 100vh;
  line-height: 1.6;
}
a { color: var(--c-primary); text-decoration: none; }
a:hover { color: var(--c-primary-hover); }

/* ── Layout ─────────────────────────────────────────────────────── */
.app-header {
  background: var(--c-surface);
  border-bottom: 1px solid var(--c-border);
  padding: 0 32px;
  height: 56px;
  display: flex;
  align-items: center;
  gap: 12px;
}
.app-header__logo {
  font-family: var(--mono);
  font-size: 1rem;
  font-weight: 700;
  color: var(--c-primary);
  letter-spacing: .04em;
}
.app-header__badge {
  font-size: .7rem;
  padding: 2px 8px;
  background: var(--c-surface-alt);
  border: 1px solid var(--c-border);
  border-radius: 999px;
  color: var(--c-text-muted);
}

.main { max-width: 900px; margin: 0 auto; padding: 40px 24px; }

/* ── Card ───────────────────────────────────────────────────────── */
.card {
  background: var(--c-surface);
  border: 1px solid var(--c-border);
  border-radius: var(--radius-lg);
  padding: 28px;
  margin-bottom: 28px;
  box-shadow: var(--shadow);
}
.card__title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--c-text);
  margin-bottom: 20px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.card__title::before {
  content: '';
  display: block;
  width: 3px;
  height: 18px;
  background: var(--c-primary);
  border-radius: 2px;
}

/* ── Form ───────────────────────────────────────────────────────── */
.form-row {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}
.form-group { display: flex; flex-direction: column; gap: 6px; flex: 1; }
.form-label {
  font-size: .8rem;
  font-weight: 500;
  color: var(--c-text-muted);
  text-transform: uppercase;
  letter-spacing: .06em;
}
.form-input {
  background: var(--c-bg);
  border: 1px solid var(--c-border);
  border-radius: var(--radius);
  color: var(--c-text);
  font-family: var(--font);
  font-size: .95rem;
  padding: 10px 14px;
  transition: border-color .15s, box-shadow .15s;
  outline: none;
}
.form-input:focus {
  border-color: var(--c-primary);
  box-shadow: 0 0 0 3px rgba(91,138,245,.18);
}
.form-input::placeholder { color: var(--c-text-muted); }
.form-error {
  font-size: .78rem;
  color: var(--c-danger);
  display: none;
}
.form-error.visible { display: block; }

/* ── Buttons ────────────────────────────────────────────────────── */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  border-radius: var(--radius);
  border: none;
  font-family: var(--font);
  font-size: .9rem;
  font-weight: 600;
  cursor: pointer;
  transition: background .15s, opacity .15s, transform .1s;
  white-space: nowrap;
}
.btn:active { transform: scale(.97); }
.btn:disabled { opacity: .5; cursor: not-allowed; transform: none; }
.btn--primary { background: var(--c-primary); color: #fff; }
.btn--primary:hover:not(:disabled) { background: var(--c-primary-hover); }
.btn--ghost {
  background: transparent;
  color: var(--c-primary);
  border: 1px solid var(--c-border);
}
.btn--ghost:hover:not(:disabled) {
  background: var(--c-surface-alt);
  border-color: var(--c-primary);
}
.btn--danger { background: transparent; color: var(--c-danger); border: 1px solid var(--c-border); }
.btn--danger:hover:not(:disabled) { background: rgba(245,101,101,.1); }
.btn--sm { padding: 5px 12px; font-size: .82rem; }

/* ── Table ──────────────────────────────────────────────────────── */
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: .9rem; }
th {
  text-align: left;
  padding: 10px 14px;
  font-size: .75rem;
  font-weight: 600;
  color: var(--c-text-muted);
  text-transform: uppercase;
  letter-spacing: .06em;
  border-bottom: 1px solid var(--c-border);
}
td {
  padding: 13px 14px;
  border-bottom: 1px solid var(--c-border);
  color: var(--c-text);
}
tr:last-child td { border-bottom: none; }
tr:hover td { background: var(--c-surface-alt); }
.td--id {
  font-family: var(--mono);
  font-size: .82rem;
  color: var(--c-text-muted);
  width: 60px;
}
.td--actions { width: 120px; text-align: right; }

/* ── Empty State ────────────────────────────────────────────────── */
.empty-state {
  text-align: center;
  padding: 56px 24px;
  color: var(--c-text-muted);
}
.empty-state__icon { font-size: 2.5rem; margin-bottom: 12px; }
.empty-state__title { font-size: 1rem; font-weight: 600; color: var(--c-text); margin-bottom: 6px; }
.empty-state__desc { font-size: .875rem; }

/* ── Toast ──────────────────────────────────────────────────────── */
.toast-container {
  position: fixed;
  bottom: 24px;
  right: 24px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  z-index: 9999;
}
.toast {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 13px 18px;
  border-radius: var(--radius);
  font-size: .875rem;
  font-weight: 500;
  box-shadow: var(--shadow);
  transform: translateX(110%);
  transition: transform .25s cubic-bezier(.34,1.56,.64,1);
  max-width: 340px;
  pointer-events: none;
}
.toast.visible { transform: translateX(0); }
.toast--success { background: #1a3329; border: 1px solid var(--c-success); color: var(--c-success); }
.toast--error   { background: #331a1a; border: 1px solid var(--c-danger);  color: var(--c-danger);  }
.toast--info    { background: #1a2333; border: 1px solid var(--c-primary); color: var(--c-primary); }

/* ── Loading spinner ────────────────────────────────────────────── */
.spinner {
  width: 16px; height: 16px;
  border: 2px solid rgba(255,255,255,.2);
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin .6s linear infinite;
  display: inline-block;
}
@keyframes spin { to { transform: rotate(360deg); } }

/* ── Stats bar ──────────────────────────────────────────────────── */
.stats { display: flex; gap: 16px; margin-bottom: 28px; }
.stat {
  flex: 1;
  background: var(--c-surface);
  border: 1px solid var(--c-border);
  border-radius: var(--radius-lg);
  padding: 20px 24px;
}
.stat__value { font-size: 1.75rem; font-weight: 700; font-family: var(--mono); }
.stat__label { font-size: .78rem; color: var(--c-text-muted); margin-top: 4px; text-transform: uppercase; letter-spacing: .05em; }
```

---

## web/static/js/api.js

```javascript
// api.js — Fetch wrapper for the go-base-project API
const API_BASE = window.__API_BASE__ || '';

async function apiFetch(method, path, body = null) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  if (body !== null) opts.body = JSON.stringify(body);

  const res = await fetch(API_BASE + path, opts);

  if (!res.ok) {
    const text = await res.text().catch(() => `HTTP ${res.status}`);
    throw new Error(text || `HTTP ${res.status}`);
  }
  if (res.status === 204 || res.headers.get('content-length') === '0') return null;
  const ct = res.headers.get('content-type') || '';
  return ct.includes('application/json') ? res.json() : res.text();
}

export const api = {
  get:    path        => apiFetch('GET',    path),
  post:   (path, body) => apiFetch('POST',   path, body),
  put:    (path, body) => apiFetch('PUT',    path, body),
  delete: path        => apiFetch('DELETE', path),
};
```

---

## web/static/js/dummy.js

```javascript
import { api } from './api.js';

// ── State ──────────────────────────────────────────────────────────
let dummies = [];

// ── Toast ──────────────────────────────────────────────────────────
const toastContainer = document.getElementById('toast-container');

function toast(message, type = 'success') {
  const el = document.createElement('div');
  el.className = `toast toast--${type}`;
  el.setAttribute('role', 'status');
  el.setAttribute('aria-live', 'polite');
  el.textContent = message;
  toastContainer.appendChild(el);
  requestAnimationFrame(() => {
    requestAnimationFrame(() => el.classList.add('visible'));
  });
  setTimeout(() => {
    el.classList.remove('visible');
    el.addEventListener('transitionend', () => el.remove(), { once: true });
  }, 3500);
}

// ── Validation ─────────────────────────────────────────────────────
function validateCreate(data) {
  const errors = {};
  if (!data.text?.trim()) errors.text = 'O texto não pode ser vazio.';
  else if (data.text.length > 255) errors.text = 'Máximo de 255 caracteres.';
  return errors;
}

// ── Render ─────────────────────────────────────────────────────────
function renderStats() {
  document.getElementById('stat-total').textContent = dummies.length;
}

function renderList() {
  const tbody = document.getElementById('dummy-tbody');
  const emptyState = document.getElementById('empty-state');

  if (dummies.length === 0) {
    tbody.innerHTML = '';
    emptyState.hidden = false;
    return;
  }

  emptyState.hidden = true;
  tbody.innerHTML = dummies.map(d => `
    <tr data-id="${d.id}">
      <td class="td--id">#${d.id}</td>
      <td>${escapeHtml(d.text)}</td>
      <td class="td--actions">
        <button
          class="btn btn--ghost btn--sm"
          data-action="process"
          data-id="${d.id}"
          data-testid="process-btn-${d.id}"
          aria-label="Processar dummy ${d.id}"
        >
          ▶ Processar
        </button>
      </td>
    </tr>
  `).join('');
}

// ── Load Data ──────────────────────────────────────────────────────
async function loadDummies() {
  const listCard = document.getElementById('list-card');
  listCard.setAttribute('aria-busy', 'true');
  try {
    const data = await api.get('/dummy/all');
    dummies = Array.isArray(data) ? data : [];
    renderStats();
    renderList();
  } catch (e) {
    toast('Erro ao carregar dados: ' + e.message, 'error');
  } finally {
    listCard.removeAttribute('aria-busy');
  }
}

// ── Create ─────────────────────────────────────────────────────────
const form = document.getElementById('create-form');
const textInput = document.getElementById('dummy-text');
const textError = document.getElementById('text-error');
const submitBtn = document.getElementById('submit-btn');

form.addEventListener('submit', async e => {
  e.preventDefault();
  const data = { text: textInput.value };

  // Validate
  const errors = validateCreate(data);
  textError.textContent = errors.text || '';
  textError.classList.toggle('visible', !!errors.text);
  if (errors.text) { textInput.focus(); return; }

  // Submit
  submitBtn.disabled = true;
  submitBtn.innerHTML = '<span class="spinner" aria-hidden="true"></span> Salvando…';
  try {
    await api.post('/dummy', data);
    textInput.value = '';
    textError.classList.remove('visible');
    toast('Dummy criado com sucesso!', 'success');
    await loadDummies();
  } catch (e) {
    toast('Erro ao criar: ' + humanizeError(e.message), 'error');
  } finally {
    submitBtn.disabled = false;
    submitBtn.innerHTML = '+ Criar';
  }
});

// ── Process ────────────────────────────────────────────────────────
document.getElementById('dummy-tbody').addEventListener('click', async e => {
  const btn = e.target.closest('[data-action="process"]');
  if (!btn) return;

  const id = btn.dataset.id;
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner" aria-hidden="true"></span>';

  try {
    await api.post(`/dummy/process?id=${id}`);
    toast(`Dummy #${id} processado!`, 'success');
    await loadDummies();
  } catch (e) {
    toast('Erro ao processar: ' + humanizeError(e.message), 'error');
    btn.disabled = false;
    btn.innerHTML = '▶ Processar';
  }
});

// ── Helpers ────────────────────────────────────────────────────────
function escapeHtml(str) {
  return str.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
            .replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}

function humanizeError(msg) {
  if (msg.includes('invalid id'))   return 'ID inválido.';
  if (msg.includes('not found'))    return 'Item não encontrado.';
  if (msg.includes('EOF'))          return 'Corpo da requisição inválido.';
  return msg || 'Erro desconhecido. Tente novamente.';
}

// ── Init ───────────────────────────────────────────────────────────
loadDummies();
```

---

## web/templates/index.html

```html
<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Dummy Admin — go-base-project</title>
  <link rel="stylesheet" href="/static/css/main.css">
  <!-- Set API base URL for non-same-origin deployments -->
  <script>window.__API_BASE__ = '';</script>
</head>
<body>

  <header class="app-header" role="banner">
    <span class="app-header__logo">⬡ go-base-project</span>
    <span class="app-header__badge">admin</span>
  </header>

  <main class="main" role="main">

    <!-- Stats bar -->
    <div class="stats" role="region" aria-label="Estatísticas">
      <div class="stat">
        <div class="stat__value" id="stat-total" aria-live="polite">—</div>
        <div class="stat__label">Total de Dummies</div>
      </div>
    </div>

    <!-- Create form -->
    <section class="card" aria-labelledby="create-title">
      <h2 class="card__title" id="create-title">Criar Dummy</h2>
      <form id="create-form" novalidate>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label" for="dummy-text">Texto</label>
            <input
              class="form-input"
              id="dummy-text"
              name="text"
              type="text"
              placeholder="Digite o conteúdo do dummy…"
              autocomplete="off"
              maxlength="255"
              data-testid="dummy-text-input"
              aria-required="true"
              aria-describedby="text-error"
            >
            <span class="form-error" id="text-error" role="alert" data-testid="field-error-text"></span>
          </div>
          <button
            class="btn btn--primary"
            type="submit"
            id="submit-btn"
            data-testid="dummy-submit-btn"
          >
            + Criar
          </button>
        </div>
      </form>
    </section>

    <!-- List -->
    <section class="card" id="list-card" aria-labelledby="list-title" aria-busy="false">
      <h2 class="card__title" id="list-title">Dummies</h2>

      <!-- Empty state -->
      <div id="empty-state" hidden aria-hidden="true">
        <div class="empty-state">
          <div class="empty-state__icon">📭</div>
          <div class="empty-state__title">Nenhum dummy cadastrado</div>
          <div class="empty-state__desc">Crie o primeiro usando o formulário acima.</div>
        </div>
      </div>

      <!-- Table -->
      <div class="table-wrap">
        <table role="table" aria-label="Lista de dummies" data-testid="dummy-list">
          <thead>
            <tr>
              <th scope="col">ID</th>
              <th scope="col">Texto</th>
              <th scope="col"><span class="sr-only">Ações</span></th>
            </tr>
          </thead>
          <tbody id="dummy-tbody" role="rowgroup">
          </tbody>
        </table>
      </div>
    </section>

  </main>

  <!-- Toast container -->
  <div
    id="toast-container"
    class="toast-container"
    role="region"
    aria-label="Notificações"
    aria-live="polite"
    data-testid="toast-container"
  ></div>

  <script type="module" src="/static/js/dummy.js"></script>
</body>
</html>
```

---

## Go: Serving Static Files (cmd/api)

Add to `cmd/api/modules/frontend.go`:
```go
package modules

import (
    "embed"
    "net/http"

    "github.com/gorilla/mux"
    "go.uber.org/fx"
)

//go:embed ../../../web/static
var StaticFiles embed.FS

//go:embed ../../../web/templates/index.html
var IndexHTML []byte

var FrontendModule = fx.Module("frontend",
    fx.Invoke(func(router *mux.Router) {
        // Serve static assets
        router.PathPrefix("/static/").Handler(
            http.FileServer(http.FS(StaticFiles)),
        )
        // Serve the SPA shell for all non-API routes
        router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Content-Type", "text/html; charset=utf-8")
            w.Write(IndexHTML)
        }).Methods("GET")
    }),
)
```

Then add `FrontendModule` to `fx.New(...)` in `cmd/api/main.go`.

---

## Playwright Tests

```typescript
// tests/frontend/dummy-page.spec.ts
import { test, expect } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  await page.goto('http://localhost:8080');
});

test('page loads and shows header', async ({ page }) => {
  await expect(page).toHaveTitle(/Dummy Admin/);
  await expect(page.locator('.app-header__logo')).toBeVisible();
});

test('creates a dummy and shows it in the list', async ({ page }) => {
  await page.fill('[data-testid="dummy-text-input"]', 'e2e test item');
  await page.click('[data-testid="dummy-submit-btn"]');
  await expect(page.locator('[data-testid="toast-container"]')).toContainText('sucesso');
  await expect(page.locator('[data-testid="dummy-list"]')).toContainText('e2e test item');
});

test('shows validation error for empty submission', async ({ page }) => {
  await page.click('[data-testid="dummy-submit-btn"]');
  await expect(page.locator('[data-testid="field-error-text"]')).toBeVisible();
  await expect(page.locator('[data-testid="field-error-text"]')).toContainText('vazio');
});

test('processes a dummy item', async ({ page }) => {
  // Pre-create via API so list is not empty
  await page.request.post('http://localhost:8080/dummy', {
    data: { text: 'to be processed' },
  });
  await page.reload();
  const row = page.locator('tr', { hasText: 'to be processed' });
  await row.locator('[data-action="process"]').click();
  await expect(page.locator('[data-testid="toast-container"]')).toContainText('processado');
});
```

---

## Makefile Additions

```makefile
serve-frontend: ## Run API (serves frontend at :8080)
	go run cmd/api/main.go

test-frontend: ## Run Playwright frontend tests (requires server running)
	npx playwright test tests/frontend/

test-frontend-ui: ## Open Playwright UI mode
	npx playwright test tests/frontend/ --ui
```
