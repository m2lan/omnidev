# OmniDev — 系统架构

## 1. 架构原则

当前系统是面向团队的 AI 知识与编码工作台，只保留三条产品主线：Chat、知识库、后台 Agent。

- **模块化单体**：认证、用户、Chat、知识库和 Agent 均运行在 `apps/gateway` 的同一个 Go 进程中，通过包边界而不是网络拆分模块。
- **复用知识引擎**：gateway 直接 import 兄弟仓库 `knowledge-engine`，不维护第二份 RAG 实现；该引擎也可独立提供 HTTP API 和 MCP server。
- **按真实依赖部署**：只需要 PostgreSQL + pgvector、Redis、MinIO、Tika 和 Docker。Kafka、Elasticsearch、Temporal、Kubernetes、Helm、Terraform 均不属于当前架构。
- **隔离不可信代码**：Agent 代码只在 Docker 沙箱中执行，没有宿主机直接执行的后备路径。
- **文档与代码一致**：在线 IDE、拖拽工作流、支付、部署托管和监控全家桶已经移除；Agent 自动提交 PR 仍是规划项。

## 2. 运行拓扑

```mermaid
graph TB
    Browser["浏览器"]
    Web["apps/web<br/>Next.js 15 / React 19"]

    subgraph Gateway["apps/gateway — 单个 Go 进程"]
        HTTP["Gin Router / Middleware<br/>JWT · API Key · CORS · Recovery"]
        Auth["认证与用户模块"]
        Chat["Chat 模块<br/>会话 · 附件 · 多模态"]
        Knowledge["知识模块<br/>knowledge-engine 内嵌实例"]
        MCP["MCP Client<br/>工具发现与调用"]
        Agent["Agent 模块<br/>Planner · Executor · Tools"]
        Repo["Repository<br/>统一 PostgreSQL 数据访问"]
    end

    Models["模型服务<br/>OpenAI · DeepSeek · Qwen · Anthropic"]
    ExternalMCP["外部 MCP Servers"]
    Docker["Docker Daemon<br/>一次性 Agent 沙箱容器"]
    PG[("PostgreSQL 16 + pgvector")]
    Redis[("Redis 7<br/>知识检索与嵌入缓存")]
    MinIO[("MinIO<br/>附件与文档对象")]
    Tika["Apache Tika<br/>文档解析"]
    KE["knowledge-engine<br/>可选独立 HTTP / MCP 服务"]

    Browser --> Web
    Web -->|"HTTP / SSE"| HTTP
    HTTP --> Auth
    HTTP --> Chat
    HTTP --> Knowledge
    HTTP --> Agent
    Chat --> Knowledge
    Chat --> MCP
    Chat --> Models
    Agent --> Models
    Agent --> Docker
    MCP --> ExternalMCP
    Auth --> Repo
    Chat --> Repo
    Knowledge --> Repo
    Agent --> Repo
    Repo --> PG
    Knowledge --> Redis
    Knowledge --> MinIO
    Knowledge --> Tika
    Chat --> MinIO
    Chat --> Tika
    KE -. "可独立部署，不是 gateway 的必需进程" .-> PG
```

`knowledge-engine` 是编译进 gateway 的 Go 依赖，不是 gateway 必须访问的微服务。只有需要把它提供给其他客户端时，才单独启动它的 HTTP/MCP 入口。

## 3. 后端模块边界

| 包 | 职责 | 不负责 |
|---|---|---|
| `internal/router`、`internal/handler` | 路由、HTTP 参数与响应、SSE、JWT 保护 | 业务规则与数据库细节 |
| `internal/service` | 认证、用户、会话、消息和模型配置等业务流程 | Agent 内部执行状态机 |
| `internal/adapter` | 不同模型协议的请求、响应和流式解析 | 跨供应商的伪统一能力承诺 |
| `internal/knowledge` | 把知识引擎 Handler 接入 gateway 路由 | 复制 RAG 存储与检索实现 |
| `internal/mcp` | Streamable HTTP MCP 初始化、工具列表、工具调用与命名冲突处理 | MCP server 管理后台 |
| `internal/agent` | Agent 配置、异步运行、规划、步骤、工具、取消与沙箱 | Git 操作、自动 PR、长期工作流 |
| `internal/repository` | gateway 业务表的数据访问 | 跨服务 RPC |
| `packages/go-common` | 配置、日志、数据库、缓存、认证、存储、解析等共享能力 | 消息总线或工作流引擎 |

模块间以 Go 接口和结构体交互。当前没有 gRPC、服务发现、分布式事件总线或独立 worker 集群。

## 4. Chat 请求链路

### 4.1 普通对话与知识增强

```mermaid
sequenceDiagram
    participant U as Web
    participant G as Gateway
    participant C as Chat Service
    participant K as Knowledge Engine
    participant M as Model Adapter
    participant P as Model Provider
    participant D as PostgreSQL / MinIO

    U->>G: 发送消息（可带附件与知识库）
    G->>G: 验证 JWT 与用户资源权限
    G->>C: 构造对话请求
    C->>D: 读取会话、历史消息与附件元数据
    opt 选择知识库
        C->>K: 按用户与知识库检索
        K-->>C: 返回片段与引用
        C->>C: 注入检索上下文
    end
    C->>M: Chat / StreamChat
    M->>P: 对应模型协议请求
    alt 流式响应
        P-->>M: SSE chunks
        M-->>C: 文本增量
        C-->>U: SSE 文本事件
    else 非流式响应
        P-->>M: 完整响应
        M-->>C: 助手消息
        C-->>U: JSON 响应
    end
    C->>D: 保存用户消息、最终助手消息与用量
```

### 4.2 外部 MCP 工具调用

启用 `MCP_ENABLED` 后，gateway 在启动时连接配置中的 MCP server 并获取工具列表。Chat 在模型支持工具调用时进入最多 5 轮的“模型 → 工具 → 模型”循环，工具结果截断到 8000 字符，最终把工具名记录到助手消息的 `tools_used` 元数据。

当前 OpenAI、DeepSeek、Qwen 适配器支持该链路。Anthropic 普通对话可用，但 `tool_use` / `tool_result` 尚未映射，因此不会向 Anthropic 请求宣告 MCP 工具。MCP server 暂时不可用时记录警告，不阻止 gateway 启动。

## 5. Agent 异步执行链路

```mermaid
sequenceDiagram
    participant U as Web / API Client
    participant H as Agent Handler
    participant S as Agent Service
    participant P as Planner
    participant E as Executor
    participant T as Tool Registry
    participant D as Docker Sandbox
    participant DB as PostgreSQL

    U->>H: POST /api/v1/agents/:id/run
    H->>S: 创建运行
    S->>DB: 保存 run（created）
    S-->>U: 立即返回 run，后台 goroutine 执行
    S->>P: 根据任务与 Agent 配置规划
    loop 直到完成、失败、达到步数上限或取消
        P-->>E: 下一步动作
        E->>DB: 持久化步骤与状态
        alt file / search / calculator
            E->>T: 在当前 run 工作区执行内置工具
            T-->>E: 工具结果
        else code_exec
            E->>D: 挂载 run 工作区并启动隔离容器
            D-->>E: stdout / stderr / 退出状态
        end
        E->>P: 将步骤结果交回后续规划
    end
    S->>DB: 保存 success / failed / cancelled 与最终结果
    U->>H: 轮询 GET /api/v1/agent-runs/:run_id
    H-->>U: 当前状态、步骤、结果或错误
    opt 用户取消
        U->>H: POST /api/v1/agent-runs/:run_id/cancel
        H->>S: 请求取消并回收执行上下文
    end
```

运行记录和步骤存 PostgreSQL；每个 run 使用独立工作区。当前异步队列是进程内 goroutine，不是持久化分布式任务队列，因此不能把进程重启后的任务恢复描述为已实现能力。

## 6. Docker 沙箱边界

`code_exec` 为每次执行创建一次性容器：

- 仅把当前 run 的工作区绑定到 `/workspace`，文件工具也限制在对应工作区内。
- 默认关闭网络，根文件系统只读，`/tmp` 使用 tmpfs。
- 丢弃全部 Linux capabilities，启用 `no-new-privileges`。
- 限制 CPU、内存、PID 数、执行超时和输出大小；默认超时 120 秒，输出最多保留 1 MiB。
- 执行结束或取消后强制清理容器。

当 gateway 自身运行在容器中时，Docker daemon 看到的是宿主机路径。`SANDBOX_WORKSPACE_ROOT` 表示 gateway 容器内路径，`SANDBOX_HOST_WORKSPACE_ROOT` 表示同一目录在宿主机上的路径；二者不能混用。Compose 通过挂载 Docker socket 和宿主机工作区支持这一模式。

挂载 Docker socket 等价于授予 gateway 较高的宿主机权限，只应在可信、受控的部署环境使用。当前隔离基于普通 Docker 容器，不宣称已经提供 gVisor、Seccomp 定制配置或多租户强隔离保证。

## 7. 数据与外部依赖

| 依赖 | 当前用途 |
|---|---|
| PostgreSQL 16 | 用户、认证、模型配置、会话、消息、附件、Agent、运行与步骤 |
| pgvector | knowledge-engine 的文档向量与相似度检索，维度随 embedding 配置 |
| Redis 7 | knowledge-engine 的检索与嵌入缓存 |
| MinIO | Chat 附件和知识文档对象 |
| Apache Tika | 附件与知识文档文本解析 |
| Docker | Agent `code_exec` 沙箱 |
| 外部模型 API | Chat 与 Agent 推理、知识 embedding |
| 外部 MCP server | 可选的 Chat 工具扩展 |

数据库迁移集中在 `apps/gateway/migrations`，按 `001` 至 `009` 顺序执行，所有业务模块共享一个数据库 schema。

## 8. 部署形态

开发环境使用两份 Compose 文件：

- `docker-compose.infra.yml`：PostgreSQL/pgvector、Redis、MinIO、Tika 和一次性 migration 容器。
- `docker-compose.dev.yml`：gateway 与 nginx；按需覆盖基础设施文件共同启动。

前端可在宿主机用 Next.js 开发服务器运行，后端可用 `go run` 或容器运行。staging CD 当前只构建并推送 gateway 镜像，不包含虚构的 Helm 自动部署；实际发布按 Compose 步骤执行。

当前未提供生产级高可用、自动扩缩容、备份恢复、SLA 或完整可观测性平台。这些必须通过后续运维设计与验证单独交付，不能从开发 Compose 推导为既有能力。
