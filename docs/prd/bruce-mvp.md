# PRD — Bruce MVP

## 1. Product Overview

**One-liner**: Bruce is a self-hosted Go server that lets you control an AI agent through the messaging apps you already use (WhatsApp, Discord, Telegram), designed to run on low-resource machines.

**Problem Statement**: OpenClaw requires 1GB+ RAM, a full Node.js runtime, and a complex install process — making it unusable on constrained hardware like old laptops, WSL2 environments, or home servers. Developers with these setups have no practical way to run a self-hosted AI agent that integrates with their daily communication channels. The current workaround is either paying for cloud AI tools or not having an agent at all.

**Solution**: Bruce is a lightweight Go server deployable via a single `docker-compose up`. It connects to WhatsApp and Discord as input/output channels, routes messages to Claude, and executes responses. The whole stack runs in under 100MB RAM. A simple web UI lets you configure it from any device on the network.

**Primary User**: A solo developer or tech enthusiast with constrained hardware (old laptop, Raspberry Pi, home lab) who wants a personal AI agent accessible through WhatsApp or Discord — without the overhead of Node.js-based tools.

**Stage assumption**: Raw concept → MVP. Scoped for the minimum shippable version that proves the core loop works end-to-end. Monetization and multi-user support are explicitly out of scope for v1.

---

## 2. Goals & Success Metrics

| Goal | Launch Metric | 6-Month Target |
|------|--------------|----------------|
| Core loop works end-to-end | Send a message on WhatsApp, get a Claude response back | <2s median response time |
| Resource efficiency | Runs on WSL2 with ≤128MB RAM total stack | Verified on Raspberry Pi 4 |
| Zero-friction deploy | `docker-compose up` to working agent in <5 min | Community install guide with 0 reported blockers |
| Connector adoption | WhatsApp + 1 more connector live | 3 connectors (WhatsApp, Discord, Telegram) |
| Personal utility | Used daily by the author | 10+ external users running their own instance |

---

## 3. User Stories

### Theme A — Setup & Deployment

> **As a** developer with a low-RAM laptop, **I want to** deploy Bruce with a single `docker-compose up`, **so that** I don't need to install Node.js or manage complex dependencies.

> **As a** user, **I want to** configure my Claude API key and WhatsApp connection through a simple web UI, **so that** I can set everything up from my MacBook without SSH-ing into the server.

### Theme B — Core Agent Loop

> **As a** user, **I want to** send a message to Bruce on WhatsApp, **so that** I get a Claude-powered response directly in my chat.

> **As a** user, **I want to** give Bruce a task (e.g., "summarize my last 5 GitHub PRs"), **so that** it executes the action and replies with the result.

> **As a** user, **I want to** send a message on Discord and get the same Claude response, **so that** I can use whichever platform I'm already on.

### Theme C — Control & Visibility

> **As a** user, **I want to** see a log of all messages and agent actions in the web UI, **so that** I can debug what Bruce did and why.

> **As a** user, **I want to** define a system prompt for Bruce (e.g., "You are my personal dev assistant"), **so that** the agent has my preferred personality and context.

> **As a** user, **I want to** pause/resume Bruce from the web UI, **so that** I can stop it from consuming API tokens when I don't need it.

---

## 4. Feature List with MoSCoW Prioritization

| Feature | Priority | Description | Why |
|---------|----------|-------------|-----|
| WhatsApp connector | M | Receive & send messages via WhatsApp (whatsmeow) | Primary input channel |
| Claude integration | M | Route messages to Claude API, return response | Core value — nothing works without this |
| Docker Compose deploy | M | Single-command startup, all services containerized | Primary success metric |
| Web config UI | M | Set API keys, system prompt, connector toggles from browser | Required to configure from MacBook |
| Message/action log | M | Persistent log of all inputs, outputs, and tool calls | Essential for debugging during MVP |
| Discord connector | S | Receive & send messages via Discord bot | Second connector, official stable API |
| System prompt config | S | Per-connector configurable system prompt | Makes Bruce actually useful day-to-day |
| Pause/resume agent | S | Toggle Bruce on/off without restarting Docker | Prevents runaway API spend |
| Telegram connector | C | Receive & send messages via Telegram bot | Third connector — defer until WhatsApp + Discord proven |
| Tool/skill execution | C | Let Bruce run predefined shell commands or HTTP calls | Powerful but adds attack surface — Phase 2 |
| Token usage dashboard | C | Show Claude API token consumption per session | Useful but not blocking anything |
| ARM/Raspberry Pi builds | C | Pre-built Docker images for ARM64 | Targets the defensible niche |
| Multi-user support | W | Multiple users with separate contexts | Out of scope for v1 |
| Bruce Cloud tier | W | Hosted SaaS version | Valid business direction but irrelevant for MVP |
| Skill marketplace | W | Community-contributed integrations | Phase 3+ only |

---

## 5. Product Roadmap

### Phase 1 — Core (Weeks 1–3)

**Goal**: End-to-end message loop. WhatsApp message → Claude → reply back.

**Features:**
- WhatsApp connector via `go-whatsmeow` (native Go, no Node dependency)
- Claude API integration via Anthropic Go SDK or direct HTTP
- Basic system prompt via `config.yml`
- Docker Compose with Redis + Bruce server
- Message log stored in SQLite

**Key milestone**: Send "What's 2+2?" on WhatsApp from MacBook → Bruce replies "4" from Lenovo running WSL2.

**Risks**: `whatsmeow` is reverse-engineered and can break on WhatsApp updates. Have Discord as fallback connector if WhatsApp integration blocks Phase 1.

---

### Phase 2 — Usable (Weeks 4–7)

**Goal**: Daily driver. Multiple connectors, configurable, observable.

**Features:**
- Web config UI served by existing Gorilla Mux API
- Discord connector (official bot API — stable and well-documented)
- System prompt configurable per connector via web UI
- Pause/resume toggle
- Token usage tracking
- ARM64 Docker image builds via GitHub Actions

**Key milestone**: Bruce is running daily on the Lenovo, configured from MacBook, with at least one external user from r/selfhosted or r/homelab.

**Risks**: Scope creep — resist adding tool execution until Phase 3. Focus on the message loop being rock-solid first.

---

### Phase 3 — Powerful (Weeks 8+)

**Goal**: Bruce does things, not just responds.

**Features:**
- Tool/skill execution (HTTP calls, shell commands behind explicit allowlist)
- Telegram connector
- Token usage dashboard in web UI
- Community install guide + GitHub README improvements
- Scheduled tasks via Asynq (already in go-base-project)

**Key milestone**: Ask Bruce on WhatsApp to "check my GitHub notifications" → replies with a summary.

**Risks**: Tool execution is a security surface — implement strict allowlist from day one, never open-ended shell exec.

---

## 6. Technical Considerations

- **WhatsApp library**: Use [`go-whatsmeow`](https://github.com/tulir/whatsmeow) — the most maintained Go WhatsApp library. Stores session state in SQLite (already a dep in `go.mod`).
- **Claude integration**: Use Anthropic's Go SDK or direct HTTP calls to `api.anthropic.com`. Store API key in `config.yml` via existing Viper configuration.
- **Core entities**: `Message{ID, Source, Content, Role, Timestamp}`, `Session{ConnectorType, ChannelID, SystemPrompt, Active}`, `ToolCall{ID, Name, Input, Output}`.
- **Web UI**: Gorilla Mux already wired — serve a minimal static `index.html` with `fetch()` calls. No React, no build step.
- **Build vs buy**: WhatsApp (`whatsmeow`) and Discord (`discordgo`) are build. Everything else (Docker, Redis, SQLite, Claude API) is buy. Don't write a custom queue — Asynq is already wired.
- **Persistence**: SQLite for message logs and session state (zero infra, works in Docker volume). Redis for Asynq task queue and connector state caching.
- **ARM builds**: Add GitHub Actions multi-platform build (`linux/amd64`, `linux/arm64`) from day one — 5-line Dockerfile addition that opens the Raspberry Pi market immediately.

---

## 7. Go-to-Market Steps

1. **Beachhead**: Solo developers running WSL2 or home lab setups who have abandoned OpenClaw/OpenHands due to resource limits. Active members of r/selfhosted, r/homelab, and r/brasil_dev.

2. **First 10 users**: Post a "Show HN" and a r/selfhosted thread once Phase 1 is running. Lead with the benchmark: *"Bruce runs in 45MB RAM vs OpenClaw's 1GB — here's how I built a WhatsApp AI agent in Go."*

3. **Activation hook**: The moment a user sends their first WhatsApp message and gets a Claude response back. Everything in Phase 1 is designed to make this happen in under 5 minutes from `git clone`.

4. **Monetization moment**: Not in MVP. If pursued, the trigger is: user has run Bruce for 30 days → offer Bruce Cloud (managed hosting) at $9/month. Validate demand with a waitlist first.

5. **Feedback loop**: Add a `/feedback <message>` command in Bruce itself — users send feedback from WhatsApp directly, logged to a Notion database. Zero friction, fits the tool's own UX.

6. **Growth unlock**: ARM64 support. If Bruce is the only Go-based AI agent with a working Raspberry Pi Docker image, it owns that community. r/homelab + r/RASPBERRY_PI_PROJECTS have millions of members and love "runs on Pi" content.

---

## 8. Open Questions & Assumptions

### Assumptions to validate in the next 30 days

1. **Does `go-whatsmeow` work reliably in WSL2?** Test session persistence, reconnection after restart, and media handling before committing. If it breaks, Discord is the fallback.

2. **WhatsApp or Discord first?** Discord has an official API and is more stable. Consider building Discord first and WhatsApp second if whatsmeow proves unreliable in Phase 1.

3. **How much of OpenClaw's complexity is actually needed?** OpenClaw has 50+ integrations. Bruce needs 3. Resist the pull to replicate its feature set — the value prop is being smaller and simpler.

4. **Who else hits this wall?** Before Phase 2, spend 2 hours in r/selfhosted searching "OpenClaw memory" or "OpenHands RAM". 20+ threads = community. 2 threads = personal tool (still valid).

### Deliberate decisions (can be revisited)

- Assumed **personal use only for MVP** — revisit if early users want multi-account or team features
- Assumed **WhatsApp first** — revisit after Phase 1 if whatsmeow proves unreliable
- Assumed **web UI served by Go** — revisit if config complexity grows beyond what static HTML handles
- Assumed **no tool execution in Phase 1** — revisit only after the message loop runs stably for 2+ weeks
