<h1 align="center">OmniDev</h1>

<p align="center">
  <strong>面向团队的 AI 知识与编码工作台</strong>
</p>

<p align="center">
  多模型 Chat、可检索的知识库、在 Docker 沙箱里跑代码的后台 Agent —— 一个 Go 二进制 + 一个 Next.js 应用。
</p>

<p align="center">
  <a href="README.md">English</a> · <a href="#快速开始">快速开始</a> · <a href="#架构">架构</a> · <a href="docs/architecture/">文档</a>
</p>

---

## OmniDev 是什么？

OmniDev 是一个可私有部署的工作台，目前聚焦三件已经能真实使用的事：

- **Chat** —— 多模型对话（OpenAI、DeepSeek、Qwen、Anthropic），支持流式输出、文件附件、图片输入与生成、用户自带的模型 Key（BYOK），以及在对话中调用外部 MCP 工具。
- **知识库** —— 上传文档、分块、向量化，用 pgvector 检索，并把结果作为带引用的上下文注入对话。由内置的 [knowledge-engine](../knowledge-engine) 模块提供；该模块也能独立部署为 HTTP API 和 MCP server。
- **后台 Agent** —— 用自然语言描述任务，Agent 异步规划并执行，可读写运行工作区、搜索、计算、在隔离的 Docker 容器里执行代码。运行记录落库，可轮询进度、可取消。

后端是**模块化单体**：`apps/gateway` 一个进程承载认证、用户、Chat、知识库、Agent，不需要编排任何微服务。

### 明确不做 / 已下线

以下能力因为此前是演示桩、可以被集成替代、或运维成本过高而移除：在线 IDE（Monaco + 假终端）、可视化拖拽工作流画布、自建支付与订阅、Prometheus/Grafana/Loki/Jaeger 监控全家桶、一键部署、通知微服务，以及 Kafka、Elasticsearch、Temporal 和 16 微服务拆分。**Agent 自动提交 PR 尚未实现。**

## 功能现状

| 模块 | 状态 | 说明 |
|------|------|------|
| 认证（JWT、刷新令牌、API Key、OAuth 接入点） | 可用 | 所有资源按用户隔离 |
| Chat（流式 / 非流式） | 可用 | OpenAI / DeepSeek / Qwen / Anthropic 适配器 |
| 附件与多模态 | 可用 | MinIO 存储、Tika 解析、图片生成 |
| 用户自定义模型配置（BYOK） | 可用 | 配置 `ENCRYPTION_KEY` 后密钥加密存储 |
| 知识库 + 对话内 RAG | 可用 | Tika 解析、按知识库分块、pgvector 检索、Redis 缓存 |
| Chat 调用外部 MCP 工具 | 可用 | 目前支持 OpenAI 兼容适配器；Anthropic 工具调用尚未接入 |
| 后台 Agent + Docker 沙箱 | 可用 | 异步运行、步骤落库、可取消；需要 Docker |
| Agent 自动交付 PR | 规划中 | 尚无 git/PR 集成 |
| 在线 IDE、可视化工作流、支付、监控栈 | 已移除 | 见上文非目标 |

## 架构

```
apps/web（Next.js 15，React 19）
        │ HTTP / SSE
        ▼
apps/gateway（Go 1.22，Gin）—— 单二进制
  ├─ 认证 / 用户 / API Key
  ├─ chat ─────────────┬─ knowledge-engine（内嵌 Go 模块）
  │                    └─ MCP client → 外部 MCP server
  └─ agent：planner → executor → tools → Docker 沙箱

packages/go-common —— 配置、认证、缓存、存储、解析、中间件

PostgreSQL 16 + pgvector · Redis 7 · MinIO · Apache Tika · Docker（Agent 沙箱）
```

```
omnidev/
├── apps/
│   ├── gateway/            # API：认证 + Chat + 知识库 + Agent
│   │   ├── cmd/gateway/    # 入口与知识引擎装配
│   │   ├── internal/       # adapter、agent、handler、knowledge、mcp、repository、router、service
│   │   └── migrations/     # 001–009，单库 schema
│   └── web/                # Next.js 前端（Chat、知识库、Agent、设置）
├── packages/go-common/     # 共享 Go 包
├── deploy/docker/          # compose：基础设施 + gateway + nginx
├── docs/                   # 架构与功能文档
└── Makefile

../knowledge-engine/        # 兄弟仓库：可嵌入的 RAG 引擎 + 独立 MCP/HTTP 服务
```

## 快速开始

依赖：Go 1.22+、Node.js 20+、pnpm 9、Docker。

两个仓库需要**并列克隆**，gateway 通过本地 `replace` 引用 `knowledge-engine`：

```bash
git clone git@github.com:m2lan/omnidev.git
git clone git@github.com:m2lan/knowledge-engine.git
cd omnidev
cp .env.example .env   # 填入你的模型 Key
```

### 方式 A：在宿主机运行 gateway

```bash
make dev-infra      # PostgreSQL、Redis、MinIO、Tika，并自动执行迁移
make run-gateway    # API：:9090
make web-dev        # 前端：:3000
```

### 方式 B：全部用 Docker 运行

```bash
make dev            # 基础设施 + gateway + nginx（自动构建镜像）
```

gateway 容器挂载 `/var/run/docker.sock`，用于启动 Agent 沙箱容器。请把 `SANDBOX_HOST_WORKSPACE_ROOT` 设为工作区目录在**宿主机**上的路径（默认 `/tmp/omnidev-agent-workspaces`）；Linux 下如 Docker socket 有用户组限制，还需设置 `DOCKER_GID`。

### Agent 沙箱

```bash
make sandbox-images   # 预拉取 python/node/go/shell 运行镜像
```

每次运行都有独立工作区，挂载到容器 `/workspace`。容器默认无网络、根文件系统只读、`/tmp` 为 tmpfs、丢弃全部 capabilities、启用 `no-new-privileges`，并限制 CPU / 内存 / PID 数与超时时间（`SANDBOX_TIMEOUT`，默认 120 秒）。支持 Python、JavaScript/Node、Go、shell。

## 配置

全部通过环境变量配置，完整清单见 [`.env.example`](.env.example)。

| 范围 | 变量 |
|------|------|
| 服务与数据库 | `APP_*`、`DB_*`、`REDIS_*` |
| 认证 | `JWT_SECRET`、`JWT_ACCESS_EXPIRY`、`JWT_REFRESH_EXPIRY`、`ENCRYPTION_KEY` |
| 模型 | `OPENAI_API_KEY`、`DEEPSEEK_API_KEY`、`QWEN_API_KEY`、`ANTHROPIC_API_KEY`、`AI_DEFAULT_MODEL` |
| 文件 | `MINIO_*`、`TIKA_ENDPOINT` |
| 知识库 | `RAG_EMBEDDING_PROVIDER/MODEL/API_KEY/BASE_URL`、`KNOWLEDGE_*` |
| MCP | `MCP_ENABLED`、`MCP_SERVERS`（JSON 数组，元素为 `{name, url, api_key, timeout_seconds, headers}`） |
| Agent | `AGENT_MODEL`、`AGENT_BASE_URL`、`AGENT_API_KEY`、`SANDBOX_*` |

向量维度按知识库跟随 embedding 模型（`KNOWLEDGE_EMBEDDING_DIMENSIONS=0` 表示使用 Provider 默认值），768 维与 1536 维模型可以共存。

MCP 配置示例：

```env
MCP_ENABLED=true
MCP_SERVERS=[{"name":"knowledge","url":"http://localhost:9100/mcp","timeout_seconds":15}]
```

Chat 每条消息最多执行 5 轮工具调用，工具输出截断到 8000 字符；调用过的工具记录在助手消息的 `tools_used` 元数据中。当前 MCP 工具仅提供给 OpenAI 兼容适配器（OpenAI、DeepSeek、Qwen），Anthropic 的 `tool_use` 映射尚未实现。

## Agent API

以下路由都需要 JWT，且仅在 `SANDBOX_ENABLED=true` 时注册。

| 方法 | 路径 | 作用 |
|------|------|------|
| GET / POST | `/api/v1/agents` | 列出 / 创建 Agent |
| GET / PATCH / DELETE | `/api/v1/agents/:id` | 查看 / 更新 / 删除 Agent |
| POST | `/api/v1/agents/:id/run` | 启动异步运行（`{"task": "..."}`） |
| GET | `/api/v1/agents/:id/runs` | 运行列表 |
| GET | `/api/v1/agent-runs/:run_id` | 运行详情（含步骤） |
| POST | `/api/v1/agent-runs/:run_id/cancel` | 取消运行 |

## 开发

```bash
go build ./apps/gateway/... ./packages/go-common/...
go test  ./apps/gateway/... ./packages/go-common/...
go vet   ./apps/gateway/... ./packages/go-common/...
pnpm --filter web typecheck

make db-migrate                 # 对本地数据库执行迁移
make db-migrate-create NAME=x   # 在 apps/gateway/migrations 新建迁移
```

## 文档

| 文档 | 内容 |
|------|------|
| [执行摘要](docs/architecture/00-EXECUTIVE-SUMMARY.md) | 定位与范围 |
| [需求分析](docs/architecture/01-REQUIREMENTS-ANALYSIS.md) | 做什么、不做什么 |
| [功能边界](docs/architecture/02-FEATURE-BOUNDARY.md) | 模块边界 |
| [非功能需求](docs/architecture/03-NON-FUNCTIONAL-REQUIREMENTS.md) | 安全、隔离、限额 |
| [技术选型](docs/architecture/04-TECHNOLOGY-SELECTION.md) | 技术栈与移除的依赖 |
| [系统架构](docs/architecture/05-SYSTEM-ARCHITECTURE.md) | 请求链路与运行拓扑 |
| [数据库设计](docs/architecture/06-DATABASE-DESIGN.md) | 真实表结构 |
| [目录结构](docs/architecture/07-DIRECTORY-STRUCTURE.md) | 项目布局 |
| [开发规范](docs/architecture/08-DEVELOPMENT-STANDARDS.md) | 约定与流程 |
| [里程碑计划](docs/architecture/09-MILESTONE-PLAN.md) | 已完成 / 规划中 / 已取消 |
| [文件上传与多模态](docs/features/file-upload-and-multimodal.md) | 附件与解析 |

## 许可证

[MIT](LICENSE)
