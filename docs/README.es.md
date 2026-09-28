<div class="intro-hero">
  <img src="../assets/bruce-logo.png" alt="Bruce Logo" class="intro-logo" />
  <div class="intro-text">
    <div class="intro-badge">Asistente de IA Personal</div>
    <h1 class="intro-title">Introducción</h1>
    <p class="intro-subtitle">Un asistente de IA personal, autónomo y autohospedado que vive directamente en tus aplicaciones de mensajería con razonamiento multipaso, programación proactiva y más de 15 herramientas del mundo real.</p>
  </div>
</div>

La mayoría de interfaces de chat son simples generadores de texto: haces una pregunta, devuelven texto. Cuando necesitas que una IA realice acciones reales en tu nombre — programar recordatorios, revisar pull requests, consultar bases de datos o crear paneles interactivos —, necesitas un sistema autónomo que razone en bucles de múltiples turnos y utilice herramientas directamente.

Bruce se ejecuta como un único binario Go autónomo con los recursos del panel web integrados y una base de datos SQLite local en modo Write-Ahead Log (WAL). Los mensajes recibidos de tus plataformas se desacoplan mediante una cola de tareas asíncrona en Redis (Asynq), lo que permite a Bruce realizar tareas de larga duración, encadenamiento de herramientas y alertas programadas sin bloquearse ni perder mensajes.

```mermaid
flowchart LR
    user["<b>Tú</b><br/>WhatsApp · Discord · Telegram"]
    
    subgraph core["Núcleo de Bruce"]
        queue["Cola Asíncrona Redis<br/>(Asynq)"]
        loop["Bucle del Agente Autónomo<br/>(RunAgentLoop)"]
        tools["Registro de Herramientas<br/>Búsqueda · Archivos · Git · Google"]
    end
    
    ai["<b>Proveedores de IA</b><br/>Claude · Gemini · OpenAI"]
    
    user -- "mensaje" --> queue
    queue --> loop
    loop <--> ai
    loop <--> tools
    loop -- "respuesta directa o artefacto" --> user
```

## Primitivas Principales

Bruce está diseñado en torno a cinco primitivas esenciales pensadas para la automatización personal y la productividad:

| Primitiva | Propósito | Cómo Funciona |
| --- | --- | --- |
| [**Bucle Autónomo**](architecture/agent-loop.md) | Razonamiento multipaso | Ejecuta herramientas en secuencia dinámicamente hasta resolver instrucciones complejas. |
| [**Programador Proactivo**](features/scheduler-and-proactive.md) | Tareas programadas y alertas | Ejecución en modo dual: recordatorios directos sin tokens (`message`) o informes dinámicos de IA (`agent`). |
| [**Artefactos HTML**](features/artifacts.md) | Documentos web interactivos | Genera paneles, calculadoras e informes HTML independientes servidos en `/artifacts/...`. |
| [**Memoria Híbrida**](architecture/memory-and-context.md) | Continuidad de contexto | Ventana deslizante de mensajes recientes + resumen automático a largo plazo en segundo plano. |
| [**Conectores Omnicanal**](features/connectors.md) | Una sola IA, en todas partes | Conexión simultánea a Discord, WhatsApp, Telegram y Web Chat integrado. |

## Navegación Rápida

<div class="card-grid">
  <a href="guides/installation-docker/" class="card">
    <span class="card-icon">🚀</span>
    <span class="card-title">Inicio Rápido con Docker</span>
    <span class="card-description">Despliega Bruce en menos de 2 minutos con almacenamiento persistente y Redis.</span>
  </a>
  <a href="architecture/overview/" class="card">
    <span class="card-icon">🏛️</span>
    <span class="card-title">Arquitectura del Sistema</span>
    <span class="card-description">Conoce la persistencia SQLite WAL, colas Asynq y ciclos de vida resilientes.</span>
  </a>
  <a href="features/scheduler-and-proactive/" class="card">
    <span class="card-icon">⚡</span>
    <span class="card-title">Programador y Recordatorios</span>
    <span class="card-description">Programa recordatorios directos o informes de IA recurrentes mediante lenguaje natural.</span>
  </a>
  <a href="tools/reference/" class="card">
    <span class="card-icon">🛠️</span>
    <span class="card-title">Referencia de Herramientas</span>
    <span class="card-description">Explora las más de 15 herramientas integradas para búsqueda, bash, git y Google Workspace.</span>
  </a>
</div>

## Código Abierto y Comunidad

Bruce es un proyecto 100% de código abierto publicado bajo la licencia **MIT**.

- 🐙 **Código Fuente**: [Repositorio en GitHub (DemitriusQuadros/bruce)](https://github.com/DemitriusQuadros/bruce)
- 🐛 **Problemas y Sugerencias**: [Abre una incidencia en GitHub](https://github.com/DemitriusQuadros/bruce/issues)
- 💡 **Contribuye**: ¡Las contribuciones con nuevas herramientas, conectores o documentación son bienvenidas!
- ⭐ **Apoya**: Si Bruce te resulta útil, ¡apoya el proyecto con [una estrella en GitHub](https://github.com/DemitriusQuadros/bruce)!
