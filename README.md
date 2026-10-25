<h1 align="center">OmniDev</h1>

<p align="center">
  <strong>AI knowledge &amp; coding workbench for teams</strong>
</p>

<p align="center">
  Multi-model chat, searchable knowledge bases, and background agents that run code in Docker sandboxes — one Go binary, one Next.js app.
</p>

<p align="center">
  <a href="README_zh.md">中文文档</a> · <a href="#quickstart">Quickstart</a> · <a href="#architecture">Architecture</a> · <a href="docs/architecture/">Docs</a>
</p>

---

## What is OmniDev?

OmniDev is a self-hostable workbench built around three things that work today:

- **Chat** — multi-model conversations (OpenAI, DeepSeek, Qwen, Anthropic) with streaming, file attachments, image input/output, per-user provider keys, and external MCP tools.
- **Knowledge** — upload documents, chunk and embed them, search with pgvector, and inject the results into chat as cited context. Powered by the embedded [knowledge-engine](../knowledge-engine) module, which also runs standalone as an HTTP API and MCP server.
- **Background agents** — describe a task, and an agent plans and executes it asynchronously with workspace file tools, search, a calculator, and code execution inside isolated Docker containers. Runs are persisted, pollable, and cancellable.

Everything is served by a **modular monolith**: `apps/gateway` contains auth, users, chat, knowledge, and agent. There are no microservices to orchestrate.

### Explicitly not included

These were dropped because they were mocked, replaceable by integrations, or not worth the operational cost: online IDE (Monaco + fake terminal), visual drag-and-drop workflow canvas, self-built billing/subscriptions, the Prometheus/Grafana/Loki/Jaeger monitoring stack, one-click deployment, notification service, Kafka, Elasticsearch, Temporal, and the 16-microservice split. Agent **PR delivery is not implemented yet**.

## Feature status

| Area | Status | Notes |
|------|--------|-------|
| Auth (JWT, refresh, API keys, OAuth hooks) | Working | Per-user ownership checks on all resources |
| Chat (streaming + non-streaming) | Working | OpenAI / DeepSeek / Qwen / Anthropic adapters |
| Attachments & multimodal | Working | MinIO storage, Tika parsing, image generation |
| Per-user AI provider configs (BYOK) | Working | Optional `ENCRYPTION_KEY` for stored keys |
| Knowledge bases + RAG in chat | Working | Tika parsing, per-KB chunking, pgvector search, Redis cache |
| External MCP tools in chat | Working | OpenAI-compatible adapters; Anthropic tool calling not mapped yet |
| Background agents + Docker sandbox | Working | Async runs, step records, cancel; requires Docker |
| Agent PR delivery | Planned | No git/PR integration exists |
| Online IDE, visual workflows, billing, monitoring | Removed | Non-goals, see above |

## Architecture

```
apps/web (Next.js 15, React 19)
        │ HTTP / SSE
        ▼
apps/gateway (Go 1.22, Gin) — one binary
  ├─ auth / users / API keys
  ├─ chat ─────────────┬─ knowledge-engine (embedded Go module)
  │                    └─ MCP client → external MCP servers
  └─ agent: planner → executor → tools → Docker sandbox

packages/go-common — config, auth, cache, storage, parser, middleware

PostgreSQL 16 + pgvector · Redis 7 · MinIO · Apache Tika · Docker (agent sandbox)
```

```
omnidev/
├── apps/
│   ├── gateway/            # API: auth + chat + knowledge + agent
│   │   ├── cmd/gateway/    # entrypoint + knowledge wiring
│   │   ├── internal/       # adapter, agent, handler, knowledge, mcp, repository, router, service
│   │   └── migrations/     # 001–009, single database schema
│   └── web/                # Next.js frontend (chat, knowledge, agents, settings)
├── packages/go-common/     # shared Go packages
├── deploy/docker/          # compose: infra + gateway + nginx
├── docs/                   # architecture and feature docs
└── Makefile

../knowledge-engine/        # sibling repo: embeddable RAG engine + standalone MCP/HTTP server
```

## Quickstart

Prerequisites: Go 1.22+, Node.js 20+, pnpm 9, Docker.

Clone **both** repositories side by side — the gateway imports `knowledge-engine` through a local `replace` directive:

```bash
git clone git@github.com:m2lan/omnidev.git
git clone git@github.com:m2lan/knowledge-engine.git
cd omnidev
cp .env.example .env   # add your provider keys
```

### Option A: run the gateway on the host

```bash
make dev-infra      # PostgreSQL, Redis, MinIO, Tika + migrations
make run-gateway    # API on :9090
make web-dev        # web UI on :3000
```

### Option B: run everything in Docker

```bash
make dev            # infra + gateway + nginx (builds the gateway image)
```

The gateway container mounts `/var/run/docker.sock` so agents can start sandbox containers. Set `SANDBOX_HOST_WORKSPACE_ROOT` to the host path of the agent workspace directory (default `/tmp/omnidev-agent-workspaces`) and `DOCKER_GID` on Linux if the socket is group-restricted.

### Agent sandbox

```bash
make sandbox-images   # pre-pull python/node/go/shell runtime images
```

Each run gets its own workspace mounted at `/workspace`. Containers have no network, a read-only root filesystem, `tmpfs /tmp`, dropped capabilities, `no-new-privileges`, CPU/memory/PID limits, and a configurable timeout (`SANDBOX_TIMEOUT`, default 120s). Supported languages: Python, JavaScript/Node, Go, shell.

## Configuration

All configuration is environment-driven; see [`.env.example`](.env.example) for the full list.

| Area | Variables |
|------|-----------|
| Server & DB | `APP_*`, `DB_*`, `REDIS_*` |
| Auth | `JWT_SECRET`, `JWT_ACCESS_EXPIRY`, `JWT_REFRESH_EXPIRY`, `ENCRYPTION_KEY` |
| Models | `OPENAI_API_KEY`, `DEEPSEEK_API_KEY`, `QWEN_API_KEY`, `ANTHROPIC_API_KEY`, `AI_DEFAULT_MODEL` |
| Files | `MINIO_*`, `TIKA_ENDPOINT` |
| Knowledge | `RAG_EMBEDDING_PROVIDER/MODEL/API_KEY/BASE_URL`, `KNOWLEDGE_*` |
| MCP | `MCP_ENABLED`, `MCP_SERVERS` (JSON array of `{name, url, api_key, timeout_seconds, headers}`) |
| Agent | `AGENT_MODEL`, `AGENT_BASE_URL`, `AGENT_API_KEY`, `SANDBOX_*` |

Embedding dimensions follow the embedding model per knowledge base (`KNOWLEDGE_EMBEDDING_DIMENSIONS=0` uses the provider default), so 768-dim and 1536-dim models can coexist.

Example MCP configuration:

```env
MCP_ENABLED=true
MCP_SERVERS=[{"name":"knowledge","url":"http://localhost:9100/mcp","timeout_seconds":15}]
```

Chat executes at most 5 tool rounds per message and truncates tool output to 8000 runes. Tool calls are recorded in the assistant message metadata as `tools_used`. MCP tools are currently offered to OpenAI-compatible adapters (OpenAI, DeepSeek, Qwen); Anthropic `tool_use` mapping is not implemented yet.

## Agent API

All routes require a JWT and are registered only when `SANDBOX_ENABLED=true`.

| Method | Path | Purpose |
|--------|------|---------|
| GET / POST | `/api/v1/agents` | List / create agents |
| GET / PATCH / DELETE | `/api/v1/agents/:id` | Read / update / delete an agent |
| POST | `/api/v1/agents/:id/run` | Start an async run (`{"task": "..."}`) |
| GET | `/api/v1/agents/:id/runs` | List runs |
| GET | `/api/v1/agent-runs/:run_id` | Run detail with steps |
| POST | `/api/v1/agent-runs/:run_id/cancel` | Cancel a running run |

## Development

```bash
go build ./apps/gateway/... ./packages/go-common/...
go test  ./apps/gateway/... ./packages/go-common/...
go vet   ./apps/gateway/... ./packages/go-common/...
pnpm --filter web typecheck

make db-migrate                 # apply migrations to a local database
make db-migrate-create NAME=x   # new migration in apps/gateway/migrations
```

## Documentation

| Document | Description |
|----------|-------------|
| [Executive Summary](docs/architecture/00-EXECUTIVE-SUMMARY.md) | Positioning and scope |
| [Requirements](docs/architecture/01-REQUIREMENTS-ANALYSIS.md) | What the product does and does not do |
| [Feature Boundary](docs/architecture/02-FEATURE-BOUNDARY.md) | Module boundaries |
| [Non-Functional Requirements](docs/architecture/03-NON-FUNCTIONAL-REQUIREMENTS.md) | Security, isolation, limits |
| [Technology Selection](docs/architecture/04-TECHNOLOGY-SELECTION.md) | Stack and removed dependencies |
| [System Architecture](docs/architecture/05-SYSTEM-ARCHITECTURE.md) | Request flows and runtime topology |
| [Database Design](docs/architecture/06-DATABASE-DESIGN.md) | Real schema |
| [Directory Structure](docs/architecture/07-DIRECTORY-STRUCTURE.md) | Project layout |
| [Development Standards](docs/architecture/08-DEVELOPMENT-STANDARDS.md) | Conventions and workflow |
| [Milestone Plan](docs/architecture/09-MILESTONE-PLAN.md) | Done / planned / cancelled |
| [File Upload & Multimodal](docs/features/file-upload-and-multimodal.md) | Attachments and parsing |

## License

[MIT](LICENSE)
