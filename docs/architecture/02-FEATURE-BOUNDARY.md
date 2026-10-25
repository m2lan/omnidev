# OmniDev — 功能边界与模块划分（当前实现）

## 1. 总体边界

OmniDev 是由一个 Go 后端和一个 Next.js 前端组成的模块化单体，不按业务模块拆分网络服务。

```text
apps/web ── HTTP/SSE ──> apps/gateway
                            ├─ auth / users
                            ├─ chat / attachments / user AI configs
                            ├─ knowledge（嵌入 ../knowledge-engine）
                            └─ background agent / Docker sandbox
                                  │
                 PostgreSQL+pgvector / Redis / MinIO / Tika
```

`packages/go-common` 是进程内共享库，不是独立服务。knowledge-engine 是可复用 Go module，通过 `replace` 引用并复用网关连接池；其独立 HTTP/MCP 入口不属于 OmniDev 默认部署拓扑。

## 2. 模块职责

### 2.1 Auth 与 Users

**边界内**：邮箱注册登录、JWT 签发与刷新、GitHub/Google OAuth、资料维护、API Key 创建/列表/吊销、个人 AI provider 配置。

**边界外**：完整组织后台、邀请邮件、企业级 RBAC 和审计系统。`organizations/org_members` 表存在不代表组织 API 已交付；当前受保护资源主要按用户所有权隔离。

### 2.2 Chat

**边界内**：

- 会话 CRUD、消息分页、普通与 SSE 流式回答。
- OpenAI、DeepSeek、Qwen、Anthropic 适配器及用户级 BYOK 配置。
- 最近消息上下文、system prompt、消息用量和延迟记录。
- MinIO 附件、图片多模态输入、Tika 文档文本提取、图片生成。
- 调用知识引擎完成 RAG 注入，调用 MCP Manager 完成外部工具循环。

**边界外**：模型训练与托管、在线 IDE、多模型并排评测、计费结算。`models` 和 `prompt_templates` 是数据表；当前没有独立 Prompt 模板路由，模型列表主要由已注册适配器和用户配置产生。

### 2.3 Knowledge

**边界内**：知识库 CRUD、文档上传/列表/删除、异步解析与分块、embedding、pgvector 与全文混合检索、搜索缓存、聊天 RAG 查询。

**职责接口**：网关 HTTP handler 调用 `KnowledgeBaseService` 和 `SearchService`；聊天服务直接复用同一个 `SearchService` 实例，不经过内部 HTTP。

**边界外**：Elasticsearch 集群、独立向量数据库、可视化数据处理流水线。文档处理使用进程内 goroutine，不是持久化任务队列。

### 2.4 Background Agent

**边界内**：Agent 配置 CRUD、异步 run、步骤持久化、轮询与取消；提供 file/search/calculator/code_exec 工具和 Docker 沙箱。

**运行边界**：每次 run 使用独立宿主目录并挂载为 `/workspace`。规划器只要求 OpenAI 兼容 endpoint；运行时不依赖 Temporal 或外部工作流引擎。

**边界外**：多 Agent 协作、可视化编排、持久化分布式任务恢复、自动创建分支或 PR。Agent 表中的扩展字段不应被解释为这些能力已经可用。

### 2.5 MCP 外部工具

MCP 是 Chat 的出站集成边界，不是独立 MCP 管理服务：

- `apps/gateway/internal/mcp` 实现 Streamable HTTP JSON-RPC client 和多服务器工具聚合。
- `MCP_ENABLED=true` 时读取 `MCP_SERVERS` JSON；可配置 URL、Bearer Key、额外请求头和超时。
- 工具名冲突时使用服务器名前缀消除冲突。
- 最多 5 轮工具调用，单次工具文本结果最多向模型注入 8000 rune。
- 当前仅 OpenAI/DeepSeek/Qwen 适配器声明工具支持；Anthropic 映射未实现。

### 2.6 Web 与共享基础设施

- `apps/web` 负责工作台 UI、API 调用和流式结果展示，不承载服务端业务规则。
- `packages/go-common` 负责配置、JWT、中间件、PostgreSQL/Redis、MinIO、Tika、日志、加密和 OTel 初始化。
- PostgreSQL 是唯一事务与向量数据库；Redis 用于缓存和会话等数据；MinIO 保存聊天附件及可选知识文档；Tika 负责文档解析。
- 开发部署使用 Docker Compose；Nginx 只是开发环境反向代理，不改变单体架构。

## 3. HTTP API 边界

除健康检查和认证入口外，下列 `/api/v1` 路由均要求 JWT。

| 领域 | 路由摘要 |
|---|---|
| 健康 | `GET /health`、`GET /ready`、`GET /info` |
| 认证 | `POST /auth/register|login|refresh`，`GET /auth/oauth/:provider`、`GET /auth/callback/:provider` |
| 用户 | `GET|PATCH /users/me`，`GET|POST /users/me/api-keys`，`DELETE /users/me/api-keys/:id` |
| 会话 | `GET|POST /conversations`，`GET|PATCH|DELETE /conversations/:id` |
| 消息 | `GET|POST /conversations/:id/messages`，`POST /conversations/:id/messages/stream` |
| 模型与图片 | `GET /models`，`POST /images/generate` |
| 附件 | `POST /upload`，`GET|DELETE /attachments/:id` |
| AI 配置 | `/user/ai-configs` CRUD，以及 `PUT /:id/default`、`POST /:id/test` |
| 知识库 | `/knowledge` CRUD，`/:id/documents` 列表/上传，`DELETE /:id/documents/:doc_id`，`POST /:id/search` |
| Agent | `/agents` CRUD，`POST /agents/:id/run`，`GET /agents/:id/runs` |
| Agent run | `GET /agent-runs/:run_id`，`POST /agent-runs/:run_id/cancel` |

Agent 路由仅在 `SANDBOX_ENABLED=true` 时注册。`/api/v1/admin` 目前只有中间件分组，没有管理端点。

## 4. 数据归属

| 数据 | 所有者/父级 | 主要写入模块 |
|---|---|---|
| users、oauth_connections、api_keys | 用户 | Auth/Users |
| user_ai_configs | 用户 | Users/Chat adapter resolver |
| conversations、messages、attachments | 用户/会话 | Chat |
| knowledge_bases、documents、document_chunks | 用户/知识库 | Knowledge |
| agents、agent_runs、agent_steps | 用户/Agent/run | Background Agent |
| organizations、org_members、prompt_templates | 预留或有限使用的数据模型 | 当前无独立业务路由 |

所有模块使用同一个 PostgreSQL 数据库；模块边界由包和所有权检查维持，而不是按服务拆库。

## 5. 非目标、已下线与未实现

**非目标/已下线**：在线 IDE、可视化工作流画布、计费/订阅、完整监控栈、一键部署、通知服务，以及 Kafka、Elasticsearch、Temporal、Kubernetes 所对应的平台能力。

**未实现/规划中**：Agent PR 交付、Anthropic tool calling。二者不得写入当前 API 或模块的验收承诺。
