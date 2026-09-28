# Referencia de Herramientas Integradas

Este documento ofrece una referencia completa de todas las herramientas disponibles en el bucle del agente autónomo de Bruce.

---

## Categorías de Herramientas

- [1. Esenciales y Productividad](#1-esenciales-y-productividad)
  - [`artifact_save`](#artifact_save)
  - [`web_search`](#web_search)
  - [`http_request`](#http_request)
- [2. Proactividad y Programación](#2-proactividad-y-programacion)
  - [`proactive_create`](#proactive_create)
  - [`proactive_list`](#proactive_list)
  - [`proactive_toggle`](#proactive_toggle)
  - [`proactive_delete`](#proactive_delete)
- [3. Sistema y Archivos](#3-sistema-y-archivos)
  - [`bash_exec`](#bash_exec)
  - [`file_read`, `file_write`, `file_list`](#herramientas-de-sistema-de-archivos)
  - [`git_local`](#git_local)
- [4. Integraciones y APIs Externas](#4-integraciones-y-apis-externas)
  - [Google Workspace (`gmail`, `calendar`, `docs`)](#herramientas-de-google-workspace)
  - [`github`](#github)
  - [`notion`](#notion)
  - [`trello`](#trello)
  - [`n8n`](#n8n)

---

## 1. Esenciales y Productividad

### `artifact_save`
Genera y guarda un documento HTML independiente almacenado en disco y servido vía HTTP.

- **Configuración**: `tools.artifacts.enabled: true`
- **Parámetros**:
  - `title` (*string, obligatorio*): Título visible del artefacto.
  - `filename` (*string, obligatorio*): Nombre del archivo terminado en `.html` (ej: `dashboard.html`).
  - `content` (*string, obligatorio*): Documento HTML autónomo completo incluyendo `<!DOCTYPE html>`, `<html>`, `<head>`, `<style>` y `<body>`.
- **Salida**: Devuelve la ruta de acceso pública `/artifacts/<id>/<filename>`.

---

### `web_search`
Consulta la API de Google Custom Search para obtener información en tiempo real y citas web.

- **Configuración**:
  ```yaml
  tools:
    web_search:
      enabled: true
      google_api_key: "AIzaSy..."
      google_cx: "0123456789:abcdef"
  ```
- **Parámetros**:
  - `query` (*string, obligatorio*): Términos de búsqueda.
  - `num_results` (*entero, opcional*): Cantidad de resultados a devolver (1-10, predeterminado: 5).
- **Salida**: Lista formateada de títulos, fragmentos y URLs fuente.

---

### `http_request`
Realiza peticiones HTTP arbitrarias (GET, POST, PUT, DELETE) hacia APIs REST externas o webhooks.

- **Configuración**: Habilitado por defecto (`tools.http_client.enabled: true`).
- **Parámetros**:
  - `url` (*string, obligatorio*): URL completa de destino HTTP/HTTPS.
  - `method` (*string, obligatorio*): Método HTTP (`GET`, `POST`, `PUT`, `DELETE`).
  - `headers` (*objeto, opcional*): Mapa clave-valor de encabezados HTTP.
  - `body` (*string, opcional*): Contenido del cuerpo de la solicitud.
- **Salida**: Código de estado HTTP y texto de respuesta.

---

## 2. Proactividad y Programación

### `proactive_create`
Registra una tarea programada en segundo plano o un monitoreo ambiental.

- **Configuración**: Nativo en el núcleo (requiere el sondeador de tareas).
- **Parámetros**:
  - `title` (*string, obligatorio*): Nombre descriptivo de la tarea o watch.
  - `type` (*string, obligatorio*): `'cron'` para tareas periódicas o `'watch'` para monitorización de condiciones.
  - `schedule` (*string, obligatorio*):
    - Para `cron`: Cron estándar de 5 campos (`0 9 * * 1-5`) o desfase relativo (`+2m`, `in 5 minutes`, `daqui a 2 minutos`, `en 2 minutos`).
    - Para `watch`: Intervalo de sondeo en minutos (cadena entera, ej: `'30'`, mínimo 5).
  - `execution_mode` (*string, opcional*):
    - `'message'`: Notificación/recordatorio directo (omite el LLM, 0 tokens, 0 latencia).
    - `'agent'`: Se ejecuta mediante el bucle del agente con soporte de herramientas.
  - `prompt_condition` (*string, obligatorio*):
    - Para `execution_mode: 'message'`: El texto exacto a enviar.
    - Para `execution_mode: 'agent'`: Instrucciones para el agente de IA.
    - Para `watch`: Condición a vigilar (ej: `'correos no leídos de clientes'`).
  - `target_connector` (*string, opcional*): `'discord'`, `'whatsapp'`, `'telegram'` o `'web'`.
  - `target_channel_id` (*string, opcional*): ID del canal de destino o número de teléfono.
  - `target_tools` (*array de strings, opcional*): Lista de herramientas autorizadas para la condición.
  - `timezone` (*string, opcional*): Zona horaria IANA (por defecto la del sistema).

---

### `proactive_list`
Lista todas las tareas proactivas y monitoreos de la sesión actual o globales.

---

### `proactive_toggle`
Pausa o reanuda una tarea proactiva.

- **Parámetros**:
  - `task_id` (*string, obligatorio*): UUID de la tarea.
  - `is_active` (*booleano, obligatorio*): `true` para reanudar, `false` para pausar.

---

### `proactive_delete`
Elimina definitivamente una tarea proactiva.

- **Parámetros**:
  - `task_id` (*string, obligatorio*): UUID de la tarea.

---

## 3. Sistema y Archivos

### `bash_exec`
Ejecuta comandos de terminal en un directorio aislado con límite de tiempo.

- **Configuración**:
  ```yaml
  tools:
    bash:
      enabled: true
      working_dir: "/ruta/al/sandbox"
      timeout_seconds: 30
  ```
- **Parámetros**:
  - `command` (*string, obligatorio*): Comando de shell a ejecutar.
- **Salida**: Salida combinada de stdout y stderr.

---

### Herramientas de Sistema de Archivos
- `file_read(path)`: Lee el contenido de un archivo local.
- `file_write(path, content)`: Escribe contenido en un archivo local.
- `file_list(directory)`: Lista archivos y subcarpetas.

---

### `git_local`
Realiza operaciones Git seguras en un repositorio local:
- `git_status`: Comprueba el estado del repositorio de trabajo.
- `git_log`: Muestra el historial de commits recientes.
- `git_diff`: Muestra cambios en preparación o pendientes.
- `git_commit`: Realiza un commit con mensaje descriptivo.

---

## 4. Integraciones y APIs Externas

### Herramientas de Google Workspace
Requiere credenciales OAuth de cliente de Google (`google.oauth_client_id` & `google.oauth_client_secret`).
- **`gmail_search`**: Busca en la bandeja de entrada usando filtros de Gmail (ej: `is:unread from:jefe`).
- **`gmail_read`**: Lee el cuerpo completo del mensaje y metadados.
- **`gmail_send`**: Redacta y envía correos electrónicos.
- **`calendar_list_events`**: Consulta próximos eventos en un rango de fechas.
- **`calendar_create_event`**: Agenda una reunión en Google Calendar con título y horas.
- **`docs_read` / `docs_create`**: Lee o crea documentos en Google Docs.

---

### `github`
Interactúa con incidencias, pull requests y repositorios de GitHub.
- **Configuración**: `tools.github.api_token: "ghp_..."`
- `github_list_issues`: Lista issues con estado, etiquetas y asignados.
- `github_create_issue`: Abre una nueva issue en el repositorio.
- `github_list_prs`: Revisa pull requests abiertos o cerrados.
- `github_get_repo`: Obtiene estadísticas y metadados del repositorio.

---

### `notion`
Consulta y actualiza páginas y bases de datos en Notion.
- **Configuración**: `tools.notion.api_key: "secret_..."`
- `notion_search`: Busca páginas y bases de datos por título.
- `notion_get_page`: Recupera bloques de contenido de una página.
- `notion_create_page`: Crea una nueva página o registro en base de datos.

---

### `trello`
Gestiona tableros Kanban, listas y tarjetas en Trello.
- **Configuración**: `tools.trello.api_key` & `tools.trello.user_token`.
- `trello_list_boards`: Lista los tableros disponibles.
- `trello_list_cards`: Obtiene las tarjetas de una lista.
- `trello_create_card`: Añade una nueva tarjeta con título, descripción y fecha de entrega.

---

### `n8n`
Dispara flujos de automatización y webhooks en n8n.
- **Configuración**: `tools.n8n.base_url` & `tools.n8n.api_key`.
- `n8n_trigger_webhook`: Envía datos a un nodo webhook de n8n.
- `n8n_execute_workflow`: Ejecuta un flujo de trabajo mediante la API REST de n8n.
