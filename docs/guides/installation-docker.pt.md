# Guia de Instalação com Docker Compose

Este guia ensina como implantar o Bruce utilizando o Docker Compose para ambientes de produção, homelab ou VPS.

---

## 1. Pré-requisitos

- **Docker**: Versão 24.0 ou superior.
- **Docker Compose**: Versão 2.0 ou superior.
- **Git**: Instalado no sistema operacional do servidor.
- **Portas**: 
  - `9090` (Painel Web e API REST do Bruce).
  - `6379` (Redis interno, exposto opcionalmente se necessário).

---

## 2. Instalação Passo a Passo

### Passo 1: Clonar o Repositório
```bash
git clone https://github.com/DemitriusQuadros/bruce.git
cd bruce
```

### Passo 2: Configurar Suas Variáveis
Copie o arquivo de exemplo de configuração:
```bash
cp config.example.yml config.yml
```

Edite o arquivo `config.yml` no seu editor favorito:
```yaml
server:
  port: 8080

redis:
  addr: "bruce-redis:6379" # Conecta ao contêiner Redis na mesma rede Docker

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
    bot_token: "SEU_TOKEN_DO_DISCORD"
```

### Passo 3: Inspecionar o `docker-compose.yml`
Garanta que os volumes do seu `docker-compose.yml` persistam os dados em uma pasta local:
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

### Passo 4: Compilar e Iniciar
```bash
docker compose up -d --build
```

### Passo 5: Verificar o Funcionamento
1. Cheque os contêineres ativos:
   ```bash
   docker compose ps
   ```
2. Teste o endpoint de integridade (health check):
   ```bash
   curl http://localhost:9090/health
   # Retorno esperado: {"status":"ok","uptime_seconds":...}
   ```
3. Acompanhe os logs do contêiner:
   ```bash
   docker compose logs -f bruce
   ```
4. Acesse o Painel Web no seu navegador: `http://<ip-do-seu-servidor>:9090`.

---

## 3. Atualizando para a Versão Mais Recente

Quando novas atualizações forem publicadas no repositório do Bruce, atualizar o contêiner em execução requer apenas três comandos:

```bash
# 1. Puxar o código mais recente
git pull origin master

# 2. Reconstruir a imagem
docker compose build bruce

# 3. Recriar o contêiner com persistência contínua de dados
docker compose up -d bruce
```

---

## 4. Backups & Persistência de Dados

Todo o estado do Bruce fica salvo na pasta `./bruce_data`:
- `bruce_data/bruce.db` — Banco de dados SQLite (sessões, mensagens, agendamentos, configurações).
- `bruce_data/bruce.db-wal` — Arquivo Write-Ahead Log do SQLite.
- `bruce_data/artifacts/` — Artefatos HTML gerados pelo assistente.
- `bruce_data/whatsapp.db` — Credenciais de autenticação multi-aparelho do WhatsApp (se habilitado).

### Criando um Backup Seguro
Para criar um backup com o banco de dados rodando sem corrupção, use o comando nativo de backup do SQLite:
```bash
docker exec bruce sqlite3 /app/data/bruce.db ".backup '/app/data/backup-$(date +%Y%m%d).db'"
```
Ou compacte a pasta enquanto o contêiner estiver pausado:
```bash
tar -czvf bruce-backup-$(date +%Y%m%d).tar.gz ./bruce_data config.yml
```
