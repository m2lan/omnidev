# OmniDev — 非功能性需求与当前约束

本文只记录代码和配置中可验证的性质，不定义尚未测量或建设完成的 SLA、容量和合规承诺。

## 1. 身份、授权与数据保护

| 项目 | 当前实现 |
|---|---|
| 密码 | 使用 bcrypt 哈希；注册与登录不保存明文密码。 |
| JWT | `/api/v1` 受保护路由校验 access token；默认有效期 15 分钟，refresh token 默认 168 小时，均可配置。 |
| 资源授权 | 会话、消息、附件、AI 配置、知识库和 Agent 服务按 `user_id` 做所有权检查。 |
| API Key | 支持创建、列表和吊销，数据库保存 bcrypt 哈希、前缀、scopes 和过期时间；完整 Key 只在创建响应中返回。 |
| Provider 密钥 | 配置有效 `ENCRYPTION_KEY` 时，用户 BYOK 密钥使用 AES-256-GCM 加密存储；未配置时按明文存储，生产环境必须显式配置并妥善轮换。 |
| OAuth | GitHub/Google 凭据来自环境变量，只有完成对应配置才可用。 |

CORS 当前允许所有来源并支持凭据，生产环境应在反向代理层限制来源。API Key 当前主要提供管理与校验能力，不能据此推断所有业务路由都已支持 API Key 替代 JWT。系统没有独立审计日志套件或认证合规声明。

## 2. Agent 沙箱隔离

代码执行通过一次性 Docker 容器完成，默认配置如下：

| 控制项 | 当前值/行为 |
|---|---|
| 网络 | `SANDBOX_NETWORK=none`，容器默认无网络。 |
| 根文件系统 | `--read-only`；仅 run 工作区和指定 tmpfs 可写。 |
| 工作区 | 每个 run 独立目录，挂载到 `/workspace`，工具路径限制在该目录内。 |
| 临时目录 | `/tmp` 为 64 MiB tmpfs，挂载参数包含 `noexec,nosuid`。 |
| 权限 | `--cap-drop ALL`、`no-new-privileges`，并使用宿主当前 UID/GID 运行。 |
| CPU / 内存 | 默认 1 CPU、512 MiB，可由 `SANDBOX_CPU_LIMIT`、`SANDBOX_MEMORY_LIMIT` 调整。 |
| 进程数 | `--pids-limit 128`。 |
| 超时 | 默认 120 秒；结束后强制清理容器。 |
| 输出 | stdout、stderr 各最多保留 1 MiB，超出时标记 `truncated`。 |
| 语言 | Python、JavaScript、Go、Shell，各自使用可配置 Docker 镜像。 |

Docker socket 的挂载使网关具备启动容器的能力，部署时应限制网关宿主权限和工作区目录。沙箱是资源隔离措施，不应被描述为经过认证的多租户安全边界。

## 3. 请求与资源约束

| 项目 | 当前约束 |
|---|---|
| 聊天附件 | 单文件最大 20 MiB，仅允许代码中列出的图片、PDF、Word、文本和 Markdown 类型。 |
| 聊天上下文 | 每次读取最近 20 条消息；知识库请求每库 TopK=3，总共最多注入 10 条。 |
| MCP 循环 | 最多 5 轮；每次工具输出截断到 8000 rune；单服务器默认 HTTP 超时 30 秒，可配置。 |
| Agent 步数 | 默认最多 20 步，可在 Agent 配置中调整。 |
| 分页 | 会话、知识库、Agent 和 run 默认每页 20，消息默认 50；页大小通常限制为最多 100。 |
| 知识处理 | 文档后台处理上下文超时 30 分钟；分块大小有校验，默认 512、重叠 50。 |
| HTTP Server | ReadTimeout 30 秒、WriteTimeout 300 秒（兼顾 SSE）、IdleTimeout 120 秒。 |

这些值是代码限制或默认值，不代表已验证的性能指标。AI 首 Token、模型吞吐、文档处理时间主要取决于外部 provider、文件大小和部署资源，当前没有 P95/P99 承诺。

## 4. 限流、连接池与缓存

- Redis 连接成功时网关挂载按客户端 IP 的令牌桶限流器，默认速率 30 token/s、突发 50；Redis 不可用时不启用该中间件。
- **当前限流状态实际保存在进程内 map 中，并非 Redis 分布式限流**；多实例不会共享额度，重启会清空状态。
- PostgreSQL 使用 pgxpool：默认最大连接 25、最小空闲连接 5、连接最大生命周期 5 分钟，另有 5 分钟最大空闲时间和 1 分钟健康检查周期。
- Redis 配置支持重试次数和池大小；知识搜索缓存使用 Redis，连接失败时 knowledge-engine 可退回内存缓存。
- Tika 聊天解析结果可使用 Redis 缓存；知识库变更通过搜索版本使旧缓存失效。

## 5. 可用性与故障语义

- `/health`、`/ready`、`/info` 提供基础健康与版本信息；网关收到 SIGINT/SIGTERM 后最多用 30 秒优雅关闭 HTTP Server。
- AI provider、MCP 或单个知识检索失败会返回错误或记录警告；聊天对单库 RAG 检索失败采用跳过该库的方式继续。
- MinIO 初始化失败会禁用聊天上传；Redis 初始化失败会禁用限流并降低缓存能力；PostgreSQL 或知识引擎初始化失败会阻止启动。
- 知识文档和 Agent run 都由进程内 goroutine 执行，没有 Kafka/Temporal 或持久化工作队列。进程重启后的自动续跑、跨实例取消和任务恢复不属于当前能力。
- 当前没有多区域部署、自动故障转移、水平自动扩缩容或 99.9% SLA 承诺；备份与灾难恢复由部署环境另行负责。

## 6. 可观测性

- 使用 zap 记录结构化应用日志；HTTP 中间件记录 request ID、方法、路径、状态码和延迟，并支持 `X-Request-ID` 透传。
- Recovery 中间件将 panic 转换为统一 500 响应。
- `packages/go-common/telemetry` 提供 OTLP/gRPC tracing 初始化，采样率为 10%。当前默认 endpoint 是 `http://localhost:4317`，网关启动时会调用初始化，失败仅记录警告；它并非“未配置即禁用”的开关。
- 当前没有 Prometheus 指标端点、完整 Grafana 监控栈或独立告警平台；只有 tracing 初始化钩子不等于完整的端到端链路埋点。

## 7. 可维护性与部署约束

- 单一 Go 二进制降低部署和调用链复杂度；模块通过包边界、repository 和 service 维持职责分离。
- 数据访问使用参数化 SQL 和连接池；列表接口使用分页，消息表按月分区。
- 消息分区只在迁移时预建当月及后续 3 个月，后续分区需要主动维护。
- 配置由 Viper 从 `.env` 和环境变量读取，密钥不得写入版本库。
- 开发环境仅要求 Docker Compose 中的 PostgreSQL+pgvector、Redis、MinIO、Tika，以及可选网关/Nginx；生产部署不依赖 Kubernetes。
