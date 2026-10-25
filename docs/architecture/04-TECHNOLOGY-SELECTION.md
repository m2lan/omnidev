# OmniDev — 技术选型（当前实现）

## 1. 选型原则

当前技术栈服务于“单后端、可嵌入知识引擎、少量基础设施”的目标：优先复用 Go 生态和 PostgreSQL，避免为尚未出现的规模引入分布式组件。本文记录已使用的技术，不再保留虚构平台的候选方案比较。

## 2. 后端与模块组织

| 技术 | 当前用途 | 选择理由 |
|---|---|---|
| Go 1.22 | 网关、Agent、knowledge-engine | 编译部署简单，适合流式 HTTP、并发任务和 Docker 操作。 |
| Gin | `/api/v1` REST API、SSE、认证中间件 | API 简洁，兼容 `net/http`，现有中间件完整。 |
| pgx/v5 + pgxpool | PostgreSQL 访问与连接池 | 原生 SQL 可控，支持连接池、事务和 pgvector 查询。 |
| zap | 应用与 HTTP 日志 | 结构化字段、request ID 和延迟记录。 |
| Viper | `.env` 与环境变量配置 | 支持默认值、键映射和嵌套配置。 |
| Docker API/CLI | Agent 一次性代码沙箱 | 可直接设置网络、文件系统、权限和资源限制。 |

`apps/gateway` 包含 auth/users、chat、knowledge 和 agent。`packages/go-common` 是共享 Go module，承载配置、认证、中间件、缓存、存储、解析、日志、加密和 telemetry。

`../knowledge-engine` 是独立的可复用 Go module，网关通过以下方式引用，而非通过网络调用：

```go
replace github.com/omnidev/knowledge-engine => ../../../knowledge-engine
```

主程序使用 `NewFromPoolWithConfig` 嵌入知识引擎并共享 pgxpool。该模块也保留自己的 HTTP server 和 stdio MCP server，便于独立复用，但默认部署不启动额外知识服务。

## 3. 数据与基础设施

| 技术 | 当前用途 | 说明 |
|---|---|---|
| PostgreSQL 16 | 用户、聊天、知识、Agent 的唯一数据库 | 所有模块共享一个逻辑数据库，迁移按模块来源管理。 |
| pgvector | 文档 chunk 向量及余弦相似度检索 | 使用 HNSW；向量维度由 embedding 配置决定。 |
| PostgreSQL Full Text Search | 知识关键词检索 | 与向量结果通过 RRF 合并，无需 Elasticsearch。 |
| Redis 7 + go-redis | 会话、搜索/解析缓存、用量计数等 | knowledge-engine 的搜索缓存可回退到进程内存。 |
| MinIO | 聊天附件和可选知识文档对象存储 | 聊天附件依赖 MinIO；知识模块默认也可选本地目录。 |
| Apache Tika | 聊天文档与知识文档文本提取 | 通过 HTTP 调用，聊天解析结果可使用 Redis 缓存。 |

不引入独立消息队列或任务调度器。知识文档处理与 Agent run 使用进程内 goroutine；这一选择简单，但不提供持久队列和崩溃续跑能力。

## 4. AI 与 MCP

- **模型适配器**：OpenAI、DeepSeek、Qwen 使用 OpenAI 兼容 Chat Completions；Anthropic 使用其 Messages API。
- **配置来源**：支持全局环境变量和用户级 BYOK 配置；用户配置优先，provider 密钥可选 AES-256-GCM 加密存储。
- **Embedding**：knowledge-engine 支持 OpenAI、Gemini、Ollama 等配置，模型与维度记录在知识库中。
- **图片生成**：由网关图片服务和模型适配器完成，结果可作为附件保存。
- **MCP client**：网关使用 Streamable HTTP + JSON-RPC 2.0，支持 JSON 或 SSE 响应、session header 和工具分页。
- **MCP 配置**：`MCP_ENABLED` 控制启用，`MCP_SERVERS` 是包含 name、URL、API key、headers、timeout 的 JSON 数组。
- **工具循环**：OpenAI/DeepSeek/Qwen 可用，最多 5 轮、单结果 8000 rune；Anthropic tool-use 映射尚未实现。

Agent 规划器直接调用 OpenAI 兼容 HTTP endpoint，不复用 Temporal，也不要求独立 Agent 服务。

## 5. 前端

| 技术 | 当前用途 |
|---|---|
| Next.js 15 | `apps/web` 应用框架与开发/构建命令。 |
| React 19 + TypeScript | 工作台页面、组件和类型安全。 |
| Tailwind CSS | 样式体系，配合 Radix UI 等现有组件依赖。 |
| TanStack React Query | 服务端数据获取与缓存。 |
| Zustand | 必要的客户端状态。 |
| React Hook Form + Zod | 表单与输入校验。 |
| Vitest | 前端测试。 |
| pnpm 9 + Turbo | workspace 依赖和任务管理。 |

前端通过 HTTP/SSE 使用网关 API，不包含 Monaco 在线 IDE 或可视化工作流画布。

## 6. 部署与开发工具

- 使用 Dockerfile 构建单个 gateway 镜像；构建上下文包含同级 `knowledge-engine`。
- 使用 Docker Compose 启动 PostgreSQL/pgvector、Redis、MinIO、Tika，以及可选 gateway 和 Nginx。
- `migrate/migrate` 执行 `apps/gateway/migrations`；knowledge-engine 的嵌入式 migrations 在引擎启动时执行。
- Make 统一封装 Go build/test/vet/lint、前端命令、数据库迁移、Compose 和沙箱镜像预拉取。
- 可选 OTLP tracing 由 `packages/go-common/telemetry` 初始化；未采用完整 Prometheus/Grafana 平台。

## 7. 已移除/不再采用

| 技术或能力 | 不再采用的原因 |
|---|---|
| Kafka | 当前模块在进程内直接调用，异步任务无需额外消息基础设施。 |
| Elasticsearch | pgvector 与 PostgreSQL 全文检索已覆盖当前混合搜索需求。 |
| Temporal | 当前 Agent/文档处理不需要持久工作流引擎，避免额外运维成本。 |
| 微服务拆分 | 单二进制已满足交付需求，拆分只会增加网络、部署和调试成本。 |
| Helm/Kubernetes manifests | 当前部署目标是 Docker/Compose，不维护未使用的编排清单。 |
| Monaco 在线 IDE | 在线 IDE 已下线，编码能力聚焦后台 Agent 与工作区工具。 |
| 可视化 workflow engine | 当前逐步 Agent 执行已覆盖目标场景，无需独立画布和执行引擎。 |
| Billing | 没有支付或订阅产品边界；用量记录可由后续外部系统集成，不自建计费平台。 |

这些组件不应再出现在运行依赖、部署前置条件或开发命令中。
