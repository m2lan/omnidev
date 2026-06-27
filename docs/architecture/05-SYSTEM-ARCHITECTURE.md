# OmniDev AI Platform — 系统架构

## 1. 整体架构（C4 Level 1 — 系统上下文）

```mermaid
C4Context
    title OmniDev AI Platform — 系统上下文图

    Person(dev, "开发者", "使用平台进行 AI 辅助开发")
    Person(admin, "管理员", "管理平台运营")
    Person(user, "终端用户", "使用部署的应用")

    System(omnidev, "OmniDev AI Platform", "AI 开发平台，提供 IDE、Agent、RAG、部署等能力")

    System_Ext(ai_providers, "AI 模型提供商", "OpenAI, Anthropic, Google, DeepSeek, Ollama")
    System_Ext(github, "GitHub", "代码托管、OAuth")
    System_Ext(cloud, "云服务商", "AWS, Azure, GCP")
    System_Ext(payment, "支付服务", "Stripe, 微信支付, 支付宝")
    System_Ext(email, "邮件服务", "SendGrid, SMTP")

    Rel(dev, omnidev, "开发、调试、部署")
    Rel(admin, omnidev, "管理、监控")
    Rel(user, cloud, "访问部署的应用")

    Rel(omnidev, ai_providers, "AI 推理请求", "HTTPS")
    Rel(omnidev, github, "Git 操作、OAuth", "HTTPS")
    Rel(omnidev, cloud, "部署应用", "API")
    Rel(omnidev, payment, "处理支付", "HTTPS")
    Rel(omnidev, email, "发送通知", "SMTP/API")
```

---

## 2. 分层架构（C4 Level 2 — 容器图）

```mermaid
graph TB
    subgraph "客户端层 Client Layer"
        WEB["Web 浏览器<br/>Next.js SPA"]
        CLI["CLI 工具<br/>Go Binary"]
        API_CLIENT["API 客户端<br/>SDK / cURL"]
    end

    subgraph "接入层 Access Layer"
        CDN["CDN<br/>Cloudflare"]
        LB["负载均衡<br/>Nginx / K8s Ingress"]
        GW["API Gateway<br/>Gin + JWT + Rate Limit"]
    end

    subgraph "应用服务层 Application Services"
        USER_SVC["User Service<br/>认证/授权/组织"]
        CHAT_SVC["Chat Service<br/>对话/模型路由/流式"]
        AGENT_SVC["Agent Service<br/>任务/工具/执行"]
        RAG_SVC["RAG Service<br/>文档/向量/检索"]
        IDE_SVC["IDE Service<br/>文件/终端/Git"]
        WF_SVC["Workflow Service<br/>编排/触发/执行"]
        MCP_SVC["MCP Service<br/>工具注册/发现/路由"]
        DEPLOY_SVC["Deploy Service<br/>构建/部署/域名"]
        NOTIFY_SVC["Notification Service<br/>邮件/Slack/站内信"]
    end

    subgraph "支撑服务层 Supporting Services"
        BILLING_SVC["Billing Service<br/>计量/账单/支付"]
        ADMIN_SVC["Admin Service<br/>用户/模型/配置"]
        MONITOR_SVC["Monitor Service<br/>指标/日志/告警"]
        PLUGIN_SVC["Plugin Service<br/>注册/发现/沙箱"]
    end

    subgraph "基础设施层 Infrastructure"
        PG[("PostgreSQL 16<br/>+ pgvector")]
        REDIS[("Redis 7<br/>Cluster")]
        KAFKA["Kafka<br/>KRaft"]
        MINIO["MinIO<br/>对象存储"]
        ES["Elasticsearch 8"]
        TEMPORAL["Temporal<br/>工作流引擎"]
    end

    subgraph "运行时 Runtime"
        K8S["Kubernetes"]
        DOCKER["Docker<br/>沙箱容器"]
    end

    subgraph "可观测性 Observability"
        PROM["Prometheus"]
        GRAFANA["Grafana"]
        LOKI["Loki"]
        JAEGER["Jaeger"]
        OTEL["OpenTelemetry Collector"]
    end

    WEB --> CDN --> LB
    CLI --> LB
    API_CLIENT --> LB
    LB --> GW

    GW --> USER_SVC
    GW --> CHAT_SVC
    GW --> AGENT_SVC
    GW --> RAG_SVC
    GW --> IDE_SVC
    GW --> WF_SVC
    GW --> MCP_SVC
    GW --> DEPLOY_SVC
    GW --> BILLING_SVC
    GW --> ADMIN_SVC
    GW --> MONITOR_SVC

    CHAT_SVC --> RAG_SVC
    CHAT_SVC --> MCP_SVC
    AGENT_SVC --> CHAT_SVC
    AGENT_SVC --> MCP_SVC
    AGENT_SVC --> DOCKER
    IDE_SVC --> DOCKER
    WF_SVC --> TEMPORAL
    WF_SVC --> CHAT_SVC
    DEPLOY_SVC --> K8S

    USER_SVC --> PG
    CHAT_SVC --> PG
    AGENT_SVC --> PG
    RAG_SVC --> PG
    IDE_SVC --> MINIO
    WF_SVC --> PG
    BILLING_SVC --> PG
    ADMIN_SVC --> PG

    USER_SVC --> REDIS
    CHAT_SVC --> REDIS
    AGENT_SVC --> REDIS

    CHAT_SVC --> KAFKA
    AGENT_SVC --> KAFKA
    RAG_SVC --> KAFKA
    BILLING_SVC --> KAFKA
    NOTIFY_SVC --> KAFKA

    RAG_SVC --> ES
    AGENT_SVC --> ES

    ALL_SVC["所有服务"] --> OTEL
    OTEL --> PROM
    OTEL --> LOKI
    OTEL --> JAEGER
    PROM --> GRAFANA
    LOKI --> GRAFANA
    JAEGER --> GRAFANA
```

---

## 3. 微服务架构（C4 Level 3 — 组件图）

### 3.1 API Gateway

```mermaid
graph LR
    subgraph "API Gateway"
        AUTH_MW["认证中间件<br/>JWT / API Key"]
        RATE_MW["限流中间件<br/>Token Bucket"]
        CORS_MW["CORS 中间件"]
        LOG_MW["日志中间件<br/>请求/响应"]
        ROUTER["路由器<br/>Gin Router"]
        GRPC_GW["gRPC-Gateway<br/>REST→gRPC"]
        WS_PROXY["WebSocket 代理<br/>IDE/终端"]
        SSE_PROXY["SSE 代理<br/>AI 流式"]
    end

    Request --> CORS_MW --> RATE_MW --> AUTH_MW --> LOG_MW --> ROUTER
    ROUTER --> GRPC_GW
    ROUTER --> WS_PROXY
    ROUTER --> SSE_PROXY
```

### 3.2 Chat Service

```mermaid
graph TB
    subgraph "Chat Service"
        CONV_MGR["会话管理器<br/>ConversationManager"]
        MSG_MGR["消息管理器<br/>MessageManager"]
        CTX_BUILDER["上下文构建器<br/>ContextBuilder"]
        MODEL_ROUTER["模型路由器<br/>ModelRouter"]
        STREAM_MGR["流式管理器<br/>StreamManager"]
        TOKEN_METER["Token 计量器<br/>TokenMeter"]
        PROMPT_MGR["Prompt 管理器<br/>PromptManager"]
    end

    subgraph "Model Adapters"
        OPENAI["OpenAI Adapter"]
        CLAUDE["Anthropic Adapter"]
        GEMINI["Gemini Adapter"]
        DEEPSEEK["DeepSeek Adapter"]
        QWEN["Qwen Adapter"]
        OLLAMA["Ollama Adapter"]
    end

    CONV_MGR --> MSG_MGR
    MSG_MGR --> CTX_BUILDER
    CTX_BUILDER --> MODEL_ROUTER
    MODEL_ROUTER --> OPENAI
    MODEL_ROUTER --> CLAUDE
    MODEL_ROUTER --> GEMINI
    MODEL_ROUTER --> DEEPSEEK
    MODEL_ROUTER --> QWEN
    MODEL_ROUTER --> OLLAMA
    MODEL_ROUTER --> STREAM_MGR
    STREAM_MGR --> TOKEN_METER
    TOKEN_MGR --> PROMPT_MGR
```

### 3.3 Agent Service

```mermaid
graph TB
    subgraph "Agent Service"
        TASK_PLANNER["任务规划器<br/>TaskPlanner"]
        STEP_EXECUTOR["步骤执行器<br/>StepExecutor"]
        TOOL_MANAGER["工具管理器<br/>ToolManager"]
        STATE_MACHINE["状态机<br/>AgentStateMachine"]
        MEMORY_MGR["记忆管理器<br/>MemoryManager"]
        TRACE_MGR["追踪管理器<br/>TraceManager"]
    end

    subgraph "Tool Types"
        BUILTIN_TOOLS["内置工具<br/>文件/搜索/计算"]
        MCP_TOOLS["MCP 工具<br/>MCP Server"]
        CODE_TOOLS["代码工具<br/>Sandbox"]
        CHAT_TOOLS["AI 工具<br/>Chat Service"]
    end

    TASK_PLANNER --> STEP_EXECUTOR
    STEP_EXECUTOR --> TOOL_MANAGER
    TOOL_MANAGER --> BUILTIN_TOOLS
    TOOL_MANAGER --> MCP_TOOLS
    TOOL_MANAGER --> CODE_TOOLS
    TOOL_MANAGER --> CHAT_TOOLS
    STEP_EXECUTOR --> STATE_MACHINE
    STEP_EXECUTOR --> MEMORY_MGR
    STEP_EXECUTOR --> TRACE_MGR
```

### 3.4 RAG Service

```mermaid
graph TB
    subgraph "RAG Service"
        DOC_PARSER["文档解析器<br/>DocParser"]
        OCR_ENGINE["OCR 引擎<br/>Tesseract/PaddleOCR"]
        CHUNKER["分块器<br/>SemanticChunker"]
        EMBEDDER["向量化器<br/>Embedder"]
        VECTOR_STORE["向量存储<br/>pgvector"]
        BM25_INDEX["BM25 索引<br/>Elasticsearch"]
        RERANKER["重排序器<br/>Reranker"]
        HYBRID_SEARCH["混合检索<br/>HybridSearch"]
        CITATION["引用溯源<br/>CitationManager"]
    end

    DOC_PARSER --> OCR_ENGINE
    DOC_PARSER --> CHUNKER
    CHUNKER --> EMBEDDER
    EMBEDDER --> VECTOR_STORE

    HYBRID_SEARCH --> VECTOR_STORE
    HYBRID_SEARCH --> BM25_INDEX
    HYBRID_SEARCH --> RERANKER
    RERANKER --> CITATION
```

---

## 4. 数据流架构

### 4.1 AI 对话流

```mermaid
sequenceDiagram
    participant U as 用户浏览器
    participant GW as API Gateway
    participant CS as Chat Service
    participant RS as RAG Service
    participant MA as Model Adapter
    participant AI as AI Provider
    participant DB as PostgreSQL
    participant K as Kafka
    participant BS as Billing Service

    U->>GW: POST /api/v1/chat (SSE)
    GW->>GW: JWT 验证 + 限流
    GW->>CS: gRPC ChatStream

    CS->>CS: 构建上下文 (历史 + System Prompt)

    alt 启用 RAG
        CS->>RS: gRPC Retrieve(query, knowledge_base_id)
        RS->>RS: 向量检索 + BM25 + Rerank
        RS-->>CS: 返回相关文档片段
        CS->>CS: 注入 RAG 上下文
    end

    CS->>MA: StreamChat(model, messages)
    MA->>AI: HTTP SSE 请求

    loop 流式响应
        AI-->>MA: Token chunk
        MA-->>CS: ChatChunk
        CS-->>GW: SSE event
        GW-->>U: SSE event
    end

    CS->>DB: 保存消息
    CS->>K: 发送 token 用量事件
    K->>BS: 消费用量事件 → 计费
```

### 4.2 Agent 执行流

```mermaid
sequenceDiagram
    participant U as 用户
    participant AS as Agent Service
    participant CS as Chat Service
    participant TM as Tool Manager
    participant MCP as MCP Server
    participant SB as Sandbox
    participant DB as PostgreSQL
    participant ES as Elasticsearch

    U->>AS: 创建 Agent 任务
    AS->>DB: 保存任务 (status=created)
    AS->>AS: 状态机: created→planning

    AS->>CS: 请求任务规划
    CS-->>AS: 返回执行步骤列表

    loop 每个步骤
        AS->>AS: 状态机: planning→executing
        AS->>DB: 更新步骤状态

        alt 需要工具调用
            AS->>TM: 调用工具(tool_name, params)

            alt MCP 工具
                TM->>MCP: MCP 协议调用
                MCP-->>TM: 工具结果
            else 代码执行
                TM->>SB: 创建沙箱 + 执行代码
                SB-->>TM: 执行结果
            else AI 推理
                TM->>CS: Chat 请求
                CS-->>TM: AI 回答
            end

            TM-->>AS: 工具执行结果
        end

        AS->>CS: 请求下一步规划
        CS-->>AS: 下一步或完成
    end

    AS->>AS: 状态机: executing→success
    AS->>DB: 更新任务状态
    AS->>ES: 索引执行日志
```

### 4.3 RAG 文档处理流

```mermaid
sequenceDiagram
    participant U as 用户
    participant RS as RAG Service
    participant M as MinIO
    participant P as Doc Parser
    participant OCR as OCR Engine
    participant C as Chunker
    participant E as Embedder
    participant PG as PostgreSQL (pgvector)
    participant ES as Elasticsearch
    participant K as Kafka

    U->>RS: 上传文档
    RS->>M: 存储原始文件
    RS->>DB: 创建文档记录 (status=uploading)
    RS->>K: 发送文档处理事件

    K->>RS: 消费处理事件
    RS->>M: 下载文件
    RS->>P: 解析文档内容

    alt 扫描件 PDF
        P->>OCR: OCR 文字识别
        OCR-->>P: 识别文本
    end

    P-->>RS: 结构化内容

    RS->>C: 语义分块
    C-->>RS: Chunk 列表

    loop 每个 Chunk
        RS->>E: Embedding 向量化
        E-->>RS: 向量
        RS->>PG: 存储向量 + 元数据
        RS->>ES: 索引全文 (BM25)
    end

    RS->>DB: 更新文档状态 (status=ready)
```

---

## 5. 部署架构

### 5.1 Kubernetes 集群拓扑

```mermaid
graph TB
    subgraph "Kubernetes Cluster"
        subgraph "Ingress"
            INGRESS["Nginx Ingress Controller"]
        end

        subgraph "Application Namespace"
            GW["API Gateway<br/>Deployment × 3"]
            USER["User Service<br/>Deployment × 2"]
            CHAT["Chat Service<br/>Deployment × 3"]
            AGENT["Agent Service<br/>Deployment × 2"]
            RAG["RAG Service<br/>Deployment × 2"]
            IDE["IDE Service<br/>Deployment × 3"]
            WF["Workflow Service<br/>Deployment × 2"]
            MCP["MCP Service<br/>Deployment × 2"]
            DEPLOY["Deploy Service<br/>Deployment × 1"]
            BILLING["Billing Service<br/>Deployment × 2"]
            ADMIN["Admin Service<br/>Deployment × 1"]
            NOTIFY["Notification<br/>Deployment × 2"]
        end

        subgraph "Sandbox Namespace"
            SANDBOX_POOL["Sandbox Pool<br/>DaemonSet / Node Pool"]
        end

        subgraph "Data Namespace"
            PG["PostgreSQL<br/>StatefulSet (Primary + Replica)"]
            REDIS["Redis<br/>StatefulSet (Cluster)"]
            KAFKA["Kafka<br/>StatefulSet (3 Brokers)"]
            MINIO["MinIO<br/>StatefulSet (4 Nodes)"]
            ES["Elasticsearch<br/>StatefulSet (3 Nodes)"]
            TEMPORAL["Temporal<br/>Deployment"]
        end

        subgraph "Monitoring Namespace"
            PROM["Prometheus<br/>StatefulSet"]
            GRAFANA["Grafana<br/>Deployment"]
            LOKI["Loki<br/>StatefulSet"]
            JAEGER["Jaeger<br/>Deployment"]
            OTEL["OTel Collector<br/>DaemonSet"]
        end
    end

    INGRESS --> GW
    GW --> USER
    GW --> CHAT
    GW --> AGENT
    GW --> RAG
    GW --> IDE
    GW --> WF
    GW --> MCP
    GW --> DEPLOY
    GW --> BILLING
    GW --> ADMIN

    CHAT --> PG
    CHAT --> REDIS
    AGENT --> PG
    AGENT --> SANDBOX_POOL
    RAG --> PG
    RAG --> ES
    IDE --> MINIO
    WF --> TEMPORAL
    BILLING --> PG

    ALL_SVC["所有服务"] --> OTEL
    OTEL --> PROM
    OTEL --> LOKI
    OTEL --> JAEGER
```

### 5.2 网络架构

```mermaid
graph LR
    subgraph "Internet"
        USER["用户"]
    end

    subgraph "Edge"
        CF["Cloudflare<br/>CDN + WAF + DDoS"]
    end

    subgraph "Cloud VPC"
        subgraph "Public Subnet"
            LB["Load Balancer<br/>(L7)"]
        end

        subgraph "Private Subnet (App)"
            K8S["K8s Cluster"]
        end

        subgraph "Private Subnet (Data)"
            DB["PostgreSQL"]
            CACHE["Redis"]
            MQ["Kafka"]
            STORE["MinIO"]
            SEARCH["Elasticsearch"]
        end
    end

    USER --> CF --> LB --> K8S
    K8S --> DB
    K8S --> CACHE
    K8S --> MQ
    K8S --> STORE
    K8S --> SEARCH
```

---

## 6. 安全架构

```mermaid
graph TB
    subgraph "安全层"
        WAF["WAF<br/>Cloudflare"]
        TLS["TLS 1.3<br/>终端加密"]
        JWT["JWT 认证<br/>RS256"]
        RBAC["RBAC<br/>角色权限"]
        RLS["RLS<br/>行级安全"]
        ENC["加密存储<br/>AES-256"]
        AUDIT["审计日志<br/>全操作记录"]
        SANDBOX_SEC["沙箱安全<br/>gVisor + Seccomp"]
    end

    Request --> WAF --> TLS --> JWT --> RBAC --> RLS --> Response
    JWT --> AUDIT
    RBAC --> AUDIT
    ENC --> DB_ENCRYPTED["加密数据库"]
    SANDBOX_SEC --> CONTAINER["隔离容器"]
```

---

## 7. 事件驱动架构

```mermaid
graph LR
    subgraph "事件生产者"
        CHAT_P["Chat Service"]
        AGENT_P["Agent Service"]
        RAG_P["RAG Service"]
        USER_P["User Service"]
        DEPLOY_P["Deploy Service"]
    end

    subgraph "Kafka Topics"
        T1["chat.messages"]
        T2["agent.events"]
        T3["rag.documents"]
        T4["user.events"]
        T5["deploy.events"]
        T6["billing.usage"]
        T7["audit.logs"]
        T8["notification.send"]
    end

    subgraph "事件消费者"
        BILLING_C["Billing Service"]
        AUDIT_C["Admin (审计)"]
        NOTIFY_C["Notification Service"]
        SEARCH_C["Search Indexer"]
        MONITOR_C["Monitor Service"]
    end

    CHAT_P --> T1
    AGENT_P --> T2
    RAG_P --> T3
    USER_P --> T4
    DEPLOY_P --> T5

    T1 --> T6
    T1 --> T7
    T2 --> T7
    T4 --> T7
    T5 --> T8

    T6 --> BILLING_C
    T7 --> AUDIT_C
    T8 --> NOTIFY_C
    T3 --> SEARCH_C
    T1 --> MONITOR_C
    T2 --> MONITOR_C
```

---

## 8. AI 模型路由架构

```mermaid
graph TB
    subgraph "模型路由层"
        ROUTER["Model Router<br/>负载均衡 + 故障转移"]
        RATE_LIMITER["Rate Limiter<br/>按模型/用户限流"]
        CACHE_LAYER["Cache Layer<br/>语义缓存 (P2)"]
        COST_TRACKER["Cost Tracker<br/>实时成本追踪"]
    end

    subgraph "模型适配器"
        OA["OpenAI Adapter<br/>GPT-4o / GPT-4-turbo"]
        AN["Anthropic Adapter<br/>Claude 3.5 Sonnet/Opus"]
        GE["Gemini Adapter<br/>Gemini 1.5 Pro/Flash"]
        DS["DeepSeek Adapter<br/>DeepSeek-V3/Coder"]
        QW["Qwen Adapter<br/>Qwen-2.5"]
        OL["Ollama Adapter<br/>Llama / Mistral"]
    end

    subgraph "外部 AI 服务"
        OAI_API["OpenAI API"]
        ANT_API["Anthropic API"]
        GEM_API["Google Vertex AI"]
        DS_API["DeepSeek API"]
        QW_API["DashScope API"]
        LOCAL["本地 Ollama"]
    end

    Request --> RATE_LIMITER --> CACHE_LAYER --> ROUTER
    ROUTER --> OA --> OAI_API
    ROUTER --> AN --> ANT_API
    ROUTER --> GE --> GEM_API
    ROUTER --> DS --> DS_API
    ROUTER --> QW --> QW_API
    ROUTER --> OL --> LOCAL

    ROUTER --> COST_TRACKER
```

---

## 9. Sandbox 架构

```mermaid
graph TB
    subgraph "Sandbox Orchestrator"
        POOL_MGR["Pool Manager<br/>容器池管理"]
        SCHEDULER["Scheduler<br/>资源调度"]
        CLEANER["Cleaner<br/>清理回收"]
    end

    subgraph "Sandbox Runtime"
        subgraph "Container 1 (Python)"
            PY_RT["Python 3.12 Runtime"]
            PY_FS["/workspace (tmpfs)"]
        end
        subgraph "Container 2 (Go)"
            GO_RT["Go 1.22 Runtime"]
            GO_FS["/workspace (tmpfs)"]
        end
        subgraph "Container 3 (Node)"
            NODE_RT["Node.js 20 Runtime"]
            NODE_FS["/workspace (tmpfs)"]
        end
    end

    subgraph "Security"
        GVISOR["gVisor (内核隔离)"]
        SECCOMP["Seccomp (系统调用过滤)"]
        CGROUPS["cgroups v2 (资源限制)"]
        NETPOL["NetworkPolicy (网络隔离)"]
    end

    POOL_MGR --> SCHEDULER
    SCHEDULER --> PY_RT
    SCHEDULER --> GO_RT
    SCHEDULER --> NODE_RT

    PY_RT --> GVISOR
    GO_RT --> GVISOR
    NODE_RT --> GVISOR

    GVISOR --> SECCOMP
    GVISOR --> CGROUPS
    GVISOR --> NETPOL

    CLEANER --> PY_FS
    CLEANER --> GO_FS
    CLEANER --> NODE_FS
```

---

## 10. Workflow 执行架构

```mermaid
graph TB
    subgraph "前端"
        CANVAS["可视化画布<br/>React Flow"]
        NODE_PALETTE["节点面板"]
        PROPS_PANEL["属性面板"]
    end

    subgraph "Workflow Service"
        WF_API["Workflow API"]
        WF_PARSER["Workflow Parser<br/>DAG 解析"]
        WF_VALIDATOR["Validator<br/>DAG 验证"]
    end

    subgraph "Temporal"
        WF_EXEC["Workflow Execution"]
        ACTIVITIES["Activities"]
        RETRY["Retry Policy"]
        SCHEDULE["Cron Schedule"]
    end

    subgraph "Node Executors"
        AI_EXEC["AI Node<br/>→ Chat Service"]
        HTTP_EXEC["HTTP Node<br/>→ External API"]
        SQL_EXEC["SQL Node<br/>→ Database"]
        CODE_EXEC["Code Node<br/>→ Sandbox"]
        COND_EXEC["Condition Node<br/>→ 内部逻辑"]
    end

    CANVAS --> WF_API
    WF_API --> WF_PARSER --> WF_VALIDATOR
    WF_VALIDATOR --> WF_EXEC
    WF_EXEC --> ACTIVITIES
    ACTIVITIES --> AI_EXEC
    ACTIVITIES --> HTTP_EXEC
    ACTIVITIES --> SQL_EXEC
    ACTIVITIES --> CODE_EXEC
    ACTIVITIES --> COND_EXEC
    RETRY --> ACTIVITIES
    SCHEDULE --> WF_EXEC
```
