# OmniDev AI Platform — 技术选型对比分析

## 1. 后端语言

### 候选方案

| 维度 | Go | Rust | Java | Node.js | Python |
|------|-----|------|------|---------|--------|
| 并发模型 | goroutine (CSP) | async/await (Tokio) | Virtual Threads (Loom) | Event Loop | asyncio |
| 内存占用 | 低 | 极低 | 高 (JVM) | 中 | 中 |
| 启动速度 | 极快 | 极快 | 慢 (JVM) | 快 | 快 |
| 编译速度 | 快 | 慢 | 中 | N/A | N/A |
| 生态成熟度 | 高 | 中 | 极高 | 高 | 极高 |
| 微服务支持 | 优秀 | 良好 | 优秀 | 良好 | 一般 |
| gRPC 原生支持 | ✅ 原生 | ✅ 原生 | ✅ 原生 | ⚠️ 三方库 | ⚠️ 三方库 |
| K8s 生态 | 原生 (client-go) | 三方 | 好 | 好 | 好 |
| 学习曲线 | 低 | 高 | 中 | 低 | 低 |
| 招聘难度 | 中 | 高 | 低 | 低 | 低 |

### 决策：Go

**理由：**
1. **并发性能**：goroutine 轻量级并发，天然适合高并发 AI 流式场景
2. **部署简单**：静态二进制，无运行时依赖，Docker 镜像极小
3. **K8s 原生**：Kubernetes 本身就是 Go 写的，client-go 生态最成熟
4. **gRPC 原生**：Protocol Buffers + gRPC 是 Go 的一等公民
5. **编译快**：增量编译秒级，开发体验好
6. **内存低**：相同负载下内存占用是 Java 的 1/5-1/10

**风险与缓解：**
| 风险 | 缓解措施 |
|------|----------|
| 泛型支持弱 (Go 1.18+) | 使用 interface{} + 类型断言，代码生成 |
| 错误处理冗长 | 统一错误处理中间件，自定义 error 类型 |
| AI/ML 生态弱 | 通过 HTTP/gRPC 调用 Python 服务 |

---

## 2. Web 框架

### 候选方案

| 维度 | Gin | Echo | Fiber | Chi | Hertz (字节) |
|------|-----|------|-------|-----|-------------|
| 性能 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| 生态 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| 中间件 | 丰富 | 丰富 | 良好 | 良好 | 良好 |
| 文档 | 优秀 | 优秀 | 良好 | 良好 | 一般 |
| 社区活跃度 | 极高 | 高 | 高 | 中 | 中 |
| net/http 兼容 | ✅ | ✅ | ❌ (fasthttp) | ✅ | ❌ (自研) |

### 决策：Gin

**理由：** 最成熟、生态最好、性能足够、net/http 兼容（可复用中间件）

---

## 3. RPC 框架

### 候选方案

| 维度 | gRPC | Connect (Buf) | Twirp | REST (内部) |
|------|------|---------------|-------|-------------|
| 性能 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| 浏览器支持 | 需 grpc-web | 原生支持 | 原生 | 原生 |
| 代码生成 | protoc | buf generate | protoc | OpenAPI |
| 流式支持 | ✅ 双向流 | ✅ | ❌ | SSE/WS |
| 生态 | 极高 | 高 | 中 | 极高 |
| 学习曲线 | 中 | 低 | 低 | 低 |

### 决策：gRPC (内部) + REST (外部)

**理由：**
1. **gRPC 用于服务间通信**：高性能、强类型、流式支持
2. **REST 用于前端/API**：浏览器兼容、开发者友好
3. **API Gateway** 做协议转换（REST ↔ gRPC）

---

## 4. 数据库

### 候选方案

| 维度 | PostgreSQL | MySQL | CockroachDB | TiDB | MongoDB |
|------|-----------|-------|-------------|------|---------|
| ACID | ✅ 完整 | ✅ 基本 | ✅ 完整 | ✅ 完整 | ⚠️ 有限 |
| 向量搜索 | pgvector | ❌ | ❌ | ❌ | Atlas Vector |
| JSON 支持 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| 全文搜索 | 内置 (tsvector) | 内置 (全文索引) | 有限 | 有限 | Atlas Search |
| 分区 | ✅ 原生 | ✅ 原生 | ✅ 自动 | ✅ 自动 | ✅ Shard |
| 扩展生态 | 极高 | 中 | 中 | 中 | 中 |
| 运维复杂度 | 中 | 低 | 高 | 高 | 中 |
| 云托管 | 所有云 | 所有云 | Cockroach Cloud | PingCAP Cloud | Atlas |

### 决策：PostgreSQL 16 + pgvector

**理由：**
1. **向量搜索一体化**：pgvector 扩展支持 HNSW/IVFFlat 索引，无需额外向量数据库
2. **JSONB 灵活存储**：Agent 配置、工作流定义等半结构化数据
3. **全文搜索内置**：tsvector 支持中英文分词（配合 zhparser）
4. **扩展丰富**：PostGIS（地理）、pg_cron（定时）、pg_stat_statements（性能）
5. **RLS 行级安全**：原生多租户隔离

**配套扩展：**
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";      -- UUID 生成
CREATE EXTENSION IF NOT EXISTS "pgvector";        -- 向量搜索
CREATE EXTENSION IF NOT EXISTS "pg_trgm";         -- 模糊搜索
CREATE EXTENSION IF NOT EXISTS "zhparser";        -- 中文分词
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements"; -- 查询统计
CREATE EXTENSION IF NOT EXISTS "pg_cron";         -- 定时任务
```

---

## 5. 缓存

### 候选方案

| 维度 | Redis | Dragonfly | KeyDB | Valkey |
|------|-------|-----------|-------|--------|
| 性能 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| 内存效率 | 中 | 极高 | 高 | 中 |
| 集群支持 | ✅ | ✅ | ✅ | ✅ |
| 持久化 | RDB+AOF | RDB+AOF | RDB+AOF | RDB+AOF |
| 生态 | 极高 | 中 | 中 | 中 |
| 兼容性 | - | Redis 协议 | Redis 协议 | Redis 协议 |
| 许可证 | SSPLv1 | BSL 1.1 | BSD | BSD |

### 决策：Redis 7 (Cluster 模式)

**理由：** 生态最成熟，所有框架/驱动原生支持，SSPL 对自托管无影响

**用途分配：**
| 用途 | 数据结构 | TTL |
|------|----------|-----|
| 会话存储 | String (JWT Session) | 7d |
| Token 黑名单 | String | 15min |
| API 限流 | Sorted Set | 滑动窗口 |
| 热点数据缓存 | Hash/String | 5-30min |
| 分布式锁 | String (SET NX EX) | 30s |
| 实时计数器 | HyperLogLog/Counter | 按需 |
| 发布订阅 | Pub/Sub Channel | 实时 |

---

## 6. 消息队列

### 候选方案

| 维度 | Kafka | RabbitMQ | NATS | Pulsar | Redpanda |
|------|-------|----------|------|--------|----------|
| 吞吐量 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| 延迟 | 中 | 低 | 极低 | 中 | 低 |
| 持久化 | ✅ | ✅ | JetStream | ✅ | ✅ |
| 消息顺序 | 分区有序 | 队列有序 | 主题有序 | 分区有序 | 分区有序 |
| Exactly-once | ✅ | ❌ | ❌ | ✅ | ✅ |
| 生态 | 极高 | 高 | 中 | 高 | 中 |
| 运维复杂度 | 高 | 中 | 低 | 高 | 中 |

### 决策：Kafka (KRaft 模式，无 ZooKeeper)

**理由：**
1. **高吞吐**：日志型存储，适合大量事件流
2. **持久化**：消息可回溯，支持重放
3. **Exactly-once**：关键业务（计费、审计）需要
4. **流处理**：未来可扩展 Kafka Streams 做实时计算
5. **KRaft**：去掉 ZooKeeper 依赖，降低运维复杂度

**Topic 规划：**
| Topic | 分区 | 保留期 | 用途 |
|-------|------|--------|------|
| chat.messages | 12 | 7d | 对话消息事件 |
| agent.events | 12 | 7d | Agent 执行事件 |
| rag.documents | 6 | 3d | 文档处理事件 |
| billing.usage | 6 | 30d | 用量计费事件 |
| audit.logs | 6 | 90d | 审计日志事件 |
| notification.send | 6 | 3d | 通知发送事件 |
| workflow.events | 6 | 7d | 工作流执行事件 |

---

## 7. 对象存储

### 候选方案

| 维度 | MinIO | Ceph RGW | SeaweedFS | AWS S3 |
|------|-------|----------|-----------|--------|
| S3 兼容 | ✅ 完整 | ✅ 良好 | ✅ 良好 | ✅ 原生 |
| 性能 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| 部署简单 | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ | N/A |
| 运维复杂度 | 低 | 高 | 中 | 无 |
| 多租户 | ✅ | ✅ | ✅ | ✅ |
| 许可证 | AGPLv3 | LGPL | Apache 2.0 | 商业 |

### 决策：MinIO

**理由：** S3 兼容最好、部署最简单、Go 编写与技术栈统一、AGPLv3 对自托管无影响

**存储桶规划：**
| Bucket | 内容 | 生命周期 |
|--------|------|----------|
| user-uploads | 用户上传文件 | 永久 |
| rag-documents | RAG 文档原始文件 | 永久 |
| rag-processed | RAG 处理后的文件 | 永久 |
| project-files | 项目代码文件 | 永久 |
| avatars | 用户头像 | 永久 |
| temp | 临时文件 | 7d 自动清理 |
| backups | 数据库备份 | 30d 跨区域 |

---

## 8. 搜索引擎

### 候选方案

| 维度 | Elasticsearch | OpenSearch | Meilisearch | Typesense | Zinc |
|------|--------------|------------|-------------|-----------|------|
| 全文搜索 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| 分布式 | ✅ | ✅ | ❌ | ❌ | ❌ |
| 中文分词 | IK/THULAC | IK | 内置 | 内置 | 内置 |
| 向量搜索 | ✅ kNN | ✅ kNN | ✅ | ✅ | ❌ |
| 运维复杂度 | 高 | 高 | 低 | 低 | 低 |
| 资源消耗 | 高 | 高 | 低 | 低 | 低 |

### 决策：Elasticsearch 8

**理由：**
1. **全文搜索 + 向量搜索一体化**：可作为 RAG 的辅助检索
2. **中文分词成熟**：IK 分词器 + 自定义词典
3. **Agent 日志**：结构化日志存储和分析
4. **审计日志**：大量日志的高效检索

**注意：** 主要全文搜索用 PostgreSQL tsvector，ES 用于复杂场景和日志分析

---

## 9. 工作流引擎

### 候选方案

| 维度 | Temporal | Cadence | Argo Workflows | Airflow | Prefect |
|------|----------|---------|----------------|---------|---------|
| 可靠性 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| 可视化 | ✅ Web UI | ✅ Web UI | ✅ Argo UI | ✅ Web UI | ✅ Cloud |
| Go SDK | ✅ 原生 | ✅ 原生 | ⚠️ YAML | ❌ Python | ❌ Python |
| 长运行任务 | ✅ | ✅ | ✅ | ❌ | ✅ |
| 人工审批 | ✅ | ✅ | ✅ | ❌ | ✅ |
| 版本管理 | ✅ | ✅ | ✅ | ✅ | ✅ |
| 社区 | 活跃 | 中 | 活跃 | 极活跃 | 活跃 |
| 运维复杂度 | 中 | 中 | 中 | 高 | 低 |

### 决策：Temporal

**理由：**
1. **Go 原生 SDK**：与后端技术栈完美匹配
2. **确定性重放**：保证长时间运行的工作流可靠性
3. **可视化**：Web UI 查看工作流执行历史
4. **Activity 重试**：细粒度的重试策略
5. **版本管理**：工作流代码版本化

---

## 10. 前端框架

### 候选方案

| 维度 | Next.js 15 | Remix | Nuxt 3 | SvelteKit | Astro |
|------|-----------|-------|--------|-----------|-------|
| React 生态 | ✅ | ✅ | ❌ Vue | ❌ Svelte | ✅ 部分 |
| SSR/SSG | ✅ | ✅ | ✅ | ✅ | ✅ |
| API Routes | ✅ | ✅ | ✅ | ✅ | ✅ |
| App Router | ✅ | ✅ | - | - | - |
| 流式渲染 | ✅ RSC | ✅ | ✅ | ✅ | ❌ |
| 生态规模 | 极大 | 大 | 大 | 中 | 中 |
| 部署 | Vercel/自托管 | 自托管 | 自托管 | 自托管 | 自托管 |

### 决策：Next.js 15 (App Router)

**理由：**
1. **React 生态**：最大的组件库和社区支持
2. **RSC (React Server Components)**：服务端渲染，减少客户端 JS
3. **App Router**：嵌套路由、布局、Loading UI
4. **流式 SSR**：AI 流式输出天然适配
5. **TanStack Query 集成**：服务端数据获取最佳实践

---

## 11. UI 组件库

### 候选方案

| 维度 | Shadcn/ui | Radix + Tailwind | Ant Design | MUI | Chakra UI |
|------|----------|------------------|------------|-----|-----------|
| 可定制性 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| 代码所有权 | 完全控制 | 需包装 | npm 依赖 | npm 依赖 | npm 依赖 |
| 包体积 | 极小 | 小 | 大 | 大 | 中 |
| 设计美感 | 现代 | 现代 | 企业级 | Material | 现代 |
| Tailwind 原生 | ✅ | ✅ | ❌ | ❌ | ❌ |

### 决策：Shadcn/ui + Tailwind CSS

**理由：** 代码复制到项目中（非 npm 依赖）、完全可定制、Tailwind 原生、现代设计

---

## 12. 代码编辑器

### 候选方案

| 维度 | Monaco Editor | CodeMirror 6 | Ace Editor |
|------|--------------|-------------|------------|
| VS Code 兼容 | ✅ 同源 | ❌ | ❌ |
| 语言支持 | 极多 | 多 | 多 |
| 性能 | 中 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| 包体积 | 大 (~2MB) | 小 (~200KB) | 中 |
| 扩展性 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| AI 补全集成 | LSP | LSP | 自定义 |

### 决策：Monaco Editor

**理由：** VS Code 同源引擎、语言支持最全、用户最熟悉、LSP 协议支持

---

## 13. 终端模拟

### 候选方案

| 维度 | xterm.js | hterm | Hyper |
|------|----------|-------|-------|
| 性能 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| 功能完整度 | 极高 | 高 | 中 |
| WebGL 渲染 | ✅ | ❌ | ❌ |
| 插件生态 | 丰富 | 少 | 中 |
| 维护状态 | 活跃 | 停滞 | 停滞 |

### 决策：xterm.js + xterm-addon-webgl

**理由：** 事实标准、WebGL 渲染高性能、插件丰富

---

## 14. AI 模型接入

### 统一 Adapter 模式

```
┌──────────────────────────────────────┐
│          Unified AI Adapter          │
├──────────┬───────────┬───────────────┤
│ OpenAI   │ Anthropic │  Gemini       │
│ Adapter  │ Adapter   │  Adapter      │
├──────────┼───────────┼───────────────┤
│ DeepSeek │ Qwen      │  Ollama       │
│ Adapter  │ Adapter   │  Adapter      │
└──────────┴───────────┴───────────────┘
```

| Provider | 模型 | 接口 | 特点 |
|----------|------|------|------|
| OpenAI | GPT-4o, GPT-4-turbo | REST (OpenAI 格式) | 行业标准格式 |
| Anthropic | Claude 3.5 Sonnet/Opus | REST (Messages API) | 长上下文、安全 |
| Google | Gemini 1.5 Pro/Flash | REST (Vertex AI) | 多模态、长上下文 |
| DeepSeek | DeepSeek-V3/Coder | REST (OpenAI 兼容) | 性价比高 |
| Alibaba | Qwen-2.5 | REST (DashScope) | 中文优秀 |
| Ollama | Llama/Mistral/本地模型 | REST (本地) | 隐私、无费用 |

---

## 15. 容器与编排

| 维度 | Docker + K8s | Podman + K8s | Docker Swarm |
|------|-------------|-------------|--------------|
| 编排能力 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| 生态 | 极大 | 大 | 小 |
| 学习曲线 | 中 | 中 | 低 |
| 生产就绪 | ✅ | ✅ | ⚠️ |

### 决策：Docker (开发/沙箱) + Kubernetes (生产编排)

---

## 16. 技术栈总览

```
┌─────────────────────────────────────────────────────────┐
│                      Frontend                            │
│  Next.js 15 │ React 19 │ Tailwind │ Shadcn/ui           │
│  Monaco Editor │ xterm.js │ TanStack Query │ Zustand     │
├─────────────────────────────────────────────────────────┤
│                      API Layer                           │
│  Gin (REST Gateway) │ gRPC (内部通信) │ WebSocket        │
├─────────────────────────────────────────────────────────┤
│                    Backend Services                      │
│  Go 1.22+ │ Wire (DI) │ Zap (日志) │ Viper (配置)      │
├─────────────────────────────────────────────────────────┤
│                      AI Layer                            │
│  OpenAI SDK │ Anthropic SDK │ Ollama │ 统一 Adapter      │
├─────────────────────────────────────────────────────────┤
│                     Data Layer                           │
│  PostgreSQL 16 │ pgvector │ Redis 7 │ Elasticsearch 8   │
├─────────────────────────────────────────────────────────┤
│                   Messaging & Queue                      │
│  Kafka (KRaft) │ Temporal                                │
├─────────────────────────────────────────────────────────┤
│                     Storage                              │
│  MinIO (对象存储) │ Git (代码版本)                        │
├─────────────────────────────────────────────────────────┤
│                   Infrastructure                         │
│  Docker │ Kubernetes │ Helm │ Terraform                  │
├─────────────────────────────────────────────────────────┤
│                   Observability                          │
│  Prometheus │ Grafana │ Loki │ Jaeger │ OpenTelemetry    │
├─────────────────────────────────────────────────────────┤
│                      CI/CD                               │
│  GitHub Actions │ ArgoCD │ Trivy                         │
└─────────────────────────────────────────────────────────┘
```
