# Guía de Instalación con Docker Compose

Esta guía explica cómo desplegar Bruce utilizando Docker Compose para entornos de producción, homelab o VPS.

---

## 1. Requisitos Previos

- **Docker**: Versión 24.0 o superior.
- **Docker Compose**: Versión 2.0 o superior.
- **Git**: Instalado en el sistema operativo anfitrión.
- **Puertos**: 
  - `9090` (Panel Web y API REST de Bruce).
  - `6379` (Redis interno, expuesto opcionalmente si es necesario).

---

## 2. Instalación Paso a Paso

### Paso 1: Clonar el Repositorio
```bash
git clone https://github.com/DemitriusQuadros/bruce.git
cd bruce
```

### Paso 2: Configurar las Variables
Copia el archivo de configuración de ejemplo:
```bash
cp config.example.yml config.yml
```

Edita `config.yml` con tu editor preferido:
```yaml
server:
  port: 8080

redis:
  addr: "bruce-redis:6379" # Conecta al contenedor Redis en la misma red Docker

claude:
  api_key: "sk-ant-api03-..."
  model: "claude-haiku-4-5-20251001"
  max_tokens: 8192
  context_window: 15

llm:
  provider: "claude"
  background_provider: "claude"

connectors:
  discord:
    enabled: true
    bot_token: "TU_TOKEN_DE_DISCORD"
```

### Paso 3: Inspeccionar `docker-compose.yml`
Asegúrate de que los volúmenes en `docker-compose.yml` monten los datos en un directorio local:
```yaml
services:
  bruce-redis:
    image: redis:7-alpine
    container_name: bruce-redis
    restart: unless-stopped
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  bruce:
    build: .
    container_name: bruce
    restart: unless-stopped
    ports:
      - "9090:8080"
    volumes:
      - ./bruce_data:/app/data
      - ./config.yml:/app/config.yml:ro
    depends_on:
      bruce-redis:
        condition: service_healthy

volumes:
  redis_data:
```

### Paso 4: Construir e Iniciar
```bash
docker compose up -d --build
```

### Paso 5: Verificar el Despliegue
1. Comprueba los contenedores activos:
   ```bash
   docker compose ps
   ```
2. Consulta el endpoint de estado de salud (health check):
   ```bash
   curl http://localhost:9090/health
   # Resultado esperado: {"status":"ok","uptime_seconds":...}
   ```
3. Sigue los registros en tiempo real:
   ```bash
   docker compose logs -f bruce
   ```
4. Abre el Panel Web en tu navegador: `http://<ip-de-tu-servidor>:9090`.

---

## 3. Actualización a la Última Versión

Cuando se publiquen nuevas actualizaciones en el repositorio de Bruce, actualizar el contenedor en ejecución requiere solo tres comandos:

```bash
# 1. Obtener los últimos cambios
git pull origin master

# 2. Reconstruir la imagen
docker compose build bruce

# 3. Reiniciar el contenedor sin perder datos
docker compose up -d bruce
```

---

## 4. Copias de Seguridad y Persistencia de Datos

Todo el estado se almacena en el directorio `./bruce_data`:
- `bruce_data/bruce.db` — Base de datos SQLite (sesiones, mensajes, programaciones, configuración).
- `bruce_data/bruce.db-wal` — Archivo Write-Ahead Log de SQLite.
- `bruce_data/artifacts/` — Artefactos HTML guardados.
- `bruce_data/whatsapp.db` — Credenciales de autenticación multidispositivo de WhatsApp (si está habilitado).

### Creación de una Copia de Seguridad Segura
Para realizar una copia de seguridad mientras la base de datos está en uso sin riesgo de corrupción, usa el comando nativo de SQLite:
```bash
docker exec bruce sqlite3 /app/data/bruce.db ".backup '/app/data/backup-$(date +%Y%m%d).db'"
```
O empaqueta el directorio con el contenedor pausado:
```bash
tar -czvf bruce-backup-$(date +%Y%m%d).tar.gz ./bruce_data config.yml
```
