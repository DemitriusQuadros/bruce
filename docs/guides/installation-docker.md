# Docker Compose Installation Guide

This guide walks you through deploying Bruce using Docker Compose for production, homelab, or VPS environments.

---

## 1. Prerequisites

- **Docker**: Version 24.0 or higher.
- **Docker Compose**: Version 2.0 or higher.
- **Git**: Installed on your host machine.
- **Ports**: 
  - `9090` (Bruce Web Dashboard & API).
  - `6379` (Internal Redis, optionally exposed if needed).

---

## 2. Step-by-Step Installation

### Step 1: Clone the Repository
```bash
git clone https://github.com/DemitriusQuadros/bruce.git
cd bruce
```

### Step 2: Configure Your Settings
Copy the example configuration file:
```bash
cp config.example.yml config.yml
```

Edit `config.yml` with your editor of choice:
```yaml
server:
  port: 8080

redis:
  addr: "bruce-redis:6379" # Connects to the Redis container in the same Docker network

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
    bot_token: "YOUR_DISCORD_BOT_TOKEN"
```

### Step 3: Inspect `docker-compose.yml`
Ensure your `docker-compose.yml` volume mounts persistent data to a local directory:
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

### Step 4: Build and Start
```bash
docker compose up -d --build
```

### Step 5: Verify the Deployment
1. Check running containers:
   ```bash
   docker compose ps
   ```
2. Check health endpoint:
   ```bash
   curl http://localhost:9090/health
   # Expected: {"status":"ok","uptime_seconds":...}
   ```
3. Tail container logs:
   ```bash
   docker compose logs -f bruce
   ```
4. Open the Web Dashboard in your browser: `http://<your-server-ip>:9090`.

---

## 3. Upgrading to the Latest Version

When new updates are pushed to the Bruce repository, updating your running container takes three commands:

```bash
# 1. Pull latest code
git pull origin master

# 2. Rebuild the image
docker compose build bruce

# 3. Recreate the container with zero downtime for data
docker compose up -d bruce
```

---

## 4. Backups & Data Persistence

All state is stored inside the `./bruce_data` directory:
- `bruce_data/bruce.db` — SQLite database (sessions, messages, schedules, config).
- `bruce_data/bruce.db-wal` — SQLite Write-Ahead Log.
- `bruce_data/artifacts/` — Saved HTML artifacts.
- `bruce_data/whatsapp.db` — WhatsApp multi-device authentication credentials (if enabled).

### Creating a Safe Backup
To create a safe backup while the database is active, use the SQLite backup command:
```bash
docker exec bruce sqlite3 /app/data/bruce.db ".backup '/app/data/backup-$(date +%Y%m%d).db'"
```
Or simply archive the directory when the container is paused:
```bash
tar -czvf bruce-backup-$(date +%Y%m%d).tar.gz ./bruce_data config.yml
```
