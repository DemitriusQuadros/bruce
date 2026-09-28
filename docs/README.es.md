<div class="bruce-hero">
  <img src="../assets/bruce-logo.png" alt="Bruce Logo" />
  <h1>Documentación de Bruce</h1>
  <p>Un asistente de IA personal, autónomo y autohospedado que vive en tus aplicaciones de mensajería, con razonamiento multipaso, programación proactiva y más de 12 herramientas del mundo real.</p>
</div>

---

## 🏛️ Arquitectura del Sistema

- [Visión General de la Arquitectura](architecture/overview.md) — Componentes en tiempo de ejecución, colas de tareas Asynq, persistencia SQLite WAL y ciclo de vida.
- [Bucle del Agente Autónomo (`RunAgentLoop`)](architecture/agent-loop.md) — Ejecución de herramientas en múltiples pasos, ciclos de razonamiento y salvaguardas contra alucinaciones.
- [Memoria, Contexto y Resumen](architecture/memory-and-context.md) — Inyección de reloj temporal en tiempo real, ventana de contexto deslizante y resumen en segundo plano de sesiones a largo plazo.

---

## 🚀 Instalación y Despliegue

- [Guía de Instalación con Docker Compose](guides/installation-docker.md) — Configuración para producción con volúmenes persistentes, Redis y comprobaciones de estado.
- [Desarrollo Local Bare-Metal](guides/installation-local.md) — Compilación desde el código fuente (`CGO_ENABLED=1`), Redis local y ejecución de pruebas.
- [Despliegue en Homelab y Producción](guides/deployment-homelab.md) — Proxy inverso Caddy/Nginx, TLS automático, red privada Tailscale y servicios systemd.

---

## ⚡ Características Principales

- [Programador y Tareas Proactivas](features/scheduler-and-proactive.md) — Modos de ejecución duales (**`message`** para entrega directa vs **`agent`** para resúmenes e informes autónomos), desfases relativos (`+2m`, `daqui a 2 minutos`), monitoreo de condiciones ambientales y mecánica del poller.
- [Motor de Artefactos HTML Autónomos](features/artifacts.md) — Generación, alojamiento (`/artifacts/...`) y previsualización de documentos HTML interactivos, paneles y calculadoras.
- [Motor LLM Multiproveedor](features/llm-providers.md) — Integración con Claude, Gemini y OpenAI con cambio dinámico de proveedor en caliente.
- [Guía de Conectores Omnicanal](features/connectors.md) — Guías completas de configuración para Discord, WhatsApp (`whatsmeow`), Telegram y Web Chat.
- [Panel Web y Referencia de la API REST](features/dashboard-and-api.md) — Navegación en la interfaz de usuario, monitor de tareas Asynqmon y endpoints de la API REST.

---

## 🛠️ Referencia de Herramientas

- [Referencia de Herramientas Integradas](tools/reference.md) — Esquemas de parámetros completos, requisitos de configuración y ejemplos de uso para más de 15 herramientas nativas.
