# Show help for each make target
help:
	@echo "Comandos disponíveis:"
	@grep -E '^[a-zA-Z_-]+:.*?## .+' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

DOCKER_COMPOSE_FILE = docker-compose.yml

.PHONY: build up down stop restart logs clean

build: ## Build docker images
	docker-compose -f $(DOCKER_COMPOSE_FILE) build

up: build ## Up containers
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

down: ## Stop containers but keep volumes
	docker-compose -f $(DOCKER_COMPOSE_FILE) down

stop: ## Stop containers without removing them
	docker-compose -f $(DOCKER_COMPOSE_FILE) stop

restart: down up ## Restart containers

logs: ## Show logs for all containers
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

clean: ## Stop and remove containers, volumes, networks, and images
	docker-compose -f $(DOCKER_COMPOSE_FILE) down -v --rmi all --remove-orphans

run-api-local: ## Run the API project locally
	go run cmd/api/main.go

run-worker-local: ## Run the Worker project locally
	go run cmd/worker/main.go

run-console: ## Run the console project locally
	go run cmd/console/main.go