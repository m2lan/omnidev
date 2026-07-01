# OmniDev AI Platform

[中文](README_zh.md)

All-in-One AI development platform integrating IDE, Agent, RAG, Workflow, Deploy and more.

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+
- pnpm 9+
- Docker & Docker Compose
- Make

### 1. Clone the Project

```bash
git clone git@github.com:m2lan/omnidev.git
cd omnidev
```

### 2. Environment Configuration

```bash
cp .env.example .env
# Edit .env to set your API Key and other configurations
```

### 3. Start Infrastructure

```bash
# Start PostgreSQL, Redis, Kafka, MinIO, Elasticsearch, Temporal
make dev-infra

# Check status
make dev-status
```

### 4. Install Dependencies

```bash
# Go dependencies
make go-mod

# Frontend dependencies
make web-install
```

### 5. Run Database Migrations

```bash
make db-migrate
```

### 6. Start Development Servers

```bash
# Start API Gateway
make run-gateway

# Start frontend (another terminal)
make web-dev
```

### 7. Access

| Service | URL |
|---------|-----|
| Web App | http://localhost:3000 |
| API Gateway | http://localhost:9090 |
| Health Check | http://localhost:9090/health |
| API Info | http://localhost:9090/info |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |
| Kafka | localhost:9092 |
| MinIO Console | http://localhost:9001 |
| Elasticsearch | http://localhost:9200 |
| Temporal UI | http://localhost:8088 |
| Grafana | http://localhost:3001 |
| Prometheus | http://localhost:9091 |

## Project Structure

```
omnidev/
├── apps/                    # Application layer
│   ├── web/                 # Frontend Next.js
│   ├── gateway/             # API Gateway (Go)
│   ├── services/            # Backend microservices
│   └── workers/             # Background workers
├── packages/                # Shared packages
│   ├── proto/               # Protobuf definitions
│   ├── go-common/           # Go common library
│   ├── ts-common/           # TypeScript common library
│   └── ui/                  # UI component library
├── deploy/                  # Deployment configs
│   ├── docker/              # Docker Compose
│   ├── helm/                # Helm Charts
│   ├── terraform/           # Terraform configs
│   └── k8s/                 # K8s manifests
├── docs/                    # Documentation
├── scripts/                 # Utility scripts
└── tools/                   # Development tools
```

## Development Commands

```bash
# View all commands
make help

# Development environment
make dev              # Start full development environment
make dev-infra        # Start infrastructure only
make dev-down         # Stop development environment
make dev-logs         # View logs

# Build
make build-all        # Build all services
make build-gateway    # Build API Gateway

# Test
make test             # Run all tests
make test-short       # Run short tests
make test-integration # Run integration tests
make test-coverage    # Generate coverage report

# Code quality
make lint             # Run linter
make fmt              # Format code
make check            # Run all checks

# Code generation
make gen-proto        # Generate Protobuf code
make gen-wire         # Generate Wire code
make gen-swagger      # Generate API docs

# Database
make db-migrate       # Run migrations
make db-migrate-down  # Rollback migration
make db-migrate-create NAME=create_xxx  # Create new migration

# Docker
make docker-build     # Build all Docker images
make docker-build-gateway  # Build single image

# Kubernetes
make k8s-apply        # Deploy to K8s
make helm-install     # Install Helm Chart
```

## Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | Next.js 15, React 19, Tailwind, Shadcn/ui |
| API Gateway | Go, Gin |
| Backend | Go, gRPC |
| Database | PostgreSQL 16 + pgvector |
| Cache | Redis 7 |
| Queue | Kafka (KRaft) |
| Storage | MinIO |
| Search | Elasticsearch 8 |
| Workflow | Temporal |
| Container | Docker, Kubernetes |
| Observability | Prometheus, Grafana, Loki, Jaeger |

## Architecture Documentation

Detailed architecture documentation is located in the `docs/architecture/` directory:

- [Executive Summary](docs/architecture/00-EXECUTIVE-SUMMARY.md)
- [Requirements Analysis](docs/architecture/01-REQUIREMENTS-ANALYSIS.md)
- [Feature Boundary](docs/architecture/02-FEATURE-BOUNDARY.md)
- [Non-Functional Requirements](docs/architecture/03-NON-FUNCTIONAL-REQUIREMENTS.md)
- [Technology Selection](docs/architecture/04-TECHNOLOGY-SELECTION.md)
- [System Architecture](docs/architecture/05-SYSTEM-ARCHITECTURE.md)
- [Database Design](docs/architecture/06-DATABASE-DESIGN.md)
- [Directory Structure](docs/architecture/07-DIRECTORY-STRUCTURE.md)
- [Development Standards](docs/architecture/08-DEVELOPMENT-STANDARDS.md)
- [Milestone Plan](docs/architecture/09-MILESTONE-PLAN.md)

## API Documentation

The API follows RESTful conventions. All response formats:

```json
// Success
{
  "data": { ... },
  "meta": { "page_size": 20, "next_page_token": "...", "total_count": 100 }
}

// Error
{
  "error": { "code": 400, "message": "...", "detail": "...", "request_id": "..." }
}
```

### Authentication

```bash
# Bearer Token
curl -H "Authorization: Bearer <token>" http://localhost:9090/api/v1/users/me

# API Key
curl -H "Authorization: Bearer <api-key>" http://localhost:9090/api/v1/users/me
```

## Troubleshooting

### Database Connection Failed

```bash
# Check if PostgreSQL is running
docker compose -f deploy/docker/docker-compose.infra.yml ps postgres

# View logs
docker compose -f deploy/docker/docker-compose.infra.yml logs postgres
```

### Redis Connection Failed

```bash
# Check if Redis is running
docker compose -f deploy/docker/docker-compose.infra.yml ps redis

# Test connection
docker compose -f deploy/docker/docker-compose.infra.yml exec redis redis-cli ping
```

### Port Already in Use

```bash
# Find process using the port
netstat -ano | findstr :9090
# or
lsof -i :9090

# Kill process
taskkill /PID <pid> /F
```

### Go Module Issues

```bash
# Clean module cache
go clean -modcache

# Re-download
go mod download all

# Sync workspace
go work sync
```

## Contributing

1. Fork the project
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Commit changes: `git commit -m 'feat(scope): add my feature'`
4. Push branch: `git push origin feature/my-feature`
5. Create a Pull Request

## License

[MIT License](LICENSE)
