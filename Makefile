# Show help for each make target
help:
	@echo "Comandos disponíveis:"
	@grep -E '^[a-zA-Z_-]+:.*?## .+' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

.PHONY: build up down stop restart logs clean run run-dev build-binary test test-race test-e2e test-e2e-smoke vet fmt run-redis swag

build: ## Build docker images
	docker-compose build

up: ## Build and start all containers (Redis + Bruce)
	docker-compose up -d

run-redis: ## Start only the Redis container
	docker-compose up -d redis

down: ## Stop containers but keep volumes
	docker-compose down

stop: ## Stop containers without removing them
	docker-compose stop

restart: down up ## Restart containers

logs: ## Tail logs for all containers
	docker-compose logs -f

clean: ## Stop and remove containers, volumes, networks, and images
	docker-compose down -v --rmi all --remove-orphans

SWAG := $(shell go env GOPATH)/bin/swag

swag: ## Generate Swagger docs from annotations
	$(SWAG) init -g cmd/bruce/main.go -o docs/

run: swag ## Run bruce locally (generates docs first)
	CGO_ENABLED=1 go run cmd/bruce/main.go

build-binary: ## Compile bruce binary with embedded assets into bin/bruce
	@mkdir -p bin
	CGO_ENABLED=1 go build -o bin/bruce ./cmd/bruce

test: ## Run all unit tests
	CGO_ENABLED=1 go test ./...

test-race: ## Run all unit tests with race detector
	CGO_ENABLED=1 go test -race ./...

test-e2e: ## Run full E2E suite (requires running server + Redis)
	CGO_ENABLED=1 go test -v -timeout 120s ./tests/e2e/

test-e2e-smoke: ## Run E2E smoke scenarios only
	CGO_ENABLED=1 go test -v -timeout 60s ./tests/e2e/ -args -godog.tags="@smoke"

vet: ## Run go vet
	go vet ./...

fmt: ## Check formatting (lists unformatted files)
	gofmt -l .