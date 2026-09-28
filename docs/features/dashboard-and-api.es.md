# Panel Web y Referencia de la API REST

Bruce ofrece una interfaz web de gestión intuitiva y una API REST completa para administración, auditoría e interacción directa.

---

## 1. Visión General del Panel Web

El panel está disponible en `http://localhost:8080` (o `http://<ip-del-servidor>:9090` con Docker Compose).

- **Recursos Integrados**: Todo el frontend (HTML, CSS, JS, iconos) se compila dentro del binario de Go con el paquete `embed`. No se requiere servidor web externo.
- **JavaScript Vanilla**: Ligero, sin frameworks pesados, con carga prácticamente instantánea.

---

## 2. Pestañas del Panel

### 1. Chat
Interfaz conversacional integrada:
- Conversa con Bruce directamente desde tu navegador web.
- Crea múltiples hilos de conversación independientes.
- Soporte para markdown completo, listas, tablas y resaltado de sintaxis en bloques de código.

### 2. Sessions (Sesiones)
Inspecciona todas las conversaciones activas en cualquier conector:
- Filtra sesiones por plataforma (**Discord**, **WhatsApp**, **Telegram**, **Web**).
- Revisa el historial de mensajes con marcas de tiempo.
- Comprueba si existe un resumen de contexto activo para cada sesión.
- Elimina sesiones (borra en cascada los mensajes y tareas asociadas).

### 3. Schedules (Programaciones)
Administra tareas proactivas y monitoreos en segundo plano:
- **Distintivos Visuales**: Diferenciación entre modos `Cron`, `Watch` y `Message` directo.
- **Cuenta Regresiva en Tiempo Real**: Visualiza el tiempo exacto hasta la próxima ejecución.
- **Disparo Manual**: Botón **Run Now** para encolar una prueba de forma inmediata.
- **Controles**: Pausar, reanudar, editar o eliminar cualquier tarea.
- **Editor Modal**: Crea tareas con selección intuitiva de zona horaria y modo de ejecución.

### 4. Artifacts (Artefactos)
Galería de todos los documentos HTML generados por Bruce:
- Tarjetas con títulos, fechas y tamaños de archivo.
- **Vista Previa Interactiva**: Interactúa con el artefacto directamente en un iframe aislado.
- **Modos Adaptables**: Comprueba el diseño en formatos Escritorio, Tableta y Móvil.
- Copia el enlace compartible o descarga el archivo HTML.

### 5. Logs (Registros)
Registro de auditoría en tiempo real:
- Filtra por ejecuciones de herramientas, envíos a conectores o errores.
- Inspecciona las entradas y salidas JSON de cada herramienta invocada.

### 6. Settings (Ajustes)
Editor dinámico de configuración en caliente:
- Cambia `ui.default_system_prompt` sin editar ficheros de configuración.
- Alterna `llm.provider` entre `claude`, `gemini` y `openai`.
- Actualiza tokens de API y credenciales con campos protegidos.
- Configura la zona horaria de la aplicación (`app.timezone`).

---

## 3. Monitor de Colas Asynq (`/monitor`)

Bruce integra [Asynqmon](https://github.com/hibiken/asynqmon) en `http://localhost:8080/monitor`:
- Visibilidad en tiempo real de las colas de Redis (`default`, `low`, etc.).
- Inspección de tareas activas, en espera, programadas, en reintento y archivadas.
- Cancelación manual, eliminación o reintento de tareas fallidas.

---

## 4. Referencia de Endpoints de la API REST

| Método | Endpoint | Descripción |
|---|---|---|
| `GET` | `/health` | Estado de salud y tiempo en activo en segundos. |
| `GET` | `/api/v1/sessions` | Lista las sesiones con metadatos de canal. |
| `GET` | `/api/v1/sessions/{id}` | Obtiene una sesión concreta por ID. |
| `DELETE` | `/api/v1/sessions/{id}` | Elimina una sesión y sus mensajes asociados. |
| `GET` | `/api/v1/chat/sessions` | Lista sesiones creadas en el Web Chat. |
| `POST` | `/api/v1/chat/sessions` | Crea una nueva sesión en el Web Chat. |
| `POST` | `/api/v1/chat` | Envía un mensaje a Bruce mediante la API de chat. |
| `GET` | `/api/v1/proactive-tasks` | Lista las tareas programadas y monitoreos. |
| `POST` | `/api/v1/proactive-tasks` | Crea una nueva tarea programada (cron o watch). |
| `GET` | `/api/v1/proactive-tasks/{id}` | Consulta los detalles de una tarea programada. |
| `PATCH` | `/api/v1/proactive-tasks/{id}` | Actualiza programación, título, prompt o estado. |
| `POST` | `/api/v1/proactive-tasks/{id}/run` | Ejecuta inmediatamente una tarea programada. |
| `DELETE` | `/api/v1/proactive-tasks/{id}` | Elimina una tarea programada. |
| `GET` | `/api/v1/artifacts` | Lista los artefactos HTML guardados. |
| `GET` | `/artifacts/{id}/{filename}` | Sirve el archivo HTML del artefacto. |
| `DELETE` | `/api/v1/artifacts/{id}` | Elimina el artefacto del disco y del registro. |
| `GET` | `/api/v1/config` | Obtiene la configuración activa no confidencial. |
| `PUT` | `/api/v1/config` | Actualiza un ajuste dinámicamente. |
| `GET` | `/api/v1/connectors` | Comprueba el estado de Discord, WhatsApp y Telegram. |
| `GET` | `/swagger/` | Documentación interactiva Swagger / OpenAPI. |
