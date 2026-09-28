# Programador y Tareas Proactivas

Bruce cuenta con un programador autónomo en segundo plano que permite tanto **notificaciones futuras directas** como **informes programados con agentes de IA** en múltiples canales de mensajería.

---

## 1. Visión General de la Arquitectura

El programador se compone de tres sistemas interactivos:
1. **Almacenamiento SQLite (tabla `proactive_tasks`)**: Almacena programaciones, condiciones, canales de destino y estado.
2. **Sondeador en Segundo Plano (`internal/scheduler/poller.go`)**: Se ejecuta cada minuto, busca tareas pendientes (`next_run_at <= now AND is_active = 1`), recalcula los próximos horarios de ejecución y encola los trabajos en Redis.
3. **Motor Worker Asynq (`internal/worker/proactive_handlers.go`)**: Desencola y ejecuta las tareas de acuerdo con su **Modo de Ejecución** configurado.

---

## 2. Modos Duales de Ejecución

Bruce distingue explícitamente entre **Entrega Directa de Mensajes** e **Informes de Agentes de IA**:

```
┌────────────────────────────────────────────────────────┐
│               Tarea Programada (Cron)                  │
└──────────────────────────┬─────────────────────────────┘
                           │
             ┌─────────────┴─────────────┐
             ▼                           ▼
    execution_mode: "message"   execution_mode: "agent"
     (Notificación Directa)      (Tarea Dinámica de IA)
             │                           │
     Omite el LLM                Ejecuta Bucle del Agente
     0 tokens, 0 latencia        Ejecuta herramientas y compila informe
```

### Modo 1: Entrega Directa de Mensajes (`execution_mode: "message"`)
Diseñado para **recordatorios, alarmas y notificaciones programadas específicas**.
- **Omite el LLM por completo** en el momento del disparo.
- **Cero latencia, cero costo de tokens**, completamente inmune a límites de frecuencia o agotamiento de cuotas de API.
- Extrae y limpia el texto del mensaje directamente desde `prompt_condition` (eliminando comillas o comandos repetitivos como `"Envía el mensaje: ..."`).
- Envía el mensaje directamente a Discord, WhatsApp o Telegram.

**Ejemplos Típicos**:
- *"Envíame un hola mundo en dos minutos"*
- *"Recuérdame tomar la medicina a las 20:00"*
- *"Envíame 'Llamar al dentista' a las 14:30"*

---

### Modo 2: Modo de Informe del Agente (`execution_mode: "agent"`)
Diseñado para **investigación dinámica, verificaciones ambientales y resúmenes basados en herramientas**.
- A la hora programada, el worker invoca el **bucle autónomo del agente** (`RunAgentLoop`).
- Bruce puede navegar por la web, verificar pull requests en GitHub, leer archivos o consultar bases de datos.
- Los resultados se sintetizan en un informe estructurado y se envían a tu canal.

**Ejemplos Típicos**:
- *"Todos los días laborables a las 9:00, busca noticias sobre IA y envíame un resumen de 3 puntos."*
- *"Revisa las incidencias de GitHub cada mañana y alértame si hay nuevos errores críticos."*

---

## 3. Sintaxis de Programación Soportada

### A. Desfases Relativos (Tareas Únicas)
Puedes programar tareas utilizando expresiones naturales de tiempo relativo:
- `+2m`, `2m`, `in 2 minutes`, `daqui a 2 minutos`, `en 2 minutos`
- `+30m`, `in 1 hour`, `en 3 horas`

**Cómo funciona internamente**:
1. Cuando `proactive_create` recibe un desfase relativo (ej: `+2m`), calcula la marca de tiempo exacta de destino en la zona horaria configurada.
2. Convierte el desfase en una expresión cron concreta de 5 campos con fecha específica:
   `minuto hora día mes *` (ej: `15 14 28 9 *`).
3. Cuando el sondeador evalúa esta tarea después de la ejecución, detecta que la siguiente ejecución está a más de 30 días en el futuro (el próximo año) y marca automáticamente `is_active = 0` (desactivándola para que nunca se repita).

### B. Expresiones Cron Recurrentes de 5 Campos
Sintaxis cron estándar compatible con `robfig/cron/v3`:
```text
┌───────────── minuto (0 - 59)
│ ┌─────────── hora (0 - 23)
│ │ ┌───────── día del mes (1 - 31)
│ │ │ ┌─────── mes (1 - 12)
│ │ │ │ ┌───── día de la semana (0 - 6, Domingo a Sábado)
│ │ │ │ │
* * * * *
```
- `0 9 * * 1-5`: Cada día laborable a las 9:00 AM.
- `*/15 * * * *`: Cada 15 minutos.
- `0 18 * * 5`: Todos los viernes a las 18:00.

### C. Monitoreo de Condiciones Ambientales (`task_type: "watch"`)
Los watches se ejecutan periódicamente para monitorear condiciones en lugar de entregar un mensaje en horario fijo:
- **Intervalo**: Especificado en minutos enteros (ej: `30` o `60`). Aplica una salvaguarda de **mínimo 5 minutos estricto** para evitar límites de API.
- **Condición**: Descripción en lenguaje natural de lo que se debe monitorear (ej: *"correos no leídos de contratistas"*).
- **Herramientas Específicas**: Especifica `target_tools` (ej: `["gmail_search"]`) para limitar la ejecución de herramientas concretas.
- **Hash de Desduplicación**: Bruce genera un hash del resultado de la evaluación (`last_result_hash`). Si la condición no ha cambiado desde la última verificación, no se envía ninguna alerta, evitando spam innecesario.

---

## 4. Enrutamiento Entre Canales (Cross-Channel)

Las programaciones no están bloqueadas al canal donde se crearon:
- Puedes conversar con Bruce en **Discord** y programar una alerta para ser entregada a tu número de **WhatsApp**:
  ```json
  {
    "target_connector": "whatsapp",
    "target_channel_id": "5511999999999"
  }
  ```
- O programar un informe desde el **Panel Web** para ser entregado a un canal privado de DM en **Discord**.

---

## 5. Herramientas de Gestión Conversacional

Puedes gestionar todas las tareas mediante lenguaje natural en cualquier conector de chat:

| Herramienta | Propósito | Ejemplo de Prompt Conversacional |
|---|---|---|
| `proactive_create` | Crea una programación o watch | *"Recuérdame comprar café en 10 minutos"* |
| `proactive_list` | Lista todas tus tareas programadas | *"¿Cuáles son mis tareas activas?"* |
| `proactive_toggle` | Pausa o reanuda una tarea | *"Pausa mi informe diario de noticias"* |
| `proactive_delete` | Elimina una tarea programada | *"Elimina el recordatorio de café"* |

---

## 6. API REST y Panel Web

También puedes gestionar las tareas mediante programación:
- `GET /api/v1/proactive-tasks` — Lista todas las tareas.
- `POST /api/v1/proactive-tasks` — Crea una nueva tarea proactiva.
- `PATCH /api/v1/proactive-tasks/{id}` — Actualiza programación, modo de ejecución o estado activo.
- `POST /api/v1/proactive-tasks/{id}/run` — Dispara manualmente una ejecución de prueba inmediata.
- `DELETE /api/v1/proactive-tasks/{id}` — Elimina una tarea.

La pestaña **Schedules** en el Panel Web (`http://localhost:8080`) ofrece controles visuales, temporizadores de cuenta regresiva, distintivos de modo de ejecución y un modal de edición interactivo.
