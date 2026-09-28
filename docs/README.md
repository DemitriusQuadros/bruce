<div class="intro-hero">
  <img src="assets/bruce-logo.png" alt="Bruce Logo" class="intro-logo" />
  <div class="intro-text">
    <div class="intro-badge">Personal AI Assistant</div>
    <h1 class="intro-title">Introduction</h1>
    <p class="intro-subtitle">A self-hosted, autonomous personal AI assistant that lives directly in your messaging apps with multi-step reasoning, proactive scheduling, and 12+ real-world tools.</p>
  </div>
</div>

Most AI chat interfaces are simple text generators: you ask a question, they return text. When you need an AI to take real-world action on your behalf—scheduling reminders, checking pull requests, querying databases, or building interactive dashboards—you need an autonomous system that reasons in multi-step loops and interfaces directly with tools.

Bruce runs as a single, self-contained Go binary with embedded web dashboard assets and a local SQLite database in Write-Ahead Log (WAL) mode. Incoming messages from your messaging platforms are decoupled via an asynchronous Redis task queue (Asynq), allowing Bruce to handle long-running research, tool chaining, and scheduled tasks without dropping messages or blocking.

```mermaid
flowchart LR
    user["<b>You</b><br/>WhatsApp · Discord · Telegram"]
    
    subgraph core["Bruce Core Engine"]
        queue["Async Redis Queue<br/>(Asynq)"]
        loop["Autonomous Agent Loop<br/>(RunAgentLoop)"]
        tools["Tool Registry<br/>Search · Files · Git · Google"]
    end
    
    ai["<b>AI Providers</b><br/>Claude · Gemini · OpenAI"]
    
    user -- "message" --> queue
    queue --> loop
    loop <--> ai
    loop <--> tools
    loop -- "direct reply or artifact" --> user
```

## Core Primitives

Bruce is built around five core primitives designed for personal automation and daily productivity:

| Primitive | Purpose | How It Works |
| --- | --- | --- |
| [**Autonomous Loop**](architecture/agent-loop.md) | Multi-step reasoning | Dynamically executes tools in sequence until complex instructions are solved. |
| [**Proactive Scheduler**](features/scheduler-and-proactive.md) | Timed tasks & alerts | Dual-mode execution: direct zero-token reminders (`message`) or dynamic AI briefings (`agent`). |
| [**HTML Artifacts**](features/artifacts.md) | Interactive web documents | Generates standalone HTML dashboards, calculators, and reports served via `/artifacts/...`. |
| [**Hybrid Memory**](architecture/memory-and-context.md) | Never forgets context | Sliding window of recent turns + automatic background long-term summarization. |
| [**Omnichannel Connectors**](features/connectors.md) | Single AI, everywhere | Connects simultaneously to Discord, WhatsApp, Telegram, and embedded Web Chat. |

## Quick Navigation

<div class="card-grid">
  <a href="guides/installation-docker/" class="card">
    <span class="card-icon">🚀</span>
    <span class="card-title">Docker Compose Quickstart</span>
    <span class="card-description">Deploy Bruce in under 2 minutes with persistent storage and Redis.</span>
  </a>
  <a href="architecture/overview/" class="card">
    <span class="card-icon">🏛️</span>
    <span class="card-title">System Architecture</span>
    <span class="card-description">Learn about SQLite WAL persistence, Asynq queues, and graceful lifecycles.</span>
  </a>
  <a href="features/scheduler-and-proactive/" class="card">
    <span class="card-icon">⚡</span>
    <span class="card-title">Scheduler & Reminders</span>
    <span class="card-description">Schedule direct reminders or recurrent AI briefings via natural language.</span>
  </a>
  <a href="tools/reference/" class="card">
    <span class="card-icon">🛠️</span>
    <span class="card-title">Built-In Tools Reference</span>
    <span class="card-description">Explore the 15+ built-in tools for search, bash, git, and Google Workspace.</span>
  </a>
</div>
