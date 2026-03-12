# Phase 1 Task Breakdown (Weeks 1–6)

**26 sequenced backend tasks for the Bruce v2 agentic engine core loop**

All tasks are designed to be completed sequentially with dependencies noted. See the full PRD (`bruce-v2-agentic.md`) for context and success criteria.

---

## Task Groups

### Group A: Foundation (Tasks 1–8) — Weeks 1–2
Build the tool execution architecture and extend domain models.

| # | Task | Dependencies | Est. Hours | Details |
|---|------|--------------|-----------|---------|
| 1 | Extend domain models: add `Tool`, `ToolCall`, `ExecutionResult` | None | 4 | New domain structs in `internal/domain/`. Tool has name, description, params schema, handler func. |
| 2 | Design tool registry interface | Task 1 | 3 | `internal/tools/registry.go` — registry stores Tool, dispatches by name, error boundaries per tool. |
| 3 | Implement tool registry with locking + error isolation | Task 2 | 6 | Registry can register, list, invoke tools safely. Failed tool doesn't crash others. |
| 4 | Implement Claude `tool_use` API integration | None | 8 | Extend `internal/ai/claude.go` to parse `tool_use` response blocks from Claude API. Validate tool names. |
| 5 | Add `confirmations` table to SQLite schema | None | 3 | `internal/database/schema.sql` — store pending approvals, track status, timestamps. |
| 6 | Build worker task processor for tool execution | Tasks 2, 4 | 10 | `internal/worker/processor.go` extended: parse tool calls, dispatch via registry, wait for approval if needed. |
| 7 | Add execution confirmation table to SQLite | Task 5 | 3 | Repository + domain entity for confirmations CRUD. |
| 8 | Extend web UI domain: add Confirmations entity | Tasks 1, 7 | 2 | Domain struct for pending actions. |

**Milestone**: Tool registry exists, Claude's `tool_use` is parseable, confirmations can be stored and retrieved.

---

### Group B: Read Tools (Tasks 9–14) — Weeks 2–3
Implement the Phase 1 read-only tools (info retrieval).

| # | Task | Dependencies | Est. Hours | Details |
|---|------|--------------|-----------|---------|
| 9 | Implement Gmail API connector (list + fetch) | Task 3 | 12 | OAuth setup, list messages, fetch full content. Handler in `internal/tools/gmail.go`. Test with real API (or mock). |
| 10 | Implement Google Calendar connector | Task 3 | 8 | Query events for date range. Parse metadata (title, attendees, time). Handler in `internal/tools/calendar.go`. |
| 11 | Implement URL fetch tool | Task 3 | 6 | HTTP GET any URL, strip boilerplate, return clean text. Self-contained, no API keys. Error handling for timeouts/invalid URLs. |
| 12 | Implement local file I/O tools (read + write) | Task 3 | 6 | Read text files, write generated content. Path validation (no `..`, no traversal). Handlers in `internal/tools/files.go`. |
| 13 | Implement PDF generation tool | Tasks 3, 12 | 8 | Convert markdown/text to PDF. Save to disk. Integrate with Discord file upload. Use `github.com/go-pdf/fpdf` or similar. |
| 14 | Test all read tools end-to-end | Tasks 9–13 | 8 | Integration tests: ask Claude for Gmail list, URL fetch, Calendar events. Verify results in chat. |

**Milestone**: User can fetch Gmail, Calendar, URLs, and generate PDFs from chat.

---

### Group C: Connectors + Dispatch (Tasks 15–17) — Week 3
Add new connector and enhance existing ones to support tool outputs (files, etc.).

| # | Task | Dependencies | Est. Hours | Details |
|---|------|--------------|-----------|---------|
| 15 | Implement Telegram connector | None | 8 | Receive messages via polling API, enqueue for processing, dispatch responses back. Simpler than WhatsApp (no QR pairing). |
| 16 | Extend Discord connector for file upload | None | 4 | Discord has file attachment support. Enhance `internal/connectors/discord/dispatcher.go` to send PDFs. |
| 17 | Enhance WhatsApp connector for file delivery | None | 4 | WhatsApp media upload API. Enhance `internal/connectors/whatsapp/dispatcher.go` for files. (Lower priority if blocked by media API complexity.) |

**Milestone**: Telegram receives messages, Discord/WhatsApp can send files from tool outputs.

---

### Group D: Approval Flow (Tasks 18–20) — Week 4
Build the safety gate for destructive operations.

| # | Task | Dependencies | Est. Hours | Details |
|---|------|--------------|-----------|---------|
| 18 | Add tool execution confirmation to REST API | Tasks 7, 6 | 6 | `POST /api/v1/confirmations` (approve), `GET /api/v1/confirmations` (list pending). Handler in `internal/api/handlers/confirmations.go`. |
| 19 | Extend web UI: Confirmations tab | Tasks 8, 18 | 8 | Display pending tool calls with action preview. Approve/deny buttons. Status log of past actions. |
| 20 | Integrate confirmation prompt into worker | Tasks 6, 18 | 6 | Worker pauses on `confirmed: false`. Poll for user approval. Resume on approval or timeout. Deny flow doesn't execute tool. |

**Milestone**: User sees a confirmation prompt before write operations, can approve or deny, action reflects choice.

---

### Group E: Error Recovery (Tasks 21–23) — Week 5
Resilience and error handling.

| # | Task | Dependencies | Est. Hours | Details |
|---|------|--------------|-----------|---------|
| 21 | Build error recovery flow | Task 6 | 8 | On tool timeout/API error, capture error details. Encode as system message. Send next message to Claude with error context so it can retry or choose alternative. Test with intentional Gmail failure. |
| 22 | Add rate limiting for external APIs | Tasks 9, 10, 11 | 6 | Per-tool rate limiting (e.g., Gmail max 1 req/sec). Exponential backoff on 429. Return sentinel error to Asynq for retry logic. |
| 23 | Add timeout safeguards to all tools | Tasks 3, 9–13 | 4 | All tool invocations have max timeout (Gmail 5s, Calendar 5s, URL 10s, PDF 20s). Context timeout enforced in worker. |

**Milestone**: Failed tools don't crash worker, Claude learns from errors, rate limits are respected.

---

### Group F: Testing + Hardening (Tasks 24–26) — Week 5–6
Quality assurance.

| # | Task | Dependencies | Est. Hours | Details |
|---|------|--------------|-----------|---------|
| 24 | Input validation + sanitization | Tasks 9–13, 15 | 6 | Prevent injection attacks (Gmail filters), path traversal (file I/O), URL whitelisting if needed. SQLite parameterized queries everywhere. |
| 25 | Documentation: tool authoring guide | All | 4 | Developer guide for adding new tools. Registry pattern, error boundary, timeout defaults, testing strategy. |
| 26 | E2E tests: full agent loop | All | 8 | Test suite: Claude asks for info → approve PDF → tool executes → file delivered to Discord. All connectors. Asynq retry scenarios. |

**Milestone**: Code is production-ready for MVP. All 26 tasks complete, tests passing, no known crashes.

---

## Dependency Graph (High-Level)

```
Task 1 (Domain) → Tasks 2, 3, 4 (Architecture)
                          ↓
Tasks 9–13 (Read Tools) + Task 15 (Telegram) ← Task 3
                          ↓
Task 6 (Worker Processor) → Task 20 (Approval)
                          ↓
Tasks 21–26 (Error Recovery, Testing)
```

---

## Week-by-Week Allocation

### Week 1–2: Foundation (Tasks 1–8)
- Domain models, registry, Claude tool_use parsing, worker foundation
- ~40–45 hours
- Deliverable: Tool registry accepts registrations, Claude can parse `tool_use` blocks, confirmations stored in DB

### Week 2–3: Read Tools (Tasks 9–14)
- Gmail, Calendar, URL, files, PDF
- ~46–50 hours
- Deliverable: User can fetch Gmail and calendar from chat, PDFs are generated

### Week 3–4: Connectors + Approval (Tasks 15–20)
- Telegram, Discord/WhatsApp files, approval UI
- ~30–36 hours
- Deliverable: Telegram receives messages, approval flow blocks write ops

### Week 4–5: Error Recovery (Tasks 21–23)
- Timeout safeguards, error feedback, rate limiting
- ~18–20 hours
- Deliverable: Failed tools don't crash, Claude retries intelligently

### Week 5–6: Testing (Tasks 24–26)
- Input validation, docs, E2E tests
- ~18–20 hours
- Deliverable: All tests pass, code is hardened, docs are written

**Total**: ~150–170 hours over 6 weeks = ~25–28 hours/week (doable for 1–2 engineers)

---

## Success Criteria (End of Phase 1)

- ✓ Claude successfully calls Gmail tool, user sees email list in chat
- ✓ Claude successfully calls Calendar tool, returns next 3 days
- ✓ Claude successfully calls URL tool, returns clean summary
- ✓ User asks for PDF, sees confirmation, clicks approve, file is written and sent via Discord
- ✓ User clicks deny on GitHub issue (not in Phase 1, but approval flow tested with PDF)
- ✓ Telegram messages are received and responded to correctly
- ✓ All error scenarios are handled (timeout, rate limit, API 5xx)
- ✓ No tool execution crashes the worker; all errors are logged
- ✓ E2E tests pass

---

## Notes for go-backend-dev Agent

1. **Start with Group A (foundation)**: Registry and tool_use parsing are prerequisites. Don't skip or parallelize.
2. **Read tools first**: Gmail, Calendar, URL are low-risk and high-value. Validate the core loop with reads before adding writes.
3. **Approval flow is critical**: This is the safety gate for Phase 2 writes (GitHub, Notion). Get the UX right in Phase 1.
4. **Test early + often**: Integration tests should start in Week 2 with mocked APIs. Real API tests in Week 3+.
5. **Document as you go**: Tool authoring guide (Task 25) should be written by Week 5, not at the end.

---

## Questions for Product Manager

- Should Telegram be implemented in Phase 1, or deferred to Phase 2 if the primary focus is Gmail/Calendar/URL?
- Is approval required for *all* write operations, or should some (e.g., "save to my Downloads folder") auto-approve?
- Should the confirmation prompt timeout? (Suggested: 5 minutes, then auto-deny.)

