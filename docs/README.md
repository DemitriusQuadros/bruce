<div class="bruce-hero">
  <img src="assets/bruce-logo.png" alt="Bruce Logo" />
  <h1>Bruce Documentation</h1>
  <p>A self-hosted, autonomous personal AI assistant that lives in your messaging apps with multi-step reasoning, proactive scheduling, and 12+ real-world tools.</p>
</div>

---

## 🏛️ System Architecture

- [System Architecture Overview](architecture/overview.md) — Runtime components, Asynq task queues, SQLite WAL persistence, and lifecycle.
- [Autonomous Agent Loop (`RunAgentLoop`)](architecture/agent-loop.md) — Multi-step tool execution, reasoning cycles, and anti-hallucination guardrails.
- [Memory, Context & Summarization](architecture/memory-and-context.md) — Real-time temporal clock injection, sliding context window, and background long-term session summarization.

---

## 🚀 Installation & Deployment

- [Docker Compose Installation Guide](guides/installation-docker.md) — Production setup with persistent volumes, Redis, and health checks.
- [Bare-Metal Local Development](guides/installation-local.md) — Compiling from source (`CGO_ENABLED=1`), local Redis, and running tests.
- [Homelab & Production Deployment](guides/deployment-homelab.md) — Caddy/Nginx reverse proxy, automatic TLS, Tailscale private mesh, and systemd services.

---

## ⚡ Core Features

- [Scheduler & Proactive Tasks](features/scheduler-and-proactive.md) — Dual execution modes (**`message`** for direct delivery vs **`agent`** for briefings), relative offsets (`+2m`, `daqui a 2 minutos`), ambient condition watches, and poller mechanics.
- [Standalone HTML Artifacts Engine](features/artifacts.md) — Generating, hosting (`/artifacts/...`), and previewing rich HTML documents, dashboards, and calculators.
- [Multi-Provider LLM Engine](features/llm-providers.md) — Claude, Gemini, and OpenAI integration with dynamic runtime provider switching.
- [Omnichannel Connectors Guide](features/connectors.md) — Complete setup guides for Discord, WhatsApp (`whatsmeow`), Telegram, and Web Chat.
- [Web Dashboard & REST API Reference](features/dashboard-and-api.md) — Web UI navigation, Asynqmon task queue monitor, and REST API endpoints.

---

## 🛠️ Tools Reference

- [Built-In Tools Reference](tools/reference.md) — Full parameter schemas, configuration requirements, and usage examples for all 15+ built-in tools.
