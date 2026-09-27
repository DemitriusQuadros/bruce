# Bare-Metal Local Development Installation

This guide walks you through compiling and running Bruce directly on your local workstation without Docker.

---

## 1. Prerequisites

### 1. Go Toolchain
- **Go 1.23 or newer** is required:
  ```bash
  go version
  ```

### 2. C Compiler (CGO Required)
Bruce uses `mattn/go-sqlite3`, which requires CGO to compile SQLite:
- **macOS**: Install Xcode Command Line Tools:
  ```bash
  xcode-select --install
  ```
- **Linux (Ubuntu/Debian)**: Install `build-essential`:
  ```bash
  sudo apt-get update && sudo apt-get install -y build-essential
  ```
- **Linux (Fedora/RHEL)**:
  ```bash
  sudo dnf groupinstall "Development Tools"
  ```

### 3. Redis Instance
Bruce requires Redis for Asynq job queues:
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
- **Docker (Alternative)**:
  ```bash
  docker run -d --name local-redis -p 6379:6379 redis:7-alpine
  ```

---

## 2. Compiling & Running

1. **Clone the repository**:
   ```bash
   git clone https://github.com/DemitriusQuadros/bruce.git
   cd bruce
   ```

2. **Configure `config.yml`**:
   ```bash
   cp config.example.yml config.yml
   ```
   Ensure `redis.addr` points to `localhost:6379`:
   ```yaml
   server:
     port: 8080

   redis:
     addr: "localhost:6379"

   claude:
     api_key: "sk-ant-api03-..."
   ```

3. **Run directly with `go run`**:
   ```bash
   CGO_ENABLED=1 go run cmd/bruce/main.go
   ```

4. **Or compile a binary**:
   ```bash
   CGO_ENABLED=1 go build -o bin/bruce ./cmd/bruce
   ./bin/bruce
   ```

5. **Verify**:
   Open `http://localhost:8080` in your web browser.

---

## 3. Running the Test Suite

```bash
# Run all unit and integration tests
go test -v ./...

# Run tests with race condition detector
CGO_ENABLED=1 go test -race ./...

# Static analysis
go vet ./...
```

---

## 4. Frontend Development

Bruce's web interface uses modern Vanilla JavaScript, CSS, and HTML embedded in the Go binary:
- Frontend assets are located in `web/public/`.
- No Node.js build step, bundler, or transpilation is required.
- If you edit files in `web/public/`, re-run `go run cmd/bruce/main.go` to re-embed the updated files.

### Optional: Playwright E2E Tests
To run the automated browser tests against the web interface:
```bash
npm install
npx playwright test
```
