# OmniDev — 需求分析（当前实现）

## 1. 产品定位

OmniDev 是面向团队的 AI 知识与编码工作台，当前围绕三条主线提供能力：

1. **Chat**：多模型对话、流式输出、附件、图片生成、个人模型配置和外部 MCP 工具。
2. **Knowledge**：团队资料的知识库化管理、文档解析与向量化、混合检索及对话内 RAG。
3. **Background Agent**：异步执行编码与文件任务，在 Docker 沙箱中运行代码，并支持轮询和取消。

系统采用模块化单体：`apps/gateway` 是唯一的后端进程，`apps/web` 是 Next.js 前端，`packages/go-common` 提供共享能力，同级目录 `../knowledge-engine` 以 Go module 方式嵌入。核心资源当前按用户隔离；组织和成员表已存在，但完整组织管理与团队共享流程尚未开放。

## 2. 用户与访问需求

| 场景 | 当前需求与验收口径 |
|---|---|
| 注册与登录 | 支持邮箱/密码注册、登录、JWT 刷新；密码使用 bcrypt 哈希。 |
| OAuth | 配置 GitHub 或 Google OAuth 后可完成跳转、回调及账号关联。 |
| 个人资料 | 用户可查询和修改昵称、头像、简介及个人设置。 |
| API Key | 用户可创建、列出和吊销自己的 Key；数据库保存哈希而非明文。 |
| 资源隔离 | 受保护 API 使用 JWT；会话、知识库、Agent 等按 `user_id` 校验所有权。 |
| 个人模型配置 | 用户可维护 provider、Base URL、模型列表和密钥（BYOK），设默认值并测试连接。 |

## 3. Chat 需求

### 3.1 对话与模型

- 支持会话创建、查询、更新、删除以及消息分页查询。
- 支持普通响应和 SSE 流式响应；保存用户与助手消息、模型、延迟和 Token 信息。
- 已接入 OpenAI、DeepSeek、Qwen、Anthropic；仅注册已配置密钥的全局 provider，用户配置优先于全局配置。
- 上下文包含会话 system prompt 和最近 20 条消息；首条消息可生成会话标题。
- 提供模型列表和图片生成接口。图片生成与文本对话是不同调用路径。

### 3.2 附件

- 聊天附件上传到 MinIO，记录文件元数据并与消息关联。
- 单文件上限为 20 MiB；允许常见图片、PDF、Word、纯文本和 Markdown 类型。
- 图片以 data URL 形式进入多模态消息；文档通过 Apache Tika 提取文本后注入上下文。
- 用户可下载或删除自己的附件。MinIO 不可用时上传功能不可用，不应假设有本地聊天附件存储。

### 3.3 外部 MCP 工具

- 网关作为 MCP 客户端，通过 Streamable HTTP + JSON-RPC 2.0 连接外部服务器。
- 使用 `MCP_ENABLED` 和 JSON 数组 `MCP_SERVERS` 配置；启动时初始化并获取工具目录，失败的服务器会跳过。
- 普通与流式聊天均可执行工具循环，最多 5 轮；每次工具结果注入模型前截断到 8000 个 rune。
- 工具调用适用于 OpenAI 兼容适配器：OpenAI、DeepSeek、Qwen。
- Anthropic 的 `tool_use`/`tool_result` 映射尚未实现，因此该适配器不接收 MCP 工具目录。

## 4. Knowledge 需求

| 场景 | 当前需求与验收口径 |
|---|---|
| 知识库管理 | 创建、列表、查询、更新、删除个人知识库；配置分块大小和重叠量。 |
| 文档管理 | 上传、列出和删除文档；原文件可存本地或 MinIO。 |
| 异步处理 | 上传后后台执行解析、语义分块、批量 embedding 和入库，文档状态可推进到 ready/failed。 |
| 向量检索 | PostgreSQL + pgvector 保存向量，使用余弦距离和 HNSW 索引（维度不超过 2000 时）。 |
| 混合搜索 | 合并向量检索与 PostgreSQL 全文检索，使用 RRF 排序；全文失败时有 LIKE 回退。 |
| 缓存 | 搜索结果按知识库版本、模型和查询参数缓存，优先使用 Redis，失败时可用内存缓存。 |
| RAG 对话 | 请求可携带多个 `knowledge_base_ids`；每库取 Top 3，最多注入 10 条结果及来源编号。 |

知识引擎是独立可复用的 Go module，也提供自己的 HTTP 传输和 stdio MCP server；在 OmniDev 主程序中不单独部署服务，而与网关共享 PostgreSQL 连接池。

## 5. Background Agent 需求

- 用户可创建、查询、更新、删除 Agent，配置 system prompt、启用工具和最大步数（默认 20）。
- `POST /api/v1/agents/:id/run` 创建异步 run；客户端通过 run API 轮询状态、结果和步骤，可取消规划或执行中的任务。
- 状态类型包括 `created/planning/executing/waiting_tool/success/failed/cancelled`；当前主流程实际写入除 `waiting_tool` 外的状态。
- 规划器使用 OpenAI 兼容模型逐步决定 think、tool call、code execution 或最终 response。
- 内置工具为工作区文件操作、文本搜索、计算器和代码执行；文件与搜索路径限制在本次 run 的工作区。
- 代码语言为 Python、JavaScript、Go、Shell。每次执行使用一次性 Docker 容器，默认超时 120 秒。
- 沙箱默认无网络、只读根文件系统、可写 `/workspace`、64 MiB `/tmp` tmpfs，去除全部 capabilities，启用 no-new-privileges，并限制 CPU、内存和 PID。
- stdout 与 stderr 分别最多保留 1 MiB，截断会在结果中标记。

## 6. 核心使用流程

1. **知识问答**：上传资料 → 等待文档 ready → 在聊天中选择知识库 → 检索结果注入 system context → 模型生成带上下文回答。
2. **编码任务**：创建 Agent → 提交 task → 后端异步规划并操作工作区/运行代码 → 轮询步骤与结果 → 必要时取消。
3. **个人模型接入**：添加 provider 配置和密钥 → 测试连接 → 设为默认或在会话中显式选择模型。

## 7. 非目标、已下线与未实现

**非目标/已下线**：在线 IDE、可视化工作流画布、计费与订阅、完整监控平台、一键部署、通知服务、微服务编排平台。这些能力不属于当前产品验收范围。

**未实现/规划中**：Agent 自动提交 PR；Anthropic 工具调用映射。组织管理、Prompt 模板目前只有数据结构，不应视为已交付的完整产品功能。
