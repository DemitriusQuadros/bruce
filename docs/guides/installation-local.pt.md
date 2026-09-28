# Instalação para Desenvolvimento Local Bare-Metal

Este guia ensina como compilar e executar o Bruce diretamente no seu computador ou estação de trabalho sem Docker.

---

## 1. Pré-requisitos

### 1. Toolchain do Go
- **Go 1.23 ou mais recente** é obrigatório:
  ```bash
  go version
  ```

### 2. Compilador C (CGO Obrigatório)
O Bruce utiliza o `mattn/go-sqlite3`, que requer CGO para compilar o SQLite:
- **macOS**: Instale as ferramentas de linha de comando do Xcode:
  ```bash
  xcode-select --install
  ```
- **Linux (Ubuntu/Debian)**: Instale o pacote `build-essential`:
  ```bash
  sudo apt-get update && sudo apt-get install -y build-essential
  ```
- **Linux (Fedora/RHEL)**:
  ```bash
  sudo dnf groupinstall "Development Tools"
  ```

### 3. Instância do Redis
O Bruce precisa do Redis para as filas de tarefas do Asynq:
- **macOS (Homebrew)**:
  ```bash
  brew install redis
  brew services start redis
  ```
- **Linux**:
  ```bash
  sudo apt-get install redis-server
  sudo systemctl start redis-server
  ```
- **Via Docker (Alternativa)**:
  ```bash
  docker run -d --name local-redis -p 6379:6379 redis:7-alpine
  ```

---

## 2. Compilando & Executando

1. **Clonar o repositório**:
   ```bash
   git clone https://github.com/DemitriusQuadros/bruce.git
   cd bruce
   ```

2. **Configurar o `config.yml`**:
   ```bash
   cp config.example.yml config.yml
   ```
   Certifique-se de que `redis.addr` aponte para `localhost:6379`:
   ```yaml
   server:
     port: 8080

   redis:
     addr: "localhost:6379"

   claude:
     api_key: "sk-ant-api03-..."
   ```

3. **Executar diretamente com `go run`**:
   ```bash
   CGO_ENABLED=1 go run cmd/bruce/main.go
   ```

4. **Ou compilar o binário**:
   ```bash
   CGO_ENABLED=1 go build -o bin/bruce ./cmd/bruce
   ./bin/bruce
   ```

5. **Verificar**:
   Abra `http://localhost:8080` no seu navegador web.

---

## 3. Executando a Suíte de Testes

```bash
# Executar todos os testes unitários e de integração
go test -v ./...

# Executar testes com detector de condições de corrida (race conditions)
CGO_ENABLED=1 go test -race ./...

# Análise estática
go vet ./...
```

---

## 4. Desenvolvimento Frontend

A interface web do Bruce utiliza JavaScript Vanilla, CSS e HTML embutidos diretamente no executável Go:
- Os arquivos do frontend ficam na pasta `web/public/`.
- Nenhum passo de build com Node.js ou empacotador é necessário.
- Ao editar qualquer arquivo em `web/public/`, reinicie o comando `go run cmd/bruce/main.go` para embutir os arquivos atualizados.

### Opcional: Testes E2E com Playwright
Para executar os testes automatizados de ponta a ponta no navegador:
```bash
npm install
npx playwright test
```
