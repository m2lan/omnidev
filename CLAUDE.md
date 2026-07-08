# OmniDev AI Platform — Development Standards

## Overview
OmniDev AI Platform: All-in-One AI development platform integrating IDE, Agent, RAG, Workflow, Deploy, and more.

## Tech Stack
- Backend: Go 1.22+ / Gin / gRPC / Wire
- Frontend: Next.js 15 / React 19 / Tailwind / Shadcn/ui
- Database: PostgreSQL 16 + pgvector / Redis 7 / Kafka (KRaft)
- Storage: MinIO (S3-compatible)
- Search: Elasticsearch 8
- Workflow: Temporal
- Infra: Docker / Kubernetes / Helm / Terraform
- Observability: Prometheus / Grafana / Loki / Jaeger / OpenTelemetry

## Development Rules

### Implementation Cadence
- One module at a time, production-grade quality required
- Follow milestone order in architecture docs (M0 → M1 → M2 → M3 → M4)

### Quality Requirements (every module must include)
1. **Complete code** — no `...`, `TODO`, or placeholders
2. **Unit tests** — coverage > 80%
3. **Integration tests** — core flows covered
4. **API docs** — OpenAPI 3.0 spec
5. **DB migration scripts** — up + down
6. **README** — module description, startup, config
7. **Deployment guide** — Docker / K8s steps
8. **Performance notes** — key path optimizations
9. **Security design** — auth, authorization, input validation, log redaction
10. **Troubleshooting** — common issues and fixes
11. **Follow-ups** — incomplete items for the module

### Output Rules
- Output to model max length, no compression or omission
- No `...`, `// omitted`, `/* similar code */` placeholders
- When hitting output limit, end with `Continued: Module X, Part N`
- Wait for user to send `continue` before resuming

### Code Style
- Go: follow Effective Go, golangci-lint standards
- TypeScript: ESLint + Prettier, strict mode
- Commit: `<type>(<scope>): <description>` format
- Branch: feature/{module}-{desc} → develop → main

### Directory Structure
See `docs/architecture/07-DIRECTORY-STRUCTURE.md`

### Documentation Sync Rule

**When adding new config or features, update all three files:**

1. **`.env.example`** — add new env vars with comments
2. **`README.md`** — English docs, config descriptions and examples
3. **`README_zh.md`** — Chinese docs, keep consistent with English

Pre-commit checklist:
- [ ] `.env.example` has the new variables?
- [ ] `README.md` env section updated?
- [ ] `README_zh.md` env section updated?
- [ ] Config examples consistent across both READMEs?

Rule: config change = all three files, no exceptions.

### Architecture References
All architecture docs are in `docs/architecture/`:
- 00-EXECUTIVE-SUMMARY.md — Project vision
- 01-REQUIREMENTS-ANALYSIS.md — Requirements
- 02-FEATURE-BOUNDARY.md — Feature boundaries
- 03-NON-FUNCTIONAL-REQUIREMENTS.md — Non-functional requirements
- 04-TECHNOLOGY-SELECTION.md — Tech selection
- 05-SYSTEM-ARCHITECTURE.md — System architecture
- 06-DATABASE-DESIGN.md — Database design
- 07-DIRECTORY-STRUCTURE.md — Directory structure
- 08-DEVELOPMENT-STANDARDS.md — Development standards
- 09-MILESTONE-PLAN.md — Milestone plan
