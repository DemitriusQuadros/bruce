# Bruce v2: Agentic Personal Assistant

**Product Version**: 2.0
**Status**: Specification Complete (Technical Architecture Validated)
**Last Updated**: 2026-03-12

---

## 1. Product Overview

**One-liner**: Bruce evolves from a chat relay into an agentic personal assistant that reads, writes, and executes actions across your digital ecosystem (Gmail, Google Drive, GitHub, local databases) on your behalf.

**Problem Statement**

You own the chatbot architecture (Claude + connectors) but you're trapped in a reply-only workflow. Bruce can read your messages and respond, but it cannot act. You find yourself context-switching to Gmail, Google Docs, GitHub, or your local filesystem — breaking the conversational flow. You want Claude to fetch your calendar, create GitHub issues, write files, or query your local databases without you having to leave the chat.

**Solution**

Bruce v2 integrates Anthropic's `tool_use` API to transform Claude into an agentic executor. Claude receives a curated list of "tools" it can invoke — each backed by a Go function that runs locally. Tools span read operations (fetch Gmail messages, read GitHub branches, query SQLite) and write operations (create Notion pages, approve PRs, generate PDFs). The agent decides when to call tools, Bruce executes them, and Claude incorporates the results into its response. All actions preserve the single UI: Bruce remains a chat interface, but now it's the control plane for your entire digital life.

**Primary User**

A technical individual (engineer, researcher, PM) who lives in their terminal and chat applications, wants to automate repetitive digital tasks without context-switching, and trusts local execution over SaaS black boxes.

**Stage Assumption**

**MVP (Seed stage)**: Core tool execution engine + phase 1 tools (Gmail, Google Calendar, URL reading, local file I/O, PDF generation, Telegram connector). No multi-user auth, no cloud persistence, no API gateway. Targets single-user, self-hosted deployment on modest hardware (512MB–2GB RAM) with SQLite and Redis queuing. Proves the core agentic loop works before scaling to multi-provider LLMs or complex integrations.

---

## 2. Goals & Success Metrics

| Goal | Launch Metric (Week 6) | 6-Month Target | Rationale |
|------|------------------------|----------------|-----------|
| **Prove agent loop works** | Claude successfully calls 1 tool (fetch URL) without hallucinating | 10+ tools working, 0 false positives in execution calls | Shows core loop (tool invocation, execution, result integration) is solid |
| **Eliminate context-switching on reads** | User can fetch Gmail subject lines and Google Docs content from chat, zero manual copying | User delegates 80% of read tasks to Bruce (info retrieval) | Validates read-heavy workflow (Phase 1 focus) |
| **Enable safe local writes** | User can generate PDFs and create files via chat with explicit confirmation | User safely executes 5+ write operations/day (Notion, GitHub, files) | Write ops drive stickiness; explicit approval mitigates risk |
| **Achieve connector parity** | Discord + WhatsApp both dispatch tool results (text + files) correctly | Telegram connector added; all 3 connectors support 80% of tool outputs | Delivers tools through user's preferred chat channel |
| **Adoption readiness** | First 5 external testers run v2 standalone for 1 week without crashing | 50+ self-hosted instances, sub-1% error rate in tool execution | Signals production-readiness for single-user deployment |

---

## 3. User Stories

### Read Operations (Info Retrieval — Phase 1 Focus)

**As a** engineer checking email before a standup, **I want to** ask "show me my last 5 Gmail messages" **so that** I don't have to open Gmail and can stay focused on the chat.

**As a** researcher working on a document, **I want to** ask "fetch the content from this URL and summarize it" **so that** I avoid leaving my chat app to open a browser.

**As a** PM managing a project, **I want to** ask "list my Google Calendar events for tomorrow" **so that** I see my schedule without opening Calendar.

**As a** developer troubleshooting a bug, **I want to** ask "read my error log from the last 30 minutes" **so that** I can diagnose without context-switching to logs/dashboard.

### Write Operations (Local Action — Phase 1+)

**As a** busy engineer, **I want to** ask Bruce to "create a GitHub issue with this title and description" **so that** I can capture ideas without leaving chat.

**As a** note-taker, **I want to** ask Bruce to "save this conversation as a PDF to my Downloads folder" **so that** I have a searchable archive without manual export.

**As a** researcher, **I want to** ask Bruce to "create a Notion page with these sections" **so that** I can structure notes immediately during brainstorm.

### Safety & Control

**As a** user concerned about accidental actions, **I want to** be shown a confirmation prompt before Bruce executes any destructive tool (create issue, write to DB, delete) **so that** I can review before it happens.

**As a** user managing multiple connectors, **I want to** ask Bruce a question on Discord and get the full response with file attachments (if tools generated PDFs) **so that** my workflow is unified across chat apps.

---

## 4. Feature List with MoSCoW Prioritization

| Feature | Priority | Description | Why |
|---------|----------|-------------|-----|
| **Tool execution engine** | M | Core loop: Claude calls tool, Bruce executes locally, Claude processes result | Without this, no agentic behavior exists |
| **Tool registry & dispatch** | M | Centralized registry mapping tool names to Go functions; safe invocation with error isolation | Enables future tool additions; prevents a failed tool from crashing worker |
| **Gmail read (list + fetch)** | M | Query Gmail API for recent messages; parse sender, subject, snippet | First read tool; validates API integration + message parsing |
| **Google Calendar read** | M | Fetch calendar events for a time range; return title, time, attendees | Low-risk second read tool; common use case |
| **URL reading (fetch + parse)** | M | HTTP GET any URL, strip boilerplate, return clean text | Self-contained; no API keys; high utility |
| **Local file I/O (read + write)** | M | Read text files from disk, write generated content to specified paths | Enables PDF generation handoff; foundational for local actions |
| **PDF generation** | M | Convert text/markdown to PDF, save to disk; integrate with Discord file upload | Enables documentation workflows |
| **Telegram connector** | M | Receive messages, enqueue for processing, dispatch responses back | Phase 1 connector (easier than WhatsApp QR pairing); complements Discord |
| **Tool execution confirmation UI** | M | Web UI shows pending destructive actions, user clicks approve/deny | Prevents accidental GitHub issue creation or data loss |
| **Explicit action approval flow** | M | After Claude outputs a tool invocation with `confirmed: false`, pause and prompt user | Gating for write ops; UX centerpiece of Phase 1 |
| **Error recovery in worker** | S | If tool call fails (timeout, API error), worker logs, returns error to Claude, retries message | Improves reliability; Claude learns from failed calls |
| **Google Docs read** | S | Fetch document content and metadata; return formatted text | High-value but requires OAuth setup; deferred to Phase 2 |
| **GitHub read (branches, issues, PRs)** | S | List branches, read PR/issue metadata; query local Git repos | Complements write ops; deferred pending write ops validation |
| **Notion read (query pages, tables)** | S | Query Notion database, return paginated results | Reference integration; Phase 2 pairs with write |
| **Trello read** | S | Fetch board/card state; query by list | Lower priority than Notion; Phase 2+ |
| **Local SQLite connector** | S | Query user-owned DB (read + write); parameterized to prevent injection | Enables structured data workflows; Phase 2 pending safety review |
| **Gemini as fallback provider** | S | Support Google Gemini alongside Claude; switch via config or session override | Reduces cost; improves availability; requires multi-provider refactor in Phase 2 |
| **Git write ops (commit, push)** | C | Create commits and push to branch; read-only by default, approval required | Powerful but risky; defer until approval UX solidified (Phase 2+) |
| **GitHub write (create/update issues, PRs)** | C | Create issues/PRs with approval; auto-fill templates | High-value but requires permission management; Phase 2 |
| **Notion write** | C | Create/update pages and database rows; templates + approval | Powerful; pairs with read ops; Phase 2 |
| **WhatsApp file delivery** | C | Send files (PDFs, images) via WhatsApp | Currently text-only; requires media upload API; Phase 2 |
| **Proactive task scheduling (crons)** | C | Schedule Claude to run queries/reports daily/weekly; push results to chat | Shifts from request/response to proactive delivery; Phase 3 |
| **Persistent memory (embeddings)** | C | Vector index of past conversations and tool outputs; Claude recalls relevant context automatically | Improves coherence over long sessions; Phase 3+ |
| **Multi-user accounts** | W | Support multiple users in one Bruce instance with separate sessions/configs | Out of scope MVP; auth complexity not worth single-user UX |
| **Cloud sync** | W | Backup sessions/messages to cloud; sync across devices | Out of scope; self-hosted by design |
| **Live connector restart** | W | Restart Discord/WhatsApp from web UI without `docker compose restart` | Out of scope; requires complex state management; document workaround |
| **Tool chaining** | W | Claude calls Tool A, uses result in Tool B call without human in loop | Out of scope Phase 1; requires agentic loop redesign; Phase 3 candidate |

**Rationale Notes**:
- **M (Must Have)**: Tool engine, tool registry, Gmail/Calendar/URL tools, local file I/O, PDF generation, Telegram, confirmation UI. These form the agentic loop and first write workflow.
- **S (Should Have)**: Google Docs, GitHub read, Notion read, Gemini provider, error recovery. High-value but not blocking core thesis (tool execution + reads + basic writes).
- **C (Could Have)**: GitHub/Notion writes, crons, memory, WhatsApp files. Valuable in Phase 2/3 but require safety/design maturity.
- **W (Won't Have)**: Multi-user, cloud, live restart. Explicitly parked; revisit after proving single-user model.

---

## 5. Product Roadmap

### Phase 1: Agent Foundation (Weeks 1–6)

**Milestone**: Core agentic loop proven end-to-end. User can ask Claude questions that trigger tool invocations (read + simple write), approve actions, and receive results in chat.

**Backend Work** (26 sequenced tasks):
1. Extend domain models: add `Tool`, `ToolCall`, `ExecutionResult` entities
2. Design tool registry interface; implement registry with locking + error isolation
3. Implement Claude `tool_use` API integration (parse tool calls from response)
4. Build worker task processor for tool execution (parse, dispatch, error handling)
5. Add execution confirmation table in SQLite (store pending approvals, track status)
6. Implement Gmail API connector (OAuth setup, list/fetch messages)
7. Implement Google Calendar connector (time-range queries, event parsing)
8. Implement URL fetch tool (HTTP GET, boilerplate stripping, error handling)
9. Implement local file I/O tools (read/write with path validation)
10. Implement PDF generation tool (markdown to PDF via library)
11. Implement Telegram connector (receive, dispatch, file upload)
12. Add tool execution confirmation to REST API (`/api/v1/confirmations` POST/GET)
13. Extend web UI: show pending confirmations, approve/deny buttons
14. Integrate confirmation prompt into worker (pause on `confirmed: false`)
15. Build error recovery flow (tool timeout, API error, retry with error context)
16. Test Gmail integration end-to-end (list → fetch → Claude → dispatch)
17. Test Calendar integration (query → parse → response)
18. Test URL tool with various content types (HTML, plain text, JSON)
19. Test PDF generation (text → valid PDF file)
20. Test Telegram connector receive/dispatch
21. Test write approval flow (user denies, action does not execute)
22. Hardening: rate limiting for external API calls (Gmail, Calendar, URL)
23. Hardening: timeout safeguards on all tool executions
24. Hardening: input validation and sanitization (prevent injection, path traversal)
25. Documentation: tool authoring guide for developers
26. E2E tests: agent asks for info, approves PDF generation, receives file

**Frontend Work**:
- **Confirmations tab**: display pending tool calls with action preview; approve/deny buttons; status log of past actions
- **Sessions tab**: enhance to show which session triggered a tool call (for multi-connector debugging)
- **Settings tab**: add tool enable/disable toggles; add API credential inputs (Gmail OAuth, Calendar key)

**Risks to Watch**:
- **Tool hallucination**: Claude outputs invalid tool calls (wrong parameter names, types). Mitigate with clear system prompts and error feedback loop.
- **API rate limits**: Gmail/Calendar can 429 quickly if agent is overactive. Mitigate with per-tool rate limiting + exponential backoff.
- **Approval UX friction**: Users reject many confirmations; workflow becomes too interactive. Mitigate by tracking approval/denial rates and tuning tool thresholds.
- **Confirmation state loss**: If Bruce crashes mid-confirmation, pending approvals are lost. SQLite transaction on INSERT handles this; recovery on restart is documented.

**Launch Criteria**:
- ✓ Claude calls Gmail tool, Bruce fetches messages, Claude summarizes in chat
- ✓ Claude calls Calendar tool, returns events for next 3 days
- ✓ Claude calls URL tool, returns clean text summary
- ✓ User asks for PDF, sees confirmation, clicks approve, file is written and delivered to Discord
- ✓ User clicks deny on a GitHub issue creation, action does not execute
- ✓ Telegram messages flow through worker and receive responses
- ✓ No tool execution crashes the worker; errors are logged and reported

---

### Phase 2: Integration Expansion (Weeks 7–14)

**Milestone**: Tool ecosystem grows to 10+ integrations. GitHub and Notion read ops are live. Multi-provider LLM (Claude + Gemini) working. Write ops (GitHub issues, Notion pages) are approved and tested.

**Backend Work** (estimated 20–25 tasks):
1. Refactor `internal/ai/` to multi-provider abstraction (Claude, Gemini factory)
2. Implement Gemini provider (API, request/response mapping, error handling)
3. Add `llm.provider` config key (global default: "claude", can override per session)
4. Add session schema: `provider_override` column (nullable, lets user choose per-chat)
5. Implement GitHub API connector (OAuth, list branches/PRs/issues)
6. Implement GitHub write tool (create issue/PR with approval required)
7. Implement Google Docs read tool (fetch document content, parse metadata)
8. Implement Notion read tool (query database, return paginated results)
9. Implement Notion write tool (create/update pages with approval)
10. Implement local Git connector (list branches, read commits, parse diffs)
11. Implement Git write tool (commit + push with user approval + signature)
12. Implement local SQLite connector (parameterized queries, read + write)
13. Add Trello read tool (fetch board state, list cards)
14. Error recovery: tool execution timeouts propagate to Claude as error context
15. Telemetry: track which tools are called most (for Phase 3 prioritization)
16. Documentation: update tool registry guide with Gemini setup
17. Documentation: OAuth flow guide for GitHub, Google, Notion
18. Test multi-provider: same message sent to Claude then Gemini, compare quality/cost
19. Test GitHub write: create issue, verify in GH UI, test denial flow
20. Test Notion write: create page, verify in Notion UI
21. Test Git ops: commit message generation, verify signed commit
22. Test SQLite: parameterized queries, verify injection prevention
23. Load testing: run 100 requests with tools; measure latency, error rate
24. Hardening: tool call timeout increased to 30s for remote APIs (was 5s in Phase 1)
25. Hardening: add circuit breaker pattern for flaky remote APIs (Gmail, GitHub)

**Frontend Work**:
- **Settings tab**: add provider selection (Claude vs. Gemini) with cost/speed indicators
- **Sessions tab**: show which provider is active for each session; allow per-session override
- **Confirmations tab**: enhanced preview for complex tools (GitHub PR template, Notion schema)
- **Logs tab**: new "Tools" subtab showing execution history, latency, errors

**Risks to Watch**:
- **OAuth token management**: tokens expire or are revoked. Mitigate with refresh logic and "re-authenticate" prompts in UI.
- **Write op bugs**: accidental GitHub PR merge or Notion page deletion. Mitigate with dry-run mode and mandatory approval logs.
- **Gemini cost surprises**: Gemini pricing differs from Claude; users may hit unexpected bills. Mitigate with per-tool pricing display in UI.
- **Tool call explosion**: Claude calls too many tools per message. Mitigate with per-message tool call limits (e.g., max 3 calls/response).

**Launch Criteria**:
- ✓ User can choose Claude or Gemini in Settings; both work equally for simple queries
- ✓ GitHub read tools return branches/PRs/issues correctly
- ✓ User approves GitHub issue creation, issue appears in GH with correct title/body
- ✓ Notion read returns paginated results; write creates page with correct schema
- ✓ Local SQLite queries run without injection vulnerability (tested with `'; DROP TABLE ...`)
- ✓ Tool call latency is <5s p99 for reads, <10s p99 for writes
- ✓ 0% false positives in tool invocation (Claude never calls tools with invalid params)

---

### Phase 3: Advanced Agentic Behaviors (Weeks 15+)

**Milestone**: Bruce shifts from reactive (user asks → agent acts) to proactive (agent runs tasks on schedule, learns from history, chains tools intelligently).

**Planned Features** (not yet prioritized):
1. **Proactive scheduling**: Schedule recurring Claude queries (e.g., "every morning, summarize my inbox and highlight urgent items")
2. **Persistent memory**: Vector embeddings of past conversations; Claude can recall relevant context automatically
3. **Tool chaining**: Claude calls Tool A, integrates result, calls Tool B without user in loop (for complex workflows like "fetch issue details → check linked PRs → suggest merge strategy")
4. **Write op logging**: Full audit trail of all destructive actions (who approved, when, what changed)
5. **WhatsApp file delivery**: Send PDFs and images via WhatsApp media API
6. **Advanced Git workflows**: Merge PRs, create branches, cherry-pick commits (all with approval)
7. **Cost optimization**: Track per-call costs; suggest cheaper model (Haiku vs. Opus) for simple queries
8. **Custom tools SDK**: Publish a tool authoring framework so users can add private tools
9. **Self-hosted monitoring**: Prometheus metrics + Grafana dashboard for uptime, latency, tool health
10. **API rate limit intelligence**: Claude learns which APIs are rate-limited and avoids them

**Not Planned (Out of Scope Indefinitely)**:
- Multi-user SaaS (single-user local is the design)
- Cloud hosting (self-hosted only)
- Real-time bi-directional sync with external services
- Mobile app (LAN-only web UI)

---

## 6. Technical Considerations

1. **Tool registry isolation**: Each tool runs in its own error boundary. If Gmail API times out, other tools still execute. Prevent cascading failures via `internal/tools/` package with per-tool error handling.

2. **Confirmation state in SQLite**: Pending approvals are written to a `confirmations` table with `id`, `session_id`, `tool_call` (JSON), `status` (pending/approved/denied), `created_at`. On worker resume after crash, stale confirmations (>1 hour old) are auto-denied.

3. **API credential storage**: OAuth tokens and API keys live in `config_entries` SQLite table (not in config.yml). At-rest encryption is out of scope for MVP; rely on local-only access and file permissions.

4. **Multi-provider LLM abstraction**: `internal/ai/provider.go` defines the interface; `internal/ai/factory.go` constructs providers by name; `internal/ai/registry.go` holds instances. Worker never knows which provider is active — routes through registry at call time.

5. **Asynq worker concurrency**: Capped at 2 concurrent tasks (unchanged from Phase 1). Tool execution is CPU-light (I/O-bound HTTP calls) so 2 is sufficient. Scale horizontally by running multiple Bruce instances if needed (state is in SQLite).

6. **Tool call parsing from Claude**: Claude's `tool_use` API returns a block with `type: "tool_use"`, `name`, `input` (JSON). Parse this in the worker; validate tool name against registry; execute immediately or queue for approval.

7. **Error context to Claude**: When a tool fails (timeout, 4xx, 5xx), encode the error as a system message: `"Tool 'gmail' failed: [error details]"`. Include in next request to Claude so it can retry or choose alternative approach.

8. **File path validation**: All file I/O tools must validate paths (no `..`, no absolute paths outside user home). Use a whitelist or `filepath.Abs()` + `filepath.HasPrefix()` checks.

9. **Integration test setup**: Phase 1 E2E tests mock Gmail/Calendar APIs; Phase 2 tests use real APIs with test credentials. Document test credential setup in dev guide.

10. **Scaling decision (future)**: If tool execution becomes CPU-bound (e.g., large PDF generation), offload to a separate worker pool. For now, in-process execution is fine.

---

## 7. Go-to-Market Steps

1. **Beachhead Market**: Technical solo founders (CTOs, research engineers, hackers) building solo projects or small teams. Narrow geography: HN/LLM Discord communities first. Success signal: 5 beta testers running Phase 1 for 1 week without crashing.

2. **First 100 Users**: Organic + community-driven acquisition.
   - Post v2 release notes on HN (Show HN thread with demo).
   - Add Bruce to awesome-llm-tools and awesome-personal-ai lists.
   - Offer free setup call for first 10 testers.
   - Single distribution channel: GitHub releases (binary + docker-compose.yml).

3. **Activation Hook**: User completes setup, configures Gmail, asks Claude a question that triggers the Gmail tool. First successful tool call (info retrieval) cements the "this works" moment. Track: time-to-first-tool-call (target <10 min).

4. **Monetization Moment**: None in Phase 1. MVP is free, open-source. Phase 2+ options: optional hosted version, premium tool pack (e.g., Notion + GitHub + Git bundled), or managed service (we run Bruce, you own the data).

5. **Feedback Loop**: Post-first-tool-call survey: "What tool would you add next?" Aggregate votes; prioritize Phase 2 tools accordingly. Slack community for beta testers to report issues and share workflows.

6. **Growth Unlock**: By Phase 2, viral lever is "tool chaining" — Claude can accomplish multi-step tasks (e.g., "find urgent PRs, fetch diffs, suggest review points") with zero user interaction. Workflow videos showing chained tools drive adoption by proving ROI.

---

## 8. Open Questions & Assumptions

### Validated Assumptions (Architectural Decisions Made)

- ✓ **Tool execution safety via confirmation**: User approval is the gate for destructive actions. No complex permission model needed in v1.
- ✓ **Single-binary deployment**: Go binary with embedded web UI, SQLite on disk, Redis in Docker. No separate services.
- ✓ **Tool registry is centralized**: `internal/tools/` package with a registry that dispatches by name. Extensible for Phase 2+.
- ✓ **Asynq for background work**: Task queue remains the core for handling long-running tool executions and retries.
- ✓ **SQLite for confirmation state**: `confirmations` table stores pending approvals; recovered on restart if not too stale.

### Remaining Questions to Validate in Phase 1

1. **Claude tool call accuracy**: Will Claude correctly invoke tools with valid parameters in real-world use, or will it hallucinate and try to call non-existent tools? How often will it need error feedback to correct?
   - *Mitigation*: Track false-positive tool call rate in Phase 1; if >5%, iterate on system prompt and error messages.

2. **User approval friction**: Will frequent confirmation prompts (for write ops) make the UX feel clunky, or will users accept it as a safety tradeoff?
   - *Mitigation*: A/B test approval flow (modal popup vs. email-style log); measure approval latency and denial rate.

3. **API credential management**: Will users comfortably paste Gmail/GitHub tokens into the Bruce web UI, or will this feel unsafe?
   - *Mitigation*: Phase 1 stores plaintext in SQLite (acceptable for local-only use); Phase 2 adds encryption at rest if users request.

4. **Tool call performance**: Will Gmail/Calendar/GitHub API calls from Bruce be fast enough (<5s) for interactive chat, or will users perceive tool-using as slow compared to manual browsing?
   - *Mitigation*: Log all tool execution latencies; set performance budgets per tool; optimize or cache if necessary.

5. **Connector parity**: Will Telegram integration be simple enough to ship in Phase 1, or does it require QR pairing complexity like WhatsApp?
   - *Mitigation*: Telegram has simpler polling API; should be <1 day to implement. If not, defer to Phase 2.

### Deliberate Decisions to Revisit If Early Signals Contradict

- **No multi-user in MVP**: If 10+ testers request multi-user (shared Bruce instance per team), reconsider in Phase 2. Trade-off: auth complexity vs. team adoption.
- **No tool chaining in Phase 1**: If users report "I had to ask multiple questions to accomplish one task," consider shipping tool chaining sooner. Trade-off: worker complexity vs. UX friction.
- **Approval required for all writes**: If approval latency is >30s and users are impatient, consider whitelist model (auto-approve safe writes like "create a GitHub issue with template"). Trade-off: safety vs. speed.
- **Single provider (Claude) in Phase 1**: If Gemini becomes notably cheaper before Week 6, offer it as early option. Trade-off: validation scope vs. cost.

---

## 9. Success Metrics Deep Dive (for tracking in Phase 1)

- **Tool invocation accuracy**: (valid calls / total calls) — target >95% from week 2 onward
- **Time-to-first-tool-call**: median <10 minutes from setup
- **Approval rate**: (approved / presented confirmations) — track, not enforce (if <70%, UX may be too friction-y)
- **Tool execution latency**: p50, p95, p99 per tool (e.g., Gmail fetch <2s p95)
- **Error rate**: (failed tool executions / total invocations) — target <2%
- **Worker uptime**: (9 hours / 10 hours) = 90% minimum (crashing = failed prototype)
- **Tester retention**: 5 testers running continuously for 6 weeks without stopping = success signal for production-readiness

---

## 10. Appendix: Architectural Context

This PRD is informed by a complete technical specification (8 documents, 9000+ lines) that the software architect has produced. Key decisions from that architecture that inform this PRD:

- **Domain Model**: `Tool`, `ToolCall`, `ExecutionResult` entities now live alongside `Session`, `Message`, `ConfigEntry` in domain layer.
- **REST API**: New endpoints `/api/v1/tools` (list available), `/api/v1/confirmations` (pending approvals), `/api/v1/confirmations/{id}` (approve/deny).
- **Worker Processor**: Extended to parse Claude's `tool_use` response blocks; dispatch tool calls; wait for approval if `confirmed: false`; integrate result back into message stream.
- **Tool Package**: New `internal/tools/` package with registry, per-tool error boundaries, and timeout safeguards.
- **Provider Abstraction**: `internal/ai/` is now multi-provider ready (Claude + Gemini); worker is provider-agnostic.

See `/docs/specs/` for the full technical blueprint (phases 1–3, all 26 backend tasks sequenced, error handling, testing strategy).

---

## 11. PRD Sign-Off

**Product Manager**: Senior PM (Validated with software architect)
**Date**: 2026-03-12
**Confidence Level**: High — Core agentic loop is architecturally validated; Phase 1 scope is locked at 26 backend tasks over 6 weeks; all Must-Haves are designed and sequenced.
**Next Step**: Hand off to go-backend-dev agent for Phase 1 implementation.

