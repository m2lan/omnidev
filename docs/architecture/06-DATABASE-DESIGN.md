# OmniDev — 数据库设计（当前实现）

## 1. 范围与迁移来源

系统使用**一个 PostgreSQL 数据库**，同时保存业务数据和 pgvector 向量，不按微服务拆库。

| 来源 | 内容 | 执行方式 |
|---|---|---|
| `apps/gateway/migrations/001–009` | 用户、OAuth、API Key、组织、AI 配置、聊天、附件、Agent | `make db-migrate` 或 Compose 的 migrate 容器。 |
| `../knowledge-engine/migrations/001–002` | 知识库、文档、chunk、pgvector、旧维度兼容 | knowledge-engine 启动时默认自动执行。 |

通用习惯是 UUID 主键（`gen_random_uuid()`）、`TIMESTAMPTZ` 时间戳、JSONB 配置/元数据，以及部分表通过 `deleted_at` 软删除。并非所有表都有外键、`updated_at` 或软删除；没有启用 PostgreSQL RLS，所有权由服务层检查。

## 2. 用户、组织与模型配置

### 2.1 `users`（001）

- 关键列：`id`、`email`、`email_verified`、`password_hash`、`nickname`、`avatar_url`、`bio`。
- 账户控制：`role`、`status`、`last_login_at`、`last_login_ip`。
- 扩展与时间：`settings`、`metadata`、`created_at`、`updated_at`、`deleted_at`。
- `email` 在未软删除记录中唯一；`update_updated_at_column()` 触发器维护 `updated_at`。

### 2.2 `oauth_connections`（002）

- `user_id` 外键关联 `users`，删除用户时级联删除。
- `provider + provider_uid` 唯一；另存 `access_token`、`refresh_token`、`expires_at`、`scope`、`raw_profile` 和时间戳。

### 2.3 `api_keys`（003）

- 关键列：`user_id`、`name`、`key_hash`、`key_prefix`、`scopes TEXT[]`。
- 生命周期：`expires_at`、`last_used_at`、`last_used_ip`、`status`、`created_at`、`updated_at`。
- 活跃 Key 的哈希唯一；明文 Key 不入库。

### 2.4 `organizations` 与 `org_members`（004）

- `organizations`：`name`、唯一 `slug`、`description`、`avatar_url`、`plan`、`settings`、`metadata`、时间戳和 `deleted_at`。
- `org_members`：`org_id`、`user_id`、`role`、`invited_by`、`joined_at`；`(org_id,user_id)` 唯一，组织/用户删除可级联。
- 这两张表是当前数据模型，不代表组织管理 API 已交付；`plan` 也不表示存在计费系统。

### 2.5 `user_ai_configs`（005）

- 归属与协议：`user_id`、`provider`、`display_name`、`base_url`、`protocol`（默认 openai）。
- 模型与密钥：`api_key`、`models JSONB`、`is_default`、`is_active`。
- 包含 `created_at/updated_at/deleted_at`；`api_key` 是否加密取决于 `ENCRYPTION_KEY`，不是数据库层透明加密。

## 3. Chat 与附件

### 3.1 `models`（006）

- `provider + model_id` 唯一，另有 `display_name`、`description`。
- 能力字段：`context_window`、`max_output`、`supports_streaming`、`supports_vision`、`supports_tools`。
- 其他字段：`input_price`、`output_price`、`is_active`、`config` 和时间戳。价格列不构成账单或扣费功能。

### 3.2 `conversations`（006）

- 归属：`user_id`、可选 `org_id`；`model_id` 可引用 `models(id)`。
- 内容：`title`、`system_prompt`、`settings`、`status`、`pinned`、`tags`、`message_count`、`metadata`。
- 包含 `created_at/updated_at/deleted_at`；常用索引覆盖用户、组织、状态和创建时间。

### 3.3 `messages`（006，按月分区）

- 主键是 **`(id, created_at)`**，因为表按 `created_at` 做 RANGE 分区。
- 关键列：`conversation_id`、`role`、`content`、字符串 `model_id`。
- 执行信息：`token_input`、`token_output`、`latency_ms`、`tool_calls`、`tool_call_id`、`parent_id`、`metadata`。
- 迁移创建当月及后续 3 个月的 `messages_YYYY_MM` 分区，并建立会话/时间、创建时间和角色索引。
- 没有自动分区调度器；后续月份必须通过运维或新迁移提前创建，避免超出分区范围后写入失败。

### 3.4 `prompt_templates`（006）

- `user_id`、可选 `org_id`、`title`、`content`、`description`、`category`、`tags`、`variables`。
- 另有 `visibility`、`version`、`fork_from`、`use_count`、`like_count`、`metadata`、时间戳和 `deleted_at`。
- 当前只有表结构，没有对应 Prompt 模板业务路由。

### 3.5 `attachments`（007–008）

- 归属：`user_id` 引用 `users`，可关联 `conversation_id` 和 `message_id`。
- 文件信息：`filename`、`mime_type`、`file_size`、`storage_key`、`storage_url`、`thumbnail_key`、`width`、`height`、`metadata`。
- 时间为 `created_at/deleted_at`；008 将 `storage_key/thumbnail_key` 扩到 1024 字符、`storage_url` 改为 TEXT，以容纳预签名 URL。
- `message_id` 未对分区 `messages` 建外键，消息关联由应用维护。

## 4. Background Agent

### 4.1 `agents`（009）

- 归属与定义：`user_id`、可选 `org_id`、`name`、`description`、`avatar_url`、`system_prompt`、可选 `model_id`。
- 配置：`tools JSONB`、`mcp_servers JSONB`、`config JSONB`（默认 `max_steps=20`）。
- 扩展：`visibility`、`is_template`、`template_category`、`metadata`、时间戳和 `deleted_at`。

### 4.2 `agent_runs`（009）

- `agent_id` 引用 `agents`，同时保存 `user_id` 和 `task`。
- 状态与结果：`status`、`result`、`error`、`total_steps`、`completed_steps`。
- 计量与时间：`token_input`、`token_output`、`cost`、`started_at`、`completed_at`、`metadata`、`created_at/updated_at`。
- 状态类型为 `created/planning/executing/waiting_tool/success/failed/cancelled`；`cost` 字段不表示有计费模块。

### 4.3 `agent_steps`（009）

- `run_id` 引用 `agent_runs`，`(run_id,step_number)` 唯一。
- 步骤：`type`、`content`、`tool_name`、`tool_input/tool_output JSONB`、`status`、`error`。
- 计量：`token_input/token_output/latency_ms`，以及 `started_at/completed_at/created_at`。

## 5. Knowledge Engine 表

### 5.1 `knowledge_bases`

- 归属：`user_id`、可选 `org_id/template_id`；基本字段为 `name`、`description`、`status`。
- 向量配置：`embedding_model`、`embedding_dimensions`（默认 1536）。
- 分块配置：`chunk_size=512`、`chunk_overlap=50`；统计字段为 `doc_count`、`chunk_count`、`total_tokens`、`total_size`。
- 另有 `settings`、`created_at/updated_at/deleted_at`；查询时校验知识库模型/维度与当前引擎配置兼容。

### 5.2 `documents`

- `knowledge_base_id` 外键级联关联知识库；文件字段为 `filename`、`file_type`、`file_size`、`file_url`。
- 处理字段：`status`（初始 uploading）、`error`、`chunk_count`、`total_tokens`、`processed_at`。
- 扩展字段：`tags TEXT[]`、`metadata`、`created_at/updated_at/deleted_at`。

### 5.3 `document_chunks`

- 关联：`document_id`、`knowledge_base_id` 均带级联外键，`(document_id,chunk_index)` 唯一。
- 内容：`content`、`content_length`、`token_count`、`start_page/end_page`、`heading`、`metadata`、`created_at`。
- 检索：默认 `embedding vector(1536)` 和 `content_tsv tsvector`；触发器优先使用 chinese 配置，不可用时回退 simple。
- 默认向量列有 HNSW cosine 索引，`content_tsv` 有 GIN 索引。
- 非 1536 维度时，启动逻辑增加 `embedding_<dimensions> vector(N)` 及相应 HNSW 索引；维度大于 2000 时不建 HNSW。每个知识库仍记录自己的模型与维度。

知识迁移 002 会为旧 RAG 数据回填实际向量维度，避免旧库被错误标记为 1536 维。

## 6. 一致性与维护要求

- 多步写入需要一致性时使用事务；不要假设所有 `user_id/org_id/message_id` 都有数据库外键。
- 软删除查询必须沿用 repository 的 `deleted_at IS NULL` 条件；不要另建绕过所有权的跨用户查询。
- 新增 gateway 表使用顺序迁移并提供 up/down；不要修改已部署迁移。知识表变更放入 knowledge-engine 自己的迁移体系。
- 生产维护需关注未来消息分区、数据库备份、向量索引和连接池容量；当前没有自动分区、读写分离或跨地域复制实现。
