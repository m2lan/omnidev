<p align="center">
  <img src="docs/assets/omnidev-logo.svg" alt="OmniDev AI Platform" width="120" height="120">
</p>

<h1 align="center">OmniDev AI Platform</h1>

<p align="center">
  <strong>All-in-One AI Development Platform</strong>
</p>

<p align="center">
  Open your browser, write code with AI, manage projects, and deploy — all in one place.
</p>

<p align="center">
  <a href="README_zh.md">中文文档</a> · <a href="#-quickstart">Quickstart</a> · <a href="#-architecture">Architecture</a> · <a href="docs/architecture/">Docs</a>
</p>

---

## What is OmniDev?

OmniDev integrates an AI coding assistant, online IDE, Agent orchestration, RAG knowledge base, CI/CD pipelines, and monitoring into a single platform. It's designed for developers and teams who want to build, ship, and operate software without switching between a dozen tools.

## Key Features

### Online IDE
- **Monaco Editor** — syntax highlighting, multi-cursor, minimap, file search & replace
- **AI Code Completion** — Tab completion, multi-line suggestions, context-aware via LSP
- **Agent Code Editing** — AI Agent directly modifies code with Diff preview, accept/reject/undo
- **Terminal Emulator** — xterm.js, multi-tab, SSH, PTY support
- **Git Integration** — commit/push/pull/branch/merge, visual Diff
- **Code Review** — inline comments, AI-assisted review

### Agent System
- **Natural Language Tasks** — describe what you want, Agent decomposes, plans, and executes
- **Tool Calling** — built-in tools + custom MCP tools with Function Calling support
- **Visual Orchestration** — drag-and-drop canvas, node wiring, conditional branching
- **Sandbox Execution** — Docker isolation, resource limits, filesystem mounting
- **Execution Monitoring** — real-time logs, step tracing, intermediate state inspection
- **Agent Templates** — pre-built: code review, test generation, doc generation, etc.
- **Multi-Agent Collaboration** — CrewAI-style, role assignment, task delegation
- **Agent Marketplace** — export configs, ratings, install stats

### RAG Knowledge Base
- Multi-format upload (PDF, Markdown, code files)
- Hybrid search: vector (pgvector) + keyword (Elasticsearch)
- Code-aware chunking & indexing
- Integration with Chat and Agent for context injection

### CI/CD & Deployment
- Natural language pipeline configuration
- Kubernetes-native deployment
- Multi-cloud support (AWS/GCP/Azure)
- Git-triggered automated pipelines

### Monitoring & Observability
- Built-in OpenTelemetry integration
- Prometheus + Grafana + Loki + Jaeger stack
- AI-powered anomaly detection
- Custom alerting rules

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.22+, Gin, gRPC, Wire (DI) |
| Frontend | Next.js 15 (App Router), React 19, Tailwind CSS, Shadcn/ui |
| Database | PostgreSQL 16 + pgvector, Redis 7, Kafka (KRaft) |
| Storage | MinIO (S3 compatible) |
| Search | Elasticsearch 8 |
| Workflow | Temporal |
| Infra | Docker, Kubernetes, Helm, Terraform |
| Observability | Prometheus, Grafana, Loki, Jaeger, OpenTelemetry |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Gateway (Kong/Custom)                      │
│              Rate Limit · Auth · Routing · WAF               │
└──────────┬──────────┬──────────┬──────────┬─────────────────┘
           │          │          │          │
     ┌─────┴───┐ ┌───┴────┐ ┌──┴───┐ ┌───┴─────┐
     │   IDE   │ │  Chat  │ │Agent │ │   RAG   │
     │ Service │ │ Engine │ │System│ │ Service │
     └─────────┘ └────────┘ └──────┘ └─────────┘
           │          │          │          │
     ┌─────┴───┐ ┌───┴────┐ ┌──┴───┐ ┌───┴─────┐
     │  Git    │ │  MCP   │ │Sandbox│ │  MCP   │
     │ Service │ │ Server │ │System │ │ Server │
     └─────────┘ └────────┘ └───────┘ └─────────┘
           │          │          │          │
     ┌─────┴──────────┴──────────┴──────────┴─────┐
     │              Data Layer                      │
     │  PostgreSQL · Redis · Kafka · MinIO · ES    │
     └─────────────────────────────────────────────┘
```

## Project Structure

```
omnidev-ai-platform/
├── apps/                    # Application services
│   ├── web/                 # Next.js frontend
│   ├── gateway/             # API Gateway (Go/Gin)
│   ├── user/                # User Service (Go)
│   ├── project/             # Project Service (Go)
│   ├── workspace/           # Workspace Service (Go)
│   ├── terminal/            # Terminal Service (Go)
│   ├── file/                # File Service (Go)
│   ├── git/                 # Git Service (Go)
│   ├── chat/                # Chat Service (Go)
│   ├── agent/               # Agent Service (Go)
│   ├── rag/                 # RAG Service (Go)
│   ├── workflow/            # Workflow Service (Go)
│   ├── sandbox/             # Sandbox Service (Go)
│   ├── mcp/                 # MCP Service (Go)
│   ├── billing/             # Billing Service (Go)
│   ├── deploy/              # Deploy Service (Go)
│   └── monitor/             # Monitor Service (Go)
├── packages/                # Shared libraries
│   ├── ui/                  # UI component library
│   ├── utils/               # Common utilities
│   ├── proto/               # gRPC proto definitions
│   └── types/               # Shared TypeScript types
├── tools/                   # Dev tools & scripts
├── deploy/                  # Deployment configs
├── docs/                    # Documentation
└── web-extensions/          # Browser extensions
```

## Quickstart

### Prerequisites

- Go 1.22+
- Node.js 20+
- Docker & Docker Compose
- PostgreSQL 16
- Redis 7

### Run Locally

```bash
# Clone the repository
git clone https://github.com/your-org/omnidev-ai-platform.git
cd omnidev-ai-platform

# Start infrastructure
docker-compose up -d postgres redis kafka minio elasticsearch

# Run database migrations
cd tools/migrations && make migrate-up

# Start backend services
cd apps/gateway && go run cmd/main.go

# Start frontend
cd apps/web && pnpm install && pnpm dev
```

### Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Key variables:
- `DATABASE_URL` — PostgreSQL connection string
- `REDIS_URL` — Redis connection string
- `JWT_SECRET` — JWT signing secret
- `OPENAI_API_KEY` — OpenAI API key (or other provider)

## Competitive Landscape

| Capability | Competitors | OmniDev Differentiator |
|-----------|-------------|----------------------|
| AI Coding | Cursor, GitHub Copilot | Multi-model + local models + RAG |
| Online IDE | Codespaces, Gitpod | Embedded AI Agent + one-click deploy |
| Agent Platform | OpenHands, AutoGPT | Visual orchestration + sandbox + marketplace |
| RAG / Knowledge | ChatGPT Projects, Notion AI | Multi-format + hybrid search + code index |
| CI/CD | Jenkins, GitHub Actions | Natural language config + K8s native |
| Deployment | Vercel, Railway | Multi-cloud + K8s + cost optimization |
| Monitoring | Grafana Cloud | Built-in OTel + AI anomaly detection |

## Target Users

| Persona | Use Case |
|---------|----------|
| Solo Developer | Rapid prototype → AI-assisted coding → one-click deploy |
| Small Team (2-10) | Collaborative dev + Agent automation + shared knowledge base |
| Medium/Large Team (10-100) | RBAC + audit + compliance + private deployment |
| AI App Developer | Build/debug/deploy Agents and MCP Servers |

## Development Roadmap

| Milestone | Timeline | Deliverables |
|-----------|----------|--------------|
| **M0 — Foundation** | Month 1-2 | Infrastructure, DB, auth, CI/CD |
| **M1 — Alpha** | Month 3-4 | IDE (Monaco + terminal) + Agent (basic execution + tool calling + sandbox) |
| **M2 — Beta** | Month 5-6 | IDE (Git + code review) + Agent (frontend + monitoring) |
| **M3 — Ecosystem** | Month 7-9 | Agent visual orchestration + marketplace + multi-agent |
| **M4 — GA** | Month 10-12 | Performance + security hardening + plugin system |

## Documentation

| Document | Description |
|----------|-------------|
| [Executive Summary](docs/architecture/00-EXECUTIVE-SUMMARY.md) | Project vision & positioning |
| [Requirements Analysis](docs/architecture/01-REQUIREMENTS-ANALYSIS.md) | Functional & non-functional requirements |
| [Feature Boundary](docs/architecture/02-FEATURE-BOUNDARY.md) | Module boundaries & dependencies |
| [Non-Functional Requirements](docs/architecture/03-NON-FUNCTIONAL-REQUIREMENTS.md) | Performance, security, availability |
| [Technology Selection](docs/architecture/04-TECHNOLOGY-SELECTION.md) | Tech stack comparison & rationale |
| [System Architecture](docs/architecture/05-SYSTEM-ARCHITECTURE.md) | Architecture diagrams (Mermaid) |
| [Database Design](docs/architecture/06-DATABASE-DESIGN.md) | ER diagrams & schema |
| [Directory Structure](docs/architecture/07-DIRECTORY-STRUCTURE.md) | Project layout |
| [Development Standards](docs/architecture/08-DEVELOPMENT-STANDARDS.md) | Coding standards & workflow |
| [Milestone Plan](docs/architecture/09-MILESTONE-PLAN.md) | Detailed milestone schedule |

## License

[MIT](LICENSE)
