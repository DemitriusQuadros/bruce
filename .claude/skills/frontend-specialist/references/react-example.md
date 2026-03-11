# React Example — Dummy Domain SPA (Vite + TypeScript)

> A standalone React SPA that connects to the Go API.
> Use this mode when the UI is too complex for vanilla JS, or requires
> a component library, rich client-side routing, or a dedicated deployment.

---

## Setup

```bash
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install
npm install @tanstack/react-query
npm install -D @testing-library/react @testing-library/jest-dom vitest jsdom
npm install -D @playwright/test
npx playwright install
```

---

## vite.config.ts

```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  resolve: { alias: { '@': '/src' } },
  server: {
    proxy: {
      '/dummy': 'http://localhost:8080',
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/tests/setup.ts'],
    globals: true,
  },
});
```

---

## src/api/client.ts

```typescript
const BASE = import.meta.env.VITE_API_BASE ?? '';

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

async function apiFetch<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    const text = await res.text().catch(() => `HTTP ${res.status}`);
    throw new ApiError(res.status, text);
  }

  if (res.status === 204) return null as T;
  return res.json();
}

export const apiClient = {
  getAllDummies: () => apiFetch<Dummy[]>('GET', '/dummy/all'),
  getDummy:     (id: number) => apiFetch<Dummy>('GET', `/dummy?id=${id}`),
  createDummy:  (text: string) => apiFetch<void>('POST', '/dummy', { text }),
  processDummy: (id: number) => apiFetch<void>('POST', `/dummy/process?id=${id}`),
};

export interface Dummy {
  id: number;
  text: string;
}
```

---

## src/hooks/useDummies.ts

```typescript
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/api/client';

export function useDummies() {
  return useQuery({
    queryKey: ['dummies'],
    queryFn: apiClient.getAllDummies,
    initialData: [],
  });
}

export function useCreateDummy() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (text: string) => apiClient.createDummy(text),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dummies'] }),
  });
}

export function useProcessDummy() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiClient.processDummy(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['dummies'] }),
  });
}
```

---

## src/components/domain/DummyForm.tsx

```tsx
import { useState } from 'react';
import { useCreateDummy } from '@/hooks/useDummies';
import styles from './DummyForm.module.css';

interface Props { onSuccess?: () => void; }

function validate(text: string): string | null {
  if (!text.trim()) return 'O texto é obrigatório.';
  if (text.length > 255) return 'Máximo de 255 caracteres.';
  return null;
}

export function DummyForm({ onSuccess }: Props) {
  const [text, setText] = useState('');
  const [fieldError, setFieldError] = useState<string | null>(null);
  const mutation = useCreateDummy();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const err = validate(text);
    if (err) { setFieldError(err); return; }
    setFieldError(null);

    try {
      await mutation.mutateAsync(text);
      setText('');
      onSuccess?.();
    } catch {
      // Error shown via toast in parent
    }
  };

  return (
    <form onSubmit={handleSubmit} className={styles.form} noValidate>
      <div className={styles.field}>
        <label htmlFor="dummy-text" className={styles.label}>Texto</label>
        <input
          id="dummy-text"
          className={styles.input}
          type="text"
          value={text}
          onChange={e => setText(e.target.value)}
          placeholder="Conteúdo do dummy…"
          aria-required="true"
          aria-describedby={fieldError ? 'dummy-text-error' : undefined}
          aria-invalid={!!fieldError}
          data-testid="dummy-text-input"
          maxLength={255}
        />
        {fieldError && (
          <span
            id="dummy-text-error"
            role="alert"
            className={styles.error}
            data-testid="field-error-text"
          >
            {fieldError}
          </span>
        )}
      </div>
      <button
        type="submit"
        className={styles.btn}
        disabled={mutation.isPending}
        data-testid="dummy-submit-btn"
      >
        {mutation.isPending ? 'Salvando…' : '+ Criar'}
      </button>
    </form>
  );
}
```

---

## Unit Tests — src/tests/DummyForm.test.tsx

```tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { DummyForm } from '@/components/domain/DummyForm';
import { vi } from 'vitest';

// Mock the API client
vi.mock('@/api/client', () => ({
  apiClient: {
    createDummy: vi.fn().mockResolvedValue(undefined),
    getAllDummies: vi.fn().mockResolvedValue([]),
  },
}));

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

test('renders form with text input and submit button', () => {
  render(<DummyForm />, { wrapper: Wrapper });
  expect(screen.getByLabelText(/texto/i)).toBeInTheDocument();
  expect(screen.getByRole('button', { name: /criar/i })).toBeInTheDocument();
});

test('shows validation error when submitting empty text', async () => {
  render(<DummyForm />, { wrapper: Wrapper });
  fireEvent.click(screen.getByRole('button', { name: /criar/i }));
  await waitFor(() => {
    expect(screen.getByTestId('field-error-text')).toBeVisible();
    expect(screen.getByTestId('field-error-text')).toHaveTextContent(/obrigatório/i);
  });
});

test('calls createDummy and clears input on success', async () => {
  const { apiClient } = await import('@/api/client');
  const onSuccess = vi.fn();
  render(<DummyForm onSuccess={onSuccess} />, { wrapper: Wrapper });

  fireEvent.change(screen.getByTestId('dummy-text-input'), {
    target: { value: 'meu texto de teste' },
  });
  fireEvent.click(screen.getByRole('button', { name: /criar/i }));

  await waitFor(() => {
    expect(apiClient.createDummy).toHaveBeenCalledWith('meu texto de teste');
    expect(onSuccess).toHaveBeenCalled();
    expect(screen.getByTestId('dummy-text-input')).toHaveValue('');
  });
});

test('does not submit when text exceeds 255 characters', async () => {
  const { apiClient } = await import('@/api/client');
  render(<DummyForm />, { wrapper: Wrapper });

  fireEvent.change(screen.getByTestId('dummy-text-input'), {
    target: { value: 'a'.repeat(256) },
  });
  fireEvent.click(screen.getByRole('button', { name: /criar/i }));

  await waitFor(() => {
    expect(screen.getByTestId('field-error-text')).toHaveTextContent(/255/);
    expect(apiClient.createDummy).not.toHaveBeenCalled();
  });
});
```

---

## Playwright E2E — tests/e2e/dummy.spec.ts

```typescript
import { test, expect } from '@playwright/test';

const BASE = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:5173';

test.beforeEach(async ({ page }) => {
  await page.goto(BASE);
});

test('full create and display flow', async ({ page }) => {
  await page.fill('[data-testid="dummy-text-input"]', 'playwright test item');
  await page.click('[data-testid="dummy-submit-btn"]');
  await expect(page.locator('[data-testid="dummy-list"]')).toContainText('playwright test item');
});

test('shows validation error on empty submit', async ({ page }) => {
  await page.click('[data-testid="dummy-submit-btn"]');
  await expect(page.locator('[data-testid="field-error-text"]')).toBeVisible();
});
```

---

## package.json scripts

```json
{
  "scripts": {
    "dev":        "vite",
    "build":      "tsc && vite build",
    "test":       "vitest run",
    "test:watch": "vitest",
    "test:e2e":   "playwright test",
    "test:e2e:ui":"playwright test --ui",
    "lint":       "eslint src --ext .ts,.tsx"
  }
}
```
