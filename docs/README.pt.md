<div class="bruce-hero">
  <img src="assets/bruce-logo.png" alt="Bruce Logo" />
  <h1>Documentação do Bruce</h1>
  <p>Um assistente de IA pessoal autônomo e auto-hospedado que vive nos seus apps de mensagens, com raciocínio multi-etapas, agendamento proativo e mais de 12 ferramentas do mundo real.</p>
</div>

---

## 🏛️ Arquitetura do Sistema

- [Visão Geral da Arquitetura](architecture/overview.md) — Componentes de execução, filas de tarefas Asynq, persistência SQLite WAL e ciclo de vida.
- [Loop do Agente Autônomo (`RunAgentLoop`)](architecture/agent-loop.md) — Execução de ferramentas em multi-etapas, ciclos de raciocínio e salvaguardas anti-alucinação.
- [Memória, Contexto & Sumarização](architecture/memory-and-context.md) — Injeção de relógio temporal em tempo real, janela deslizante de contexto e sumarização em background de sessões de longo prazo.

---

## 🚀 Instalação & Deploy

- [Guia de Instalação com Docker Compose](guides/installation-docker.md) — Configuração para produção com volumes persistentes, Redis e verificações de integridade (health checks).
- [Desenvolvimento Local Bare-Metal](guides/installation-local.md) — Compilação a partir do código fonte (`CGO_ENABLED=1`), Redis local e execução de testes.
- [Deploy em Homelab & Produção](guides/deployment-homelab.md) — Proxy reverso Caddy/Nginx, TLS automático, malha privada Tailscale e serviços systemd.

---

## ⚡ Recursos Principais

- [Agendador & Tarefas Proativas](features/scheduler-and-proactive.md) — Modos duplos de execução (**`message`** para entrega direta vs **`agent`** para briefings gerados pela IA), offsets relativos (`+2m`, `daqui a 2 minutos`), monitoramento de condições ambientais e mecanismo do poller.
- [Motor de Artefatos HTML Autônomos](features/artifacts.md) — Geração, hospedagem (`/artifacts/...`) e visualização de documentos HTML ricos, dashboards e calculadoras.
- [Motor LLM Multi-Provedor](features/llm-providers.md) — Integração com Claude, Gemini e OpenAI com alternância dinâmica de provedores em tempo de execução.
- [Guia de Conectores Omnichannel](features/connectors.md) — Guias completos de configuração para Discord, WhatsApp (`whatsmeow`), Telegram e Web Chat.
- [Painel Web & Referência da API REST](features/dashboard-and-api.md) — Navegação na interface Web, monitor de filas de tarefas Asynqmon e endpoints da API REST.

---

## 🛠️ Referência de Ferramentas

- [Referência de Ferramentas Nativas](tools/reference.md) — Esquemas de parâmetros completos, requisitos de configuração e exemplos de uso para mais de 15 ferramentas integradas.
