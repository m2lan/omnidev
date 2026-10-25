# ==============================================================================
# OmniDev — AI knowledge & coding workbench (modular monolith)
# Backend: apps/gateway (auth + chat + knowledge + agent)
# Shared:  packages/go-common, ../knowledge-engine
# Frontend: apps/web
# ==============================================================================

SHELL := /bin/bash
.DEFAULT_GOAL := help

PROJECT_NAME := omnidev
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

GO := go
GOFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"
CGO_ENABLED := 0

DOCKER := docker
DOCKER_COMPOSE := docker compose
COMPOSE_FILE_DEV := deploy/docker/docker-compose.dev.yml
COMPOSE_FILE_INFRA := deploy/docker/docker-compose.infra.yml
COMPOSE := $(DOCKER_COMPOSE) -f $(COMPOSE_FILE_INFRA) -f $(COMPOSE_FILE_DEV)

PNPM := pnpm
DB_URL := postgres://omnidev:omnidev@localhost:5432/omnidev?sslmode=disable

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ==============================================================================
# Development environment
# ==============================================================================

.PHONY: dev-infra
dev-infra: ## Start PostgreSQL, Redis, MinIO, Tika and run migrations
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE_INFRA) up -d
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE_INFRA) ps

.PHONY: dev
dev: ## Start infrastructure plus the gateway and nginx
	$(COMPOSE) up -d --build
	@echo "Gateway: http://localhost:9090 (web: pnpm --filter web dev)"

.PHONY: dev-down
dev-down: ## Stop the development environment
	$(COMPOSE) down

.PHONY: dev-clean
dev-clean: ## Stop everything and delete data volumes
	$(COMPOSE) down -v

.PHONY: dev-logs
dev-logs: ## Follow development logs
	$(COMPOSE) logs -f

.PHONY: dev-status
dev-status: ## Show container status
	$(COMPOSE) ps

# ==============================================================================
# Frontend
# ==============================================================================

.PHONY: web-install
web-install: ## Install frontend dependencies
	$(PNPM) install

.PHONY: web-dev
web-dev: ## Start frontend dev server
	$(PNPM) --filter web dev

.PHONY: web-build
web-build: ## Build frontend
	$(PNPM) --filter web build

.PHONY: web-lint
web-lint: ## Lint frontend
	$(PNPM) --filter web lint

.PHONY: web-typecheck
web-typecheck: ## Type check frontend
	$(PNPM) --filter web typecheck

# ==============================================================================
# Backend
# ==============================================================================

.PHONY: go-tidy
go-tidy: ## Tidy Go modules
	cd apps/gateway && $(GO) mod tidy
	cd packages/go-common && $(GO) mod tidy

.PHONY: build
build: ## Build the gateway binary into bin/
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -o bin/gateway ./apps/gateway/cmd/gateway

.PHONY: run-gateway
run-gateway: ## Run the gateway locally (requires dev-infra)
	$(GO) run ./apps/gateway/cmd/gateway

.PHONY: test
test: ## Run Go tests
	$(GO) test ./apps/gateway/... ./packages/go-common/... -count=1

.PHONY: test-race
test-race: ## Run Go tests with the race detector
	$(GO) test ./apps/gateway/... ./packages/go-common/... -race -count=1

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./apps/gateway/... ./packages/go-common/...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./apps/gateway/... ./packages/go-common/...

.PHONY: fmt
fmt: ## Format Go code
	gofmt -s -w apps/gateway packages/go-common

# ==============================================================================
# Database
# ==============================================================================

.PHONY: db-migrate
db-migrate: ## Apply gateway migrations to the local database
	migrate -path apps/gateway/migrations -database "$(DB_URL)" up

.PHONY: db-migrate-down
db-migrate-down: ## Roll back the last migration
	migrate -path apps/gateway/migrations -database "$(DB_URL)" down 1

.PHONY: db-migrate-create
db-migrate-create: ## Create a migration (make db-migrate-create NAME=add_xxx)
	migrate create -ext sql -dir apps/gateway/migrations -seq $(NAME)

# ==============================================================================
# Docker
# ==============================================================================

.PHONY: docker-build
docker-build: ## Build the gateway image (build context includes knowledge-engine)
	$(DOCKER) build -t $(PROJECT_NAME)/gateway:$(VERSION) -f apps/gateway/Dockerfile ..

.PHONY: sandbox-images
sandbox-images: ## Pre-pull the agent sandbox runtime images
	$(DOCKER) pull python:3.12-alpine
	$(DOCKER) pull node:22-alpine
	$(DOCKER) pull golang:1.22-alpine
	$(DOCKER) pull alpine:3.19

# ==============================================================================
# Utilities
# ==============================================================================

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin/ coverage.out coverage.html

.PHONY: setup
setup: ## Initial setup: env file plus frontend dependencies
	cp -n .env.example .env 2>/dev/null || true
	$(PNPM) install
	@echo "Next: make dev-infra && make run-gateway && make web-dev"
