# ==============================================================================
# OmniDev AI Platform — Makefile
# ==============================================================================

SHELL := /bin/bash
.DEFAULT_GOAL := help

# --- Variables ---
PROJECT_NAME := omnidev
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

GO := go
GOFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"
CGO_ENABLED := 0

DOCKER := docker
DOCKER_COMPOSE := docker compose
COMPOSE_FILE := deploy/docker/docker-compose.yml
COMPOSE_FILE_DEV := deploy/docker/docker-compose.dev.yml
COMPOSE_FILE_INFRA := deploy/docker/docker-compose.infra.yml

PNPM := pnpm

# --- Services ---
SERVICES := gateway user chat agent rag ide workflow mcp deploy billing admin notification monitor
WORKERS := doc-processor embedding-worker billing-worker notification-worker

# ==============================================================================
# Help
# ==============================================================================

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ==============================================================================
# Development Environment
# ==============================================================================

.PHONY: dev-infra
dev-infra: ## Start infrastructure services (PostgreSQL, Redis, Kafka, MinIO, ES)
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE_INFRA) up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE_INFRA) ps

.PHONY: dev-infra-down
dev-infra-down: ## Stop infrastructure services
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE_INFRA) down

.PHONY: dev-infra-clean
dev-infra-clean: ## Stop and remove infrastructure volumes
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE_INFRA) down -v

.PHONY: dev
dev: dev-infra ## Start full development environment
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE_DEV) up -d
	@echo "Development environment is ready!"
	@echo "  Gateway:    http://localhost:9090"
	@echo "  Web:        http://localhost:3000"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  Redis:      localhost:6379"
	@echo "  Kafka:      localhost:9092"
	@echo "  MinIO:      http://localhost:9000 (console: 9001)"
	@echo "  ES:         http://localhost:9200"
	@echo "  Temporal:   http://localhost:8088"
	@echo "  Grafana:    http://localhost:3001"
	@echo "  Prometheus: http://localhost:9091"

.PHONY: dev-down
dev-down: ## Stop full development environment
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE_DEV) down

.PHONY: dev-clean
dev-clean: ## Stop and remove all volumes
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE_DEV) down -v

.PHONY: dev-logs
dev-logs: ## Show development logs
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE_DEV) logs -f

.PHONY: dev-status
dev-status: ## Show development environment status
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE_INFRA) ps

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

.PHONY: web-test
web-test: ## Test frontend
	$(PNPM) --filter web test

.PHONY: web-typecheck
web-typecheck: ## Type check frontend
	$(PNPM) --filter web typecheck

# ==============================================================================
# Backend — Build
# ==============================================================================

.PHONY: go-mod
go-mod: ## Download Go dependencies
	$(GO) mod download all

.PHONY: go-tidy
go-tidy: ## Tidy Go modules
	$(GO) work sync
	@for dir in $(SERVICES); do \
		echo "Tidying apps/services/$$dir..."; \
		cd apps/services/$$dir && $(GO) mod tidy && cd ../../..; \
	done
	@for dir in $(WORKERS); do \
		echo "Tidying apps/workers/$$dir..."; \
		cd apps/workers/$$dir && $(GO) mod tidy && cd ../../..; \
	done
	cd apps/gateway && $(GO) mod tidy && cd ../..
	cd packages/go-common && $(GO) mod tidy && cd ../..

define build_service
	@echo "Building $(1)..."
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -o bin/$(1) ./apps/$(2)/$(1)/cmd/$(1)/main.go
endef

.PHONY: build-gateway
build-gateway: ## Build API Gateway
	$(call build_service,gateway,services)

.PHONY: build-services
build-services: ## Build all backend services
	@for svc in $(SERVICES); do \
		if [ "$$svc" = "gateway" ]; then \
			$(call build_service,gateway,.) ; \
		else \
			$(call build_service,$$svc,services) ; \
		fi \
	done

.PHONY: build-workers
workers: ## Build all workers
	@for w in $(WORKERS); do \
		echo "Building $$w..."; \
		CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -o bin/$$w ./apps/workers/$$w/main.go; \
	done

.PHONY: build-all
build-all: build-services build-workers ## Build everything

# ==============================================================================
# Backend — Run
# ==============================================================================

.PHONY: run-gateway
run-gateway: ## Run API Gateway locally
	$(GO) run ./apps/gateway/cmd/gateway/main.go

.PHONY: run-user
run-user: ## Run User Service locally
	$(GO) run ./apps/services/user/cmd/user/main.go

.PHONY: run-chat
run-chat: ## Run Chat Service locally
	$(GO) run ./apps/services/chat/cmd/chat/main.go

# ==============================================================================
# Backend — Test
# ==============================================================================

.PHONY: test
test: ## Run all Go tests
	$(GO) test ./... -v -race -count=1 -coverprofile=coverage.out

.PHONY: test-short
test-short: ## Run short tests only
	$(GO) test ./... -short -v -count=1

.PHONY: test-integration
test-integration: ## Run integration tests (requires running infra)
	$(GO) test ./... -v -race -count=1 -tags=integration

.PHONY: test-coverage
test-coverage: test ## Show test coverage
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: test-bench
test-bench: ## Run benchmarks
	$(GO) test ./... -bench=. -benchmem

# ==============================================================================
# Backend — Lint
# ==============================================================================

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with auto-fix
	golangci-lint run --fix ./...

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

# ==============================================================================
# Code Generation
# ==============================================================================

.PHONY: gen-proto
gen-proto: ## Generate protobuf code
	@echo "Generating protobuf code..."
	buf generate packages/proto
	@echo "Done!"

.PHONY: gen-wire
gen-wire: ## Generate Wire dependency injection code
	@for svc in $(SERVICES); do \
		if [ -f "apps/services/$$svc/internal/wire.go" ]; then \
			echo "Generating Wire for $$svc..."; \
			cd apps/services/$$svc && wire ./internal/ && cd ../../..; \
		fi \
	done

.PHONY: gen-swagger
gen-swagger: ## Generate Swagger/OpenAPI documentation
	@echo "Generating OpenAPI docs..."
	swag init -g apps/gateway/cmd/gateway/main.go -o docs/api

.PHONY: gen-all
gen-all: gen-proto gen-wire gen-swagger ## Generate all code

# ==============================================================================
# Database
# ==============================================================================

.PHONY: db-migrate
db-migrate: ## Run database migrations
	@echo "Running migrations..."
	migrate -path apps/services/user/migrations -database "postgres://omnidev:omnidev@localhost:5432/omnidev?sslmode=disable" up

.PHONY: db-migrate-down
db-migrate-down: ## Rollback database migrations
	migrate -path apps/services/user/migrations -database "postgres://omnidev:omnidev@localhost:5432/omnidev?sslmode=disable" down 1

.PHONY: db-migrate-create
db-migrate-create: ## Create new migration (usage: make db-migrate-create NAME=create_xxx)
	migrate create -ext sql -dir apps/services/user/migrations -seq $(NAME)

.PHONY: db-seed
db-seed: ## Seed database with test data
	$(GO) run scripts/seed.go

.PHONY: db-reset
db-reset: ## Reset database (drop + migrate + seed)
	$(GO) run scripts/reset_db.go

# ==============================================================================
# Docker
# ==============================================================================

.PHONY: docker-build
docker-build: ## Build all Docker images
	@for svc in $(SERVICES); do \
		echo "Building $$svc..."; \
		$(DOCKER) build -t $(PROJECT_NAME)/$$svc:$(VERSION) -f apps/services/$$svc/Dockerfile .; \
	done

.PHONY: docker-build-%
docker-build-%: ## Build specific Docker image (e.g., make docker-build-gateway)
	$(DOCKER) build -t $(PROJECT_NAME)/$*:$(VERSION) -f apps/services/$*/Dockerfile .

.PHONY: docker-push
docker-push: ## Push all Docker images
	@for svc in $(SERVICES); do \
		echo "Pushing $$svc..."; \
		$(DOCKER) push $(PROJECT_NAME)/$$svc:$(VERSION); \
	done

# ==============================================================================
# Kubernetes
# ==============================================================================

.PHONY: k8s-apply
k8s-apply: ## Apply Kubernetes manifests
	kubectl apply -k deploy/k8s/overlays/staging

.PHONY: k8s-delete
k8s-delete: ## Delete Kubernetes resources
	kubectl delete -k deploy/k8s/overlays/staging

.PHONY: k8s-status
k8s-status: ## Show Kubernetes status
	kubectl get pods,svc,ingress -n omnidev

.PHONY: helm-install
helm-install: ## Install Helm chart
	helm install omnidev deploy/helm/omnidev -f deploy/helm/values-staging.yaml

.PHONY: helm-upgrade
helm-upgrade: ## Upgrade Helm chart
	helm upgrade omnidev deploy/helm/omnidev -f deploy/helm/values-staging.yaml

# ==============================================================================
# Terraform
# ==============================================================================

.PHONY: tf-init
tf-init: ## Initialize Terraform
	cd deploy/terraform/environments/staging && terraform init

.PHONY: tf-plan
tf-plan: ## Plan Terraform changes
	cd deploy/terraform/environments/staging && terraform plan

.PHONY: tf-apply
tf-apply: ## Apply Terraform changes
	cd deploy/terraform/environments/staging && terraform apply

# ==============================================================================
# Utilities
# ==============================================================================

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/ coverage.out coverage.html
	$(PNPM) turbo clean

.PHONY: fmt
fmt: ## Format Go code
	$(GO) fmt ./...
	gofmt -s -w .

.PHONY: check
check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)

.PHONY: setup
setup: ## Initial project setup
	@echo "Setting up development environment..."
	$(PNPM) install
	$(GO) mod download all
	cp -n .env.example .env 2>/dev/null || true
	@echo "Setup complete! Run 'make dev' to start."
