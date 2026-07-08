<p align="center">
  <img src="docs/assets/omnidev-logo.svg" alt="OmniDev AI Platform" width="120" height="120">
</p>

<h1 align="center">OmniDev AI Platform</h1>

<p align="center">
  <strong>一站式 AI 开发平台</strong>
</p>

<p align="center">
  打开浏览器，用 AI 写代码、管项目、部署上线，一个平台全搞定。
</p>

<p align="center">
  <a href="README.md">English</a> · <a href="#-快速开始">快速开始</a> · <a href="#-系统架构">系统架构</a> · <a href="docs/architecture/">文档</a>
</p>

---

## 项目简介

OmniDev AI Platform 是一个 **All-in-One AI 开发平台**，将 AI 编程助手、在线 IDE、Agent 编排、RAG 知识库、CI/CD 部署、监控运维融合为统一产品。

**一句话定位：** 打开浏览器，就能用 AI 写代码、管项目、部署上线。

## 核心功能

### 在线 IDE

| 功能 | 说明 |
|------|------|
| Monaco Editor | 语法高亮、多光标、Minimap、文件搜索与替换 |
| AI 代码补全 | Tab 补全、多行建议、上下文感知（通过 LSP 协议对接 Chat Engine） |
| Agent 代码修改 | AI Agent 直接修改代码，支持 Diff 预览、接受/拒绝/撤销 |
| 终端模拟 | xterm.js、多 Tab、SSH、PTY 支持 |
| Git 操作 | commit/push/pull/branch/merge、可视化 Diff |
| Code Review | 行内评论、AI 辅助审查、建议修改 |

### Agent 系统

| 功能 | 说明 |
|------|------|
| 自然语言任务执行 | 用户用自然语言描述任务，Agent 自动分解、规划、执行 |
| Tool Calling | 内置工具 + 自定义 MCP 工具，支持 Function Calling |
| 可视化编排 | 拖拽画布、节点连线、条件分支 |
| 沙箱代码执行 | Docker 隔离、资源限制、文件系统挂载 |
| 执行监控 | 实时日志、步骤追踪、中间状态查看 |
| Agent 模板 | 预置模板：代码审查、测试生成、文档生成等 |
| 多 Agent 协作 | CrewAI 模式、角色分配、任务委派 |
| Agent 市场 | 配置导出、评分、安装统计 |

### RAG 知识库

| 功能 | 说明 |
|------|------|
| 多格式上传 | PDF、Markdown、代码文件等 |
| 混合搜索 | 向量搜索 (pgvector) + 关键词搜索 (Elasticsearch) |
| 代码索引 | 代码感知的分块与索引 |
| 上下文注入 | 与 Chat 和 Agent 集成，自动注入知识库上下文 |

### CI/CD 与部署

| 功能 | 说明 |
|------|------|
| 自然语言配置 | 用自然语言描述流水线，自动生成配置 |
| K8s 原生部署 | Kubernetes 原生，支持弹性伸缩 |
| 多云支持 | AWS / GCP / Azure 统一部署 |
| Git 触发 | 代码推送自动触发构建与部署 |

### 监控与可观测性

| 功能 | 说明 |
|------|------|
| OpenTelemetry | 内置 OTel SDK，全链路追踪 |
| 指标/日志/链路 | Prometheus + Grafana + Loki + Jaeger 全栈 |
| AI 异常检测 | 基于 AI 的异常自动识别与告警 |
| 自定义告警 | 灵活的告警规则配置 |

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.22+ / Gin / gRPC / Wire |
| 前端 | Next.js 15 (App Router) / React 19 / Tailwind CSS / Shadcn/ui |
| 数据库 | PostgreSQL 16 + pgvector / Redis 7 / Kafka (KRaft) |
| 对象存储 | MinIO (S3 兼容) |
| 搜索引擎 | Elasticsearch 8 |
| 工作流引擎 | Temporal |
| 基础设施 | Docker / Kubernetes / Helm / Terraform |
| 可观测性 | Prometheus / Grafana / Loki / Jaeger / OpenTelemetry |

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Gateway (Kong/自研)                       │
│              限流 · 认证 · 路由 · WAF                        │
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
     │                 数据层                       │
     │  PostgreSQL · Redis · Kafka · MinIO · ES    │
     └─────────────────────────────────────────────┘
```

**Agent 执行状态机：**

```
Created → Planning → Executing → [ToolCall → Waiting → Executing]
                                      ↓
                                 Success / Failed / Cancelled
```

## 项目结构

```
omnidev-ai-platform/
├── apps/                    # 应用服务
│   ├── web/                 # Next.js 前端
│   ├── gateway/             # API 网关 (Go/Gin)
│   ├── user/                # 用户服务
│   ├── project/             # 项目服务
│   ├── workspace/           # 工作区服务
│   ├── terminal/            # 终端服务
│   ├── file/                # 文件服务
│   ├── git/                 # Git 服务
│   ├── chat/                # Chat 服务
│   ├── agent/               # Agent 服务
│   ├── rag/                 # RAG 服务
│   ├── workflow/            # 工作流服务
│   ├── sandbox/             # 沙箱服务
│   ├── mcp/                 # MCP 服务
│   ├── billing/             # 计费服务
│   ├── deploy/              # 部署服务
│   └── monitor/             # 监控服务
├── packages/                # 共享库
│   ├── ui/                  # UI 组件库
│   ├── utils/               # 通用工具
│   ├── proto/               # gRPC Proto 定义
│   └── types/               # 共享 TypeScript 类型
├── tools/                   # 开发工具与脚本
├── deploy/                  # 部署配置
├── docs/                    # 文档
└── web-extensions/          # 浏览器扩展
```

## 快速开始

### 环境要求

- Go 1.22+
- Node.js 20+
- Docker & Docker Compose
- PostgreSQL 16
- Redis 7

### 本地运行

```bash
# 克隆仓库
git clone https://github.com/your-org/omnidev-ai-platform.git
cd omnidev-ai-platform

# 启动基础设施
docker-compose up -d postgres redis kafka minio elasticsearch

# 执行数据库迁移
cd tools/migrations && make migrate-up

# 启动后端服务
cd apps/gateway && go run cmd/main.go

# 启动前端
cd apps/web && pnpm install && pnpm dev
```

### 环境变量

复制 `.env.example` 到 `.env` 并配置：

```bash
cp .env.example .env
```

关键变量：
- `DATABASE_URL` — PostgreSQL 连接字符串
- `REDIS_URL` — Redis 连接字符串
- `JWT_SECRET` — JWT 签名密钥
- `OPENAI_API_KEY` — OpenAI API Key（或其他 AI 提供商）

#### RAG 向量模型配置

配置 RAG 知识库的向量模型提供商。采用 provider 级别配置，不再通过模型名猜测：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `RAG_EMBEDDING_PROVIDER` | 提供商名称：`openai` / `gemini` / `ollama` | `openai` |
| `RAG_EMBEDDING_MODEL` | 模型标识符（各提供商不同） | `text-embedding-3-small` |
| `RAG_EMBEDDING_API_KEY` | API Key 覆盖（可选，不填则用对应提供商的 key） | — |
| `RAG_EMBEDDING_BASE_URL` | 自部署服务地址（Ollama、vLLM 等） | — |

配置示例：

```env
# OpenAI（默认）
RAG_EMBEDDING_PROVIDER=openai
RAG_EMBEDDING_MODEL=text-embedding-3-small

# Gemini
RAG_EMBEDDING_PROVIDER=gemini
RAG_EMBEDDING_MODEL=gemini-embedding-2

# Ollama（本地部署）
RAG_EMBEDDING_PROVIDER=ollama
RAG_EMBEDDING_MODEL=nomic-embed-text
RAG_EMBEDDING_BASE_URL=http://localhost:11434/v1

# 自部署兼容 OpenAI 接口（vLLM、text-embedding-inference 等）
RAG_EMBEDDING_PROVIDER=openai
RAG_EMBEDDING_MODEL=BAAI/bge-large-zh-v1.5
RAG_EMBEDDING_BASE_URL=http://your-server:8080/v1
```

扩展新向量模型提供商：在 `embedder/registry.go` 中注册 factory 即可，无需修改 main.go。

## 竞品对比

| 能力域 | 竞品 | OmniDev 差异化 |
|--------|------|----------------|
| AI 编程助手 | Cursor, GitHub Copilot | 多模型切换 + 本地模型 + RAG 增强 |
| 在线 IDE | GitHub Codespaces, Gitpod | 内嵌 AI Agent + 一键部署 |
| Agent 平台 | OpenHands, AutoGPT | 可视化编排 + 沙箱执行 + 插件市场 |
| RAG / 知识库 | ChatGPT Projects, Notion AI | 多格式上传 + 混合搜索 + 代码索引 |
| CI/CD | Jenkins, GitHub Actions | 自然语言配置 + K8s 原生 |
| 部署托管 | Vercel, Railway | 多云 + K8s + 成本优化 |
| 监控 | Grafana Cloud | 内置 OpenTelemetry + AI 异常检测 |

## 目标用户

| 用户画像 | 核心场景 |
|----------|----------|
| 独立开发者 | 快速原型 → AI 辅助编码 → 一键上线 |
| 小团队 (2-10 人) | 协作开发 + Agent 自动化 + 共享知识库 |
| 中大型团队 (10-100 人) | RBAC + 审计 + 合规 + 私有部署 |
| AI 应用开发者 | 构建/调试/部署 Agent 和 MCP Server |

## 开发路线图

| 里程碑 | 时间 | 交付内容 |
|--------|------|----------|
| **M0 — 基础架构** | 第 1-2 月 | 基础设施、数据库、认证、CI/CD |
| **M1 — Alpha** | 第 3-4 月 | IDE (Monaco + 终端) + Agent (基础执行 + Tool Calling + 沙箱) |
| **M2 — Beta** | 第 5-6 月 | IDE (Git + Code Review) + Agent (前端 + 执行监控) |
| **M3 — 生态扩展** | 第 7-9 月 | Agent 可视化编排 + 模板市场 + 多 Agent 协作 |
| **M4 — 正式发布** | 第 10-12 月 | 性能优化 + 安全加固 + 插件系统 |

## 商业模式

```
Free Tier        → 个人开发者，基础模型额度，公开项目
Pro ($29/月)     → 高级模型，私有项目，更多存储/算力
Team ($19/人/月) → 协作，RBAC，共享资源池
Enterprise       → 私有部署，SLA，专属支持
```

## 文档索引

| 文档 | 说明 |
|------|------|
| [执行摘要](docs/architecture/00-EXECUTIVE-SUMMARY.md) | 项目愿景与定位 |
| [需求分析](docs/architecture/01-REQUIREMENTS-ANALYSIS.md) | 功能与非功能需求 |
| [功能边界](docs/architecture/02-FEATURE-BOUNDARY.md) | 模块边界与依赖关系 |
| [非功能需求](docs/architecture/03-NON-FUNCTIONAL-REQUIREMENTS.md) | 性能、安全、可用性指标 |
| [技术选型](docs/architecture/04-TECHNOLOGY-SELECTION.md) | 技术栈对比分析 |
| [系统架构](docs/architecture/05-SYSTEM-ARCHITECTURE.md) | 架构图 (Mermaid) |
| [数据库设计](docs/architecture/06-DATABASE-DESIGN.md) | ER 图与表结构 |
| [目录结构](docs/architecture/07-DIRECTORY-STRUCTURE.md) | 项目目录规范 |
| [开发规范](docs/architecture/08-DEVELOPMENT-STANDARDS.md) | 编码规范与流程 |
| [里程碑计划](docs/architecture/09-MILESTONE-PLAN.md) | 详细里程碑排期 |

## 开源协议

[MIT](LICENSE)
