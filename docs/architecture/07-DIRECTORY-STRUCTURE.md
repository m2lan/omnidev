# OmniDev AI Platform — 项目目录结构

## 1. Monorepo 总览

```
omnidev/
├── .github/                    # GitHub Actions CI/CD
│   ├── workflows/
│   │   ├── ci.yml
│   │   ├── cd-staging.yml
│   │   ├── cd-production.yml
│   │   └── release.yml
│   ├── CODEOWNERS
│   └── pull_request_template.md
│
├── .claude/                    # Claude Code 配置
│   ├── CLAUDE.md
│   └── settings.json
│
├── apps/                       # 应用层（可独立部署）
│   ├── web/                    # 前端 Next.js 应用
│   ├── gateway/                # API Gateway (Go)
│   ├── services/               # 后端微服务
│   └── workers/                # 后台工作者
│
├── packages/                   # 共享包（库）
│   ├── proto/                  # Protobuf 定义
│   ├── go-common/              # Go 公共库
│   ├── ts-common/              # TypeScript 公共库
│   ├── ui/                     # UI 组件库 (Shadcn)
│   └── config/                 # 共享配置
│
├── deploy/                     # 部署配置
│   ├── docker/
│   ├── helm/
│   ├── terraform/
│   └── k8s/
│
├── scripts/                    # 工具脚本
├── docs/                       # 文档
├── tools/                      # 开发工具
│
├── docker-compose.yml          # 本地开发环境
├── Makefile                    # 全局构建命令
├── go.work                     # Go workspace
├── turbo.json                  # Turborepo 配置
├── package.json                # 根 package.json
└── README.md
```

---

## 2. 前端应用 `apps/web/`

```
apps/web/
├── public/
│   ├── icons/
│   ├── images/
│   └── favicon.ico
│
├── src/
│   ├── app/                    # Next.js App Router
│   │   ├── layout.tsx          # 根布局
│   │   ├── page.tsx            # 首页
│   │   ├── (auth)/             # 认证相关页面
│   │   │   ├── login/
│   │   │   │   └── page.tsx
│   │   │   ├── register/
│   │   │   │   └── page.tsx
│   │   │   └── layout.tsx
│   │   │
│   │   ├── (dashboard)/        # 仪表盘（需登录）
│   │   │   ├── layout.tsx      # 侧边栏 + 顶栏布局
│   │   │   ├── page.tsx        # Dashboard 首页
│   │   │   │
│   │   │   ├── chat/           # AI Chat 模块
│   │   │   │   ├── page.tsx    # 会话列表
│   │   │   │   ├── [id]/
│   │   │   │   │   └── page.tsx # 单个会话
│   │   │   │   └── prompts/    # Prompt 管理
│   │   │   │       └── page.tsx
│   │   │   │
│   │   │   ├── agent/          # Agent 模块
│   │   │   │   ├── page.tsx    # Agent 列表
│   │   │   │   ├── [id]/
│   │   │   │   │   └── page.tsx # Agent 详情
│   │   │   │   ├── runs/       # 执行记录
│   │   │   │   │   └── page.tsx
│   │   │   │   └── create/
│   │   │   │       └── page.tsx
│   │   │   │
│   │   │   ├── knowledge/      # RAG 知识库
│   │   │   │   ├── page.tsx    # 知识库列表
│   │   │   │   ├── [id]/
│   │   │   │   │   ├── page.tsx
│   │   │   │   │   └── documents/
│   │   │   │   │       └── page.tsx
│   │   │   │   └── upload/
│   │   │   │       └── page.tsx
│   │   │   │
│   │   │   ├── ide/            # 在线 IDE
│   │   │   │   ├── page.tsx    # 项目列表
│   │   │   │   └── [id]/
│   │   │   │       ├── page.tsx # IDE 主界面
│   │   │   │       └── terminal/
│   │   │   │           └── page.tsx
│   │   │   │
│   │   │   ├── workflow/       # 工作流
│   │   │   │   ├── page.tsx
│   │   │   │   ├── [id]/
│   │   │   │   │   ├── page.tsx
│   │   │   │   │   └── editor/
│   │   │   │   │       └── page.tsx
│   │   │   │   └── runs/
│   │   │   │       └── page.tsx
│   │   │   │
│   │   │   ├── mcp/            # MCP Server
│   │   │   │   ├── page.tsx
│   │   │   │   └── [id]/
│   │   │   │       └── page.tsx
│   │   │   │
│   │   │   ├── deploy/         # 部署
│   │   │   │   ├── page.tsx
│   │   │   │   └── [id]/
│   │   │   │       └── page.tsx
│   │   │   │
│   │   │   ├── monitoring/     # 监控
│   │   │   │   ├── page.tsx
│   │   │   │   ├── logs/
│   │   │   │   │   └── page.tsx
│   │   │   │   └── traces/
│   │   │   │       └── page.tsx
│   │   │   │
│   │   │   ├── billing/        # 计费
│   │   │   │   ├── page.tsx
│   │   │   │   ├── usage/
│   │   │   │   │   └── page.tsx
│   │   │   │   └── invoices/
│   │   │   │       └── page.tsx
│   │   │   │
│   │   │   └── settings/       # 设置
│   │   │       ├── page.tsx
│   │   │       ├── profile/
│   │   │       │   └── page.tsx
│   │   │       ├── api-keys/
│   │   │       │   └── page.tsx
│   │   │       └── organization/
│   │   │           └── page.tsx
│   │   │
│   │   ├── admin/              # 管理后台
│   │   │   ├── layout.tsx
│   │   │   ├── page.tsx
│   │   │   ├── users/
│   │   │   ├── models/
│   │   │   ├── billing/
│   │   │   ├── audit/
│   │   │   └── settings/
│   │   │
│   │   └── api/                # Next.js API Routes (BFF)
│   │       ├── auth/
│   │       ├── chat/
│   │       └── health/
│   │
│   ├── components/             # 通用组件
│   │   ├── ui/                 # Shadcn/ui 组件
│   │   │   ├── button.tsx
│   │   │   ├── input.tsx
│   │   │   ├── dialog.tsx
│   │   │   ├── dropdown-menu.tsx
│   │   │   ├── sheet.tsx
│   │   │   ├── tabs.tsx
│   │   │   ├── toast.tsx
│   │   │   └── ...
│   │   │
│   │   ├── layout/             # 布局组件
│   │   │   ├── sidebar.tsx
│   │   │   ├── header.tsx
│   │   │   ├── footer.tsx
│   │   │   └── breadcrumbs.tsx
│   │   │
│   │   ├── chat/               # Chat 专用组件
│   │   │   ├── chat-input.tsx
│   │   │   ├── message-bubble.tsx
│   │   │   ├── code-block.tsx
│   │   │   ├── model-selector.tsx
│   │   │   └── conversation-list.tsx
│   │   │
│   │   ├── agent/              # Agent 专用组件
│   │   │   ├── agent-card.tsx
│   │   │   ├── step-timeline.tsx
│   │   │   ├── tool-call-view.tsx
│   │   │   └── run-status.tsx
│   │   │
│   │   ├── editor/             # IDE 组件
│   │   │   ├── monaco-editor.tsx
│   │   │   ├── file-tree.tsx
│   │   │   ├── terminal.tsx
│   │   │   ├── git-panel.tsx
│   │   │   └── diff-viewer.tsx
│   │   │
│   │   ├── workflow/           # 工作流组件
│   │   │   ├── canvas.tsx
│   │   │   ├── node-palette.tsx
│   │   │   ├── workflow-node.tsx
│   │   │   └── edge-connector.tsx
│   │   │
│   │   └── shared/             # 通用业务组件
│   │       ├── data-table.tsx
│   │       ├── search-input.tsx
│   │       ├── file-upload.tsx
│   │       ├── markdown-renderer.tsx
│   │       ├── json-viewer.tsx
│   │       └── empty-state.tsx
│   │
│   ├── hooks/                  # 自定义 Hooks
│   │   ├── use-chat.ts
│   │   ├── use-agent.ts
│   │   ├── use-websocket.ts
│   │   ├── use-keyboard.ts
│   │   ├── use-theme.ts
│   │   └── use-debounce.ts
│   │
│   ├── lib/                    # 工具库
│   │   ├── api/                # API 客户端
│   │   │   ├── client.ts       # Axios/Fetch 封装
│   │   │   ├── auth.ts
│   │   │   ├── chat.ts
│   │   │   ├── agent.ts
│   │   │   ├── knowledge.ts
│   │   │   ├── project.ts
│   │   │   ├── workflow.ts
│   │   │   └── mcp.ts
│   │   │
│   │   ├── utils/
│   │   │   ├── cn.ts           # className 合并
│   │   │   ├── format.ts       # 格式化工具
│   │   │   ├── storage.ts      # localStorage 封装
│   │   │   └── validation.ts   # 表单验证
│   │   │
│   │   └── constants/
│   │       ├── models.ts
│   │       ├── routes.ts
│   │       └── config.ts
│   │
│   ├── stores/                 # 状态管理 (Zustand)
│   │   ├── auth-store.ts
│   │   ├── chat-store.ts
│   │   ├── editor-store.ts
│   │   ├── workflow-store.ts
│   │   └── theme-store.ts
│   │
│   ├── types/                  # TypeScript 类型
│   │   ├── api.ts
│   │   ├── chat.ts
│   │   ├── agent.ts
│   │   ├── knowledge.ts
│   │   ├── project.ts
│   │   └── workflow.ts
│   │
│   └── styles/                 # 全局样式
│       ├── globals.css
│       └── editor.css
│
├── .env.local
├── .env.example
├── next.config.ts
├── tailwind.config.ts
├── tsconfig.json
├── package.json
└── README.md
```

---

## 3. API Gateway `apps/gateway/`

```
apps/gateway/
├── cmd/
│   └── gateway/
│       └── main.go
│
├── internal/
│   ├── config/                 # 配置
│   │   └── config.go
│   │
│   ├── middleware/              # 中间件
│   │   ├── auth.go             # JWT 认证
│   │   ├── rate_limit.go       # 限流
│   │   ├── cors.go
│   │   ├── logger.go
│   │   ├── recovery.go
│   │   └── request_id.go
│   │
│   ├── router/                 # 路由
│   │   ├── router.go           # 路由注册
│   │   ├── auth.go
│   │   ├── chat.go
│   │   ├── agent.go
│   │   ├── knowledge.go
│   │   ├── project.go
│   │   ├── workflow.go
│   │   ├── mcp.go
│   │   ├── deploy.go
│   │   └── admin.go
│   │
│   ├── handler/                # HTTP 处理器
│   │   ├── auth.go
│   │   ├── chat.go             # SSE 流式处理
│   │   ├── agent.go
│   │   ├── knowledge.go
│   │   ├── project.go
│   │   ├── workflow.go
│   │   ├── mcp.go
│   │   ├── deploy.go
│   │   └── health.go
│   │
│   ├── proxy/                  # 代理层
│   │   ├── grpc_proxy.go       # gRPC 代理
│   │   ├── ws_proxy.go         # WebSocket 代理
│   │   └── sse_proxy.go        # SSE 代理
│   │
│   └── validator/              # 请求验证
│       └── validator.go
│
├── Dockerfile
├── go.mod
└── README.md
```

---

## 4. 后端微服务 `apps/services/`

```
apps/services/
├── user/                       # 用户服务
│   ├── cmd/
│   │   └── user/
│   │       └── main.go
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/             # 领域模型
│   │   │   ├── user.go
│   │   │   ├── organization.go
│   │   │   ├── api_key.go
│   │   │   └── role.go
│   │   ├── repository/         # 数据访问层
│   │   │   ├── user_repo.go
│   │   │   ├── org_repo.go
│   │   │   └── key_repo.go
│   │   ├── service/            # 业务逻辑层
│   │   │   ├── auth_service.go
│   │   │   ├── user_service.go
│   │   │   ├── org_service.go
│   │   │   └── rbac_service.go
│   │   ├── handler/            # gRPC 处理器
│   │   │   ├── auth_handler.go
│   │   │   ├── user_handler.go
│   │   │   └── org_handler.go
│   │   └── event/              # 事件处理
│   │       └── publisher.go
│   ├── migrations/             # 数据库迁移
│   │   ├── 001_create_users.up.sql
│   │   ├── 001_create_users.down.sql
│   │   └── ...
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
│
├── chat/                       # AI Chat 服务
│   ├── cmd/
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   │   ├── conversation.go
│   │   │   ├── message.go
│   │   │   └── prompt.go
│   │   ├── repository/
│   │   ├── service/
│   │   │   ├── chat_service.go
│   │   │   ├── conversation_service.go
│   │   │   ├── prompt_service.go
│   │   │   └── stream_service.go
│   │   ├── handler/
│   │   ├── adapter/            # AI 模型适配器
│   │   │   ├── adapter.go      # 接口定义
│   │   │   ├── openai.go
│   │   │   ├── anthropic.go
│   │   │   ├── gemini.go
│   │   │   ├── deepseek.go
│   │   │   ├── qwen.go
│   │   │   ├── ollama.go
│   │   │   └── router.go       # 模型路由器
│   │   └── event/
│   ├── migrations/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
│
├── agent/                      # Agent 服务
│   ├── cmd/
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   │   ├── agent.go
│   │   │   ├── run.go
│   │   │   ├── step.go
│   │   │   └── tool.go
│   │   ├── repository/
│   │   ├── service/
│   │   │   ├── agent_service.go
│   │   │   ├── planner.go      # 任务规划
│   │   │   ├── executor.go     # 步骤执行
│   │   │   └── tool_manager.go
│   │   ├── handler/
│   │   ├── tools/              # 内置工具
│   │   │   ├── tool.go         # 工具接口
│   │   │   ├── file_tool.go
│   │   │   ├── search_tool.go
│   │   │   ├── calculator.go
│   │   │   └── registry.go
│   │   └── event/
│   ├── migrations/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
│
├── rag/                        # RAG 服务
│   ├── cmd/
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   │   ├── knowledge_base.go
│   │   │   ├── document.go
│   │   │   └── chunk.go
│   │   ├── repository/
│   │   ├── service/
│   │   │   ├── kb_service.go
│   │   │   ├── doc_service.go
│   │   │   ├── parser/         # 文档解析器
│   │   │   │   ├── parser.go
│   │   │   │   ├── pdf.go
│   │   │   │   ├── docx.go
│   │   │   │   ├── pptx.go
│   │   │   │   ├── xlsx.go
│   │   │   │   └── markdown.go
│   │   │   ├── chunker/        # 分块器
│   │   │   │   ├── chunker.go
│   │   │   │   └── semantic.go
│   │   │   ├── embedder/       # 向量化
│   │   │   │   ├── embedder.go
│   │   │   │   └── openai.go
│   │   │   └── retriever/      # 检索器
│   │   │       ├── retriever.go
│   │   │       ├── vector.go
│   │   │       ├── bm25.go
│   │   │       ├── hybrid.go
│   │   │       └── reranker.go
│   │   ├── handler/
│   │   └── event/
│   ├── migrations/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
│
├── ide/                        # IDE 服务
│   ├── cmd/
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   ├── repository/
│   │   ├── service/
│   │   │   ├── file_service.go
│   │   │   ├── git_service.go
│   │   │   ├── terminal_service.go
│   │   │   └── workspace_service.go
│   │   ├── handler/
│   │   │   ├── file_handler.go
│   │   │   ├── git_handler.go
│   │   │   ├── ws_handler.go   # WebSocket 处理
│   │   │   └── terminal_handler.go
│   │   └── ws/                 # WebSocket 管理
│   │       ├── hub.go
│   │       ├── client.go
│   │       └── message.go
│   ├── migrations/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
│
├── workflow/                   # 工作流服务
│   ├── cmd/
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   │   ├── workflow.go
│   │   │   ├── node.go
│   │   │   └── run.go
│   │   ├── repository/
│   │   ├── service/
│   │   │   ├── workflow_service.go
│   │   │   ├── executor.go
│   │   │   └── parser.go
│   │   ├── handler/
│   │   ├── activities/         # Temporal Activities
│   │   │   ├── ai_activity.go
│   │   │   ├── http_activity.go
│   │   │   ├── sql_activity.go
│   │   │   ├── code_activity.go
│   │   │   └── condition_activity.go
│   │   └── workflows/          # Temporal Workflows
│   │       └── engine.go
│   ├── migrations/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
│
├── mcp/                        # MCP 服务
│   ├── cmd/
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   ├── repository/
│   │   ├── service/
│   │   │   ├── server_service.go
│   │   │   ├── tool_service.go
│   │   │   └── router.go
│   │   ├── handler/
│   │   ├── protocol/           # MCP 协议实现
│   │   │   ├── mcp.go
│   │   │   ├── sse_transport.go
│   │   │   └── stdio_transport.go
│   │   └── builtin/            # 内置 MCP Server
│   │       ├── filesystem.go
│   │       ├── github.go
│   │       ├── browser.go
│   │       ├── sql.go
│   │       └── docker.go
│   ├── migrations/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
│
├── deploy/                     # 部署服务
│   ├── cmd/
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   ├── repository/
│   │   ├── service/
│   │   │   ├── deploy_service.go
│   │   │   ├── builder.go      # 镜像构建
│   │   │   ├── docker.go       # Docker 部署
│   │   │   ├── kubernetes.go   # K8s 部署
│   │   │   └── domain.go       # 域名管理
│   │   └── handler/
│   ├── migrations/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
│
├── billing/                    # 计费服务
│   ├── cmd/
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   ├── repository/
│   │   ├── service/
│   │   │   ├── billing_service.go
│   │   │   ├── metering.go     # 用量计量
│   │   │   ├── invoice.go      # 账单生成
│   │   │   └── payment/        # 支付集成
│   │   │       ├── stripe.go
│   │   │       ├── wechat.go
│   │   │       └── alipay.go
│   │   ├── handler/
│   │   └── event/
│   ├── migrations/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
│
├── admin/                      # 管理服务
│   ├── cmd/
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   ├── repository/
│   │   ├── service/
│   │   │   ├── admin_service.go
│   │   │   ├── model_service.go
│   │   │   ├── audit_service.go
│   │   │   └── stats_service.go
│   │   └── handler/
│   ├── migrations/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
│
├── notification/               # 通知服务
│   ├── cmd/
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   ├── repository/
│   │   ├── service/
│   │   │   ├── notification_service.go
│   │   │   └── channels/       # 通知渠道
│   │   │       ├── email.go
│   │   │       ├── slack.go
│   │   │       ├── webhook.go
│   │   │       └── in_app.go
│   │   ├── handler/
│   │   └── event/
│   ├── migrations/
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
│
└── monitor/                    # 监控服务
    ├── cmd/
    ├── internal/
    │   ├── config/
    │   ├── domain/
    │   ├── service/
    │   │   ├── metrics_service.go
    │   │   ├── log_service.go
    │   │   ├── alert_service.go
    │   │   └── dashboard_service.go
    │   └── handler/
    ├── migrations/
    ├── Dockerfile
    ├── go.mod
    └── README.md
```

---

## 5. 后台工作者 `apps/workers/`

```
apps/workers/
├── doc-processor/              # 文档处理 Worker
│   ├── main.go
│   ├── processor.go
│   ├── Dockerfile
│   └── go.mod
│
├── embedding-worker/           # Embedding Worker
│   ├── main.go
│   ├── worker.go
│   ├── Dockerfile
│   └── go.mod
│
├── billing-worker/             # 计费聚合 Worker
│   ├── main.go
│   ├── aggregator.go
│   ├── Dockerfile
│   └── go.mod
│
└── notification-worker/        # 通知发送 Worker
    ├── main.go
    ├── sender.go
    ├── Dockerfile
    └── go.mod
```

---

## 6. 共享包 `packages/`

```
packages/
├── proto/                      # Protobuf 定义
│   ├── buf.yaml
│   ├── buf.gen.yaml
│   ├── user/
│   │   └── v1/
│   │       ├── user.proto
│   │       ├── auth.proto
│   │       └── org.proto
│   ├── chat/
│   │   └── v1/
│   │       ├── chat.proto
│   │       ├── conversation.proto
│   │       └── prompt.proto
│   ├── agent/
│   │   └── v1/
│   │       ├── agent.proto
│   │       └── run.proto
│   ├── rag/
│   │   └── v1/
│   │       ├── knowledge.proto
│   │       ├── document.proto
│   │       └── search.proto
│   ├── workflow/
│   │   └── v1/
│   │       └── workflow.proto
│   ├── mcp/
│   │   └── v1/
│   │       └── mcp.proto
│   ├── deploy/
│   │   └── v1/
│   │       └── deploy.proto
│   ├── billing/
│   │   └── v1/
│   │       └── billing.proto
│   └── common/
│       └── v1/
│           ├── pagination.proto
│           ├── error.proto
│           └── types.proto
│
├── go-common/                  # Go 公共库
│   ├── go.mod
│   ├── logger/                 # 日志封装
│   │   ├── logger.go
│   │   └── zap.go
│   ├── config/                 # 配置加载
│   │   ├── config.go
│   │   └── viper.go
│   ├── database/               # 数据库连接
│   │   ├── postgres.go
│   │   └── migrate.go
│   ├── cache/                  # Redis 封装
│   │   └── redis.go
│   ├── auth/                   # JWT 工具
│   │   ├── jwt.go
│   │   └── password.go
│   ├── errors/                 # 统一错误
│   │   └── errors.go
│   ├── middleware/              # 通用中间件
│   │   ├── recovery.go
│   │   └── logging.go
│   ├── event/                  # 事件封装
│   │   ├── kafka.go
│   │   └── producer.go
│   ├── storage/                # MinIO 封装
│   │   └── minio.go
│   ├── telemetry/              # OpenTelemetry
│   │   ├── otel.go
│   │   └── tracer.go
│   └── validator/              # 验证器
│       └── validator.go
│
├── ts-common/                  # TypeScript 公共库
│   ├── package.json
│   ├── tsconfig.json
│   └── src/
│       ├── api/                # API 客户端生成
│       ├── types/              # 共享类型
│       ├── utils/              # 工具函数
│       └── constants/          # 常量
│
├── ui/                         # UI 组件库 (Shadcn)
│   ├── package.json
│   ├── tsconfig.json
│   ├── tailwind.config.ts
│   └── src/
│       ├── components/
│       │   ├── ui/             # Shadcn 基础组件
│       │   ├── forms/          # 表单组件
│       │   ├── data/           # 数据展示组件
│       │   └── layout/         # 布局组件
│       ├── hooks/
│       ├── utils/
│       └── styles/
│
└── config/                     # 共享配置
    ├── eslint/
    │   └── base.js
    ├── prettier/
    │   └── base.js
    └── tsconfig/
        ├── base.json
        ├── nextjs.json
        └── node.json
```

---

## 7. 部署配置 `deploy/`

```
deploy/
├── docker/
│   ├── docker-compose.yml      # 生产级 Docker Compose
│   ├── docker-compose.dev.yml  # 开发环境
│   ├── docker-compose.infra.yml # 基础设施
│   └── nginx/
│       └── nginx.conf
│
├── helm/
│   └── omnidev/
│       ├── Chart.yaml
│       ├── values.yaml
│       ├── values-staging.yaml
│       ├── values-production.yaml
│       └── templates/
│           ├── _helpers.tpl
│           ├── gateway/
│           ├── services/
│           ├── workers/
│           ├── ingress/
│           ├── configmap/
│           ├── secret/
│           └── hpa/
│
├── terraform/
│   ├── modules/
│   │   ├── vpc/
│   │   ├── eks/
│   │   ├── rds/
│   │   ├── elasticache/
│   │   ├── s3/
│   │   └── opensearch/
│   ├── environments/
│   │   ├── staging/
│   │   │   ├── main.tf
│   │   │   ├── variables.tf
│   │   │   └── outputs.tf
│   │   └── production/
│   │       ├── main.tf
│   │       ├── variables.tf
│   │       └── outputs.tf
│   └── backend.tf
│
└── k8s/
    ├── base/
    │   ├── kustomization.yaml
    │   ├── namespace.yaml
    │   └── network-policy.yaml
    ├── overlays/
    │   ├── staging/
    │   └── production/
    └── sandbox/
        ├── runtime-python.yaml
        ├── runtime-go.yaml
        ├── runtime-node.yaml
        └── network-policy.yaml
```

---

## 8. 工具脚本 `scripts/`

```
scripts/
├── setup.sh                    # 环境初始化
├── migrate.sh                  # 数据库迁移
├── seed.sh                     # 种子数据
├── gen-proto.sh                # Protobuf 代码生成
├── gen-swagger.sh              # Swagger 文档生成
├── lint.sh                     # 代码检查
├── test.sh                     # 运行测试
├── build.sh                    # 构建所有服务
└── deploy.sh                   # 部署脚本
```

---

## 9. 文档 `docs/`

```
docs/
├── architecture/               # 架构文档（本目录）
│   ├── 00-EXECUTIVE-SUMMARY.md
│   ├── 01-REQUIREMENTS-ANALYSIS.md
│   ├── 02-FEATURE-BOUNDARY.md
│   ├── 03-NON-FUNCTIONAL-REQUIREMENTS.md
│   ├── 04-TECHNOLOGY-SELECTION.md
│   ├── 05-SYSTEM-ARCHITECTURE.md
│   ├── 06-DATABASE-DESIGN.md
│   ├── 07-DIRECTORY-STRUCTURE.md
│   ├── 08-DEVELOPMENT-STANDARDS.md
│   └── 09-MILESTONE-PLAN.md
│
├── api/                        # API 文档
│   ├── openapi.yaml            # OpenAPI 规范
│   └── grpc/                   # gRPC 文档
│
├── guides/                     # 使用指南
│   ├── getting-started.md
│   ├── deployment.md
│   └── contributing.md
│
└── adr/                        # 架构决策记录
    ├── 001-monorepo.md
    ├── 002-go-backend.md
    └── ...
```

---

## 10. 开发工具 `tools/`

```
tools/
├── cli/                        # CLI 工具
│   ├── cmd/
│   │   ├── main.go
│   │   ├── init.go
│   │   ├── dev.go
│   │   ├── deploy.go
│   │   └── config.go
│   ├── go.mod
│   └── README.md
│
└── codegen/                    # 代码生成器
    ├── proto-gen/              # Protobuf 生成
    ├── repo-gen/               # Repository 代码生成
    └── api-gen/                # API 客户端生成
```
