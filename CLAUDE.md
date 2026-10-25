# OmniDev — Development Guide

## What this is

OmniDev is an AI knowledge & coding workbench: multi-model Chat, knowledge bases with RAG, and background agents that execute code in Docker sandboxes.

**The backend is a modular monolith.** Do not reintroduce per-feature microservices, Kafka, Elasticsearch, Temporal, or a monitoring stack. Removed non-goals: online IDE, visual workflow canvas, self-built billing, one-click deploy, notification service.

## Layout

- `apps/gateway` — the only backend binary (auth, users, chat, knowledge, agent)
  - `cmd/gateway` — entrypoint, dependency wiring
  - `internal/adapter` — model providers (OpenAI, DeepSeek, Qwen, Anthropic)
  - `internal/agent` — agent domain, planner, executor, Docker sandbox, tools, HTTP handlers
  - `internal/mcp` — outbound MCP client (streamable HTTP) used by chat
  - `internal/service`, `internal/handler`, `internal/repository`, `internal/router`
  - `migrations` — single PostgreSQL schema, sequential 001…00N
- `apps/web` — Next.js 15 / React 19 frontend
- `packages/go-common` — config, auth, cache, storage, parser, middleware
- `../knowledge-engine` — sibling repo imported via `replace`; also a standalone MCP/HTTP server

## Commands

```bash
go build ./apps/gateway/... ./packages/go-common/...
go test  ./apps/gateway/... ./packages/go-common/...
go vet   ./apps/gateway/... ./packages/go-common/...
pnpm --filter web typecheck

make dev-infra     # Postgres/Redis/MinIO/Tika + migrations
make run-gateway   # API :9090
make web-dev       # UI :3000
make db-migrate-create NAME=add_xxx
```

## Rules

- Keep features in the gateway unless there is a measured reason to split; shared code goes to `packages/go-common` or `knowledge-engine`.
- Every resource query must enforce the owning user; agent workspaces must stay inside the configured sandbox root.
- Never execute agent code on the host: it goes through `agent/executor.SandboxManager` (Docker, no network, read-only root, resource limits).
- Database changes = new sequential migration in `apps/gateway/migrations` (up + down).
- Complete code only: no `...`, `TODO`, or placeholder implementations. If something is not implemented, say so in docs instead of shipping a mock.
- Go: Effective Go + golangci-lint. TypeScript: strict mode, ESLint + Prettier. Commits: `<type>(<scope>): <description>`.

### Documentation sync

When config or features change, update together:
1. `.env.example` — new variables with comments
2. `README.md` and `README_zh.md` — keep them consistent
3. `docs/architecture/09-MILESTONE-PLAN.md` — status must match reality (done / planned / cancelled)

Architecture references live in `docs/architecture/` (00 summary … 09 milestones).
