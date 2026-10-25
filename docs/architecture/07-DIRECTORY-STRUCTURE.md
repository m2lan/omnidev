# OmniDev — 项目目录结构

## 1. 仓库关系

OmniDev 当前由一个主仓库和一个并列的知识引擎仓库组成：

```text
ai-project/
├── omnidev/                 # Web、模块化单体后端、部署与文档
└── knowledge-engine/        # 可嵌入的 RAG 引擎，也可独立运行 HTTP / MCP server
```

`apps/gateway/go.mod` 通过本地 `replace` 指向 `../knowledge-engine`，因此开发和 CI 都要让两个仓库保持并列目录。知识引擎自身的目录与独立部署说明由该仓库维护。

## 2. 主仓库实际布局

```text
omnidev/
├── .github/workflows/
│   ├── ci.yml                    # Go / Web 检查、迁移与镜像构建
│   └── cd-staging.yml            # 构建并推送 gateway 镜像
├── apps/
│   ├── gateway/                  # 唯一后端进程
│   └── web/                      # Next.js 前端
├── packages/
│   └── go-common/                # 共享 Go 基础包
├── deploy/docker/
│   ├── docker-compose.infra.yml  # PostgreSQL、Redis、MinIO、Tika、迁移
│   ├── docker-compose.dev.yml    # gateway、nginx 与沙箱挂载
│   ├── init-scripts/
│   └── nginx/
├── docs/
│   ├── architecture/             # 当前架构、边界、数据库与计划
│   └── features/                 # 已实现功能的专项说明
├── scripts/setup.sh              # 本地开发环境初始化
├── .env.example                  # 当前有效环境变量
├── go.work                       # apps/gateway + packages/go-common
├── Makefile                      # 单体后端与 Web 的开发命令
├── package.json                  # 根前端任务
├── pnpm-workspace.yaml           # pnpm workspace，当前只包含 apps/web
├── pnpm-lock.yaml
├── turbo.json
├── README.md
└── README_zh.md
```

已删除旧规划中的 `apps/services`、`apps/workers`、`packages/proto`、`deploy/helm` 等目录；新增功能不应重新按微服务复制骨架。

## 3. Gateway：`apps/gateway/`

```text
apps/gateway/
├── cmd/gateway/main.go           # 配置加载、依赖装配、路由与进程启动
├── internal/
│   ├── adapter/                  # OpenAI、DeepSeek、Qwen、Anthropic 等模型适配器
│   ├── agent/
│   │   ├── domain/               # Agent、Run、Step 与工具配置模型
│   │   ├── planner/              # 任务规划与后续步骤决策
│   │   ├── executor/             # 执行循环、工作区与 Docker 沙箱
│   │   ├── tools/                # file、search、calculator、code_exec
│   │   ├── repository/           # Agent、运行和步骤持久化
│   │   ├── service/              # 异步运行、查询与取消
│   │   └── handler/              # Agent HTTP API
│   ├── domain/                   # 用户、会话、消息等领域模型
│   ├── handler/                  # 认证、用户、Chat、健康检查等 HTTP Handler
│   ├── knowledge/                # knowledge-engine 的 gateway 接入层
│   ├── mcp/                      # Streamable HTTP MCP client 与工具管理器
│   ├── repository/               # 用户、会话、消息等数据访问
│   ├── router/                   # Gin 路由与中间件组合
│   └── service/                  # 认证、用户、Chat 与模型配置业务逻辑
├── migrations/                   # 单库顺序迁移，001–009，up/down 成对
├── Dockerfile                    # gateway 镜像，包含 Docker CLI
├── go.mod                        # 引用 go-common 与 knowledge-engine
└── go.sum
```

### 约定

- 新代码优先加入现有模块，不在 `cmd` 下增加第二个长期运行服务。
- HTTP 层负责协议与鉴权，业务层负责流程，repository 负责持久化。
- Agent 专属类型和逻辑保留在 `internal/agent`，避免与 Chat 的通用 service 混杂。
- 新表只在这里增加连续编号的 up/down 迁移，不恢复各服务独立 migrations。
- `knowledge-engine` 的 RAG 实现保留在兄弟仓库；这里只放装配和平台集成代码。

## 4. Web：`apps/web/`

```text
apps/web/
├── public/
├── src/
│   ├── app/
│   │   ├── page.tsx              # 产品入口页
│   │   ├── layout.tsx
│   │   ├── (auth)/               # 登录与注册
│   │   └── (dashboard)/
│   │       ├── layout.tsx        # 当前导航，只保留实际页面
│   │       ├── chat/             # 多模型对话
│   │       ├── knowledge/        # 知识库与文档
│   │       ├── agent/            # Agent 配置、异步运行、步骤与取消
│   │       └── settings/         # 用户、模型密钥等设置
│   ├── components/
│   │   ├── ui/                   # 基础 UI 组件
│   │   ├── chat/
│   │   ├── knowledge/
│   │   ├── settings/
│   │   └── providers.tsx
│   ├── lib/
│   │   ├── api/client.ts         # 与 gateway 对应的 API 和类型
│   │   └── utils.ts
│   └── stores/
│       ├── auth-store.ts
│       └── chat-store.ts
├── next.config.ts
├── package.json
├── tsconfig.json
└── tailwind.config.ts
```

前端不保留没有后端能力的演示页。IDE、Workflow、Billing、Deploy 和 Monitoring 页面已移除；Agent 页面读取真实 API，不生成模拟步骤。

## 5. 共享包：`packages/go-common/`

```text
packages/go-common/
├── auth/                         # JWT 与认证辅助
├── cache/                        # Redis
├── config/                       # 配置结构、env 绑定与加载
├── crypto/                       # 密钥加解密
├── database/                     # PostgreSQL 连接
├── domain/                       # 跨模块共享的基础类型
├── errors/                       # 统一错误
├── logger/                       # 日志
├── middleware/                   # 通用 Gin 中间件
├── parser/                       # Tika 文档解析
├── storage/                      # MinIO 对象存储
├── telemetry/                    # 基础遥测辅助，不代表完整监控平台
├── validator/                    # 输入验证
├── go.mod
└── go.sum
```

只有 gateway 与其他 Go 模块确实共用的基础能力才放入该包；业务逻辑仍归各自模块。Kafka event 包已经删除。

## 6. 部署与脚本

```text
deploy/docker/
├── docker-compose.infra.yml      # 必需基础设施与一次性 migrate 服务
├── docker-compose.dev.yml        # gateway + nginx，支持 Docker-outside-of-Docker
├── init-scripts/                 # 数据库初始化脚本
└── nginx/                        # 反向代理配置

scripts/
└── setup.sh                      # 检查依赖、准备环境并执行迁移
```

Compose 是当前的部署事实来源。不要把已经不存在的 Helm Chart、Terraform 模块或 Kubernetes manifests 写回启动说明。

## 7. 文档结构

```text
docs/
├── architecture/
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
└── features/
    └── file-upload-and-multimodal.md
```

架构文档描述当前代码；规划项必须明确标注“规划中”，已放弃项标注“已取消/已移除”。公开文档站位于独立的 `omnidev-docs` 仓库，修改平台能力或配置时需要同步更新中英文页面。
