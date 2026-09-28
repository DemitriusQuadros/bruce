# Instalación para Desarrollo Local Bare-Metal

Esta guía explica cómo compilar y ejecutar Bruce directamente en tu estación de trabajo sin necesidad de Docker.

---

## 1. Requisitos Previos

### 1. Entorno de Go
- **Go 1.23 o superior** es necesario:
  ```bash
  go version
  ```

### 2. Compilador de C (CGO Requerido)
Bruce utiliza `mattn/go-sqlite3`, que requiere CGO para compilar SQLite:
- **macOS**: Instala las herramientas de línea de comandos de Xcode:
  ```bash
  xcode-select --install
  ```
- **Linux (Ubuntu/Debian)**: Instala `build-essential`:
  ```bash
  sudo apt-get update && sudo apt-get install -y build-essential
  ```
- **Linux (Fedora/RHEL)**:
  ```bash
  sudo dnf groupinstall "Development Tools"
  ```

### 3. Instancia de Redis
Bruce requiere Redis para gestionar las colas de Asynq:
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
- **Con Docker (Alternativa)**:
  ```bash
  docker run -d --name local-redis -p 6379:6379 redis:7-alpine
  ```

---

## 2. Compilación y Ejecución

1. **Clonar el repositorio**:
   ```bash
   git clone https://github.com/DemitriusQuadros/bruce.git
   cd bruce
   ```

2. **Configurar `config.yml`**:
   ```bash
   cp config.example.yml config.yml
   ```
   Asegúrate de que `redis.addr` apunte a `localhost:6379`:
   ```yaml
   server:
     port: 8080

   redis:
     addr: "localhost:6379"

   claude:
     api_key: "sk-ant-api03-..."
   ```

3. **Ejecutar directamente con `go run`**:
   ```bash
   CGO_ENABLED=1 go run cmd/bruce/main.go
   ```

4. **O compilar el binario**:
   ```bash
   CGO_ENABLED=1 go build -o bin/bruce ./cmd/bruce
   ./bin/bruce
   ```

5. **Verificar**:
   Abre `http://localhost:8080` en tu navegador.

---

## 3. Ejecución de Pruebas

```bash
# Ejecutar todas las pruebas unitarias y de integración
go test -v ./...

# Pruebas con detector de condiciones de carrera (race conditions)
CGO_ENABLED=1 go test -race ./...

# Análisis estático
go vet ./...
```

---

## 4. Desarrollo Frontend

La interfaz web de Bruce utiliza JavaScript Vanilla, CSS y HTML integrados directamente en el binario de Go:
- Los recursos frontend se encuentran en `web/public/`.
- No requiere Node.js para compilar ni empaquetadores como Webpack o Vite.
- Al editar cualquier archivo en `web/public/`, reinicia `go run cmd/bruce/main.go` para volver a embutir los recursos.

### Opcional: Pruebas E2E con Playwright
Para ejecutar las pruebas automatizadas del navegador:
```bash
npm install
npx playwright test
```
