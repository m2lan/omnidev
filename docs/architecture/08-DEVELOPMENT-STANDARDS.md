# OmniDev — 开发规范（当前仓库）

## 1. 仓库与变更边界

```text
omnidev/
├── apps/gateway/          # 唯一 Go 后端：auth、chat、knowledge、agent
├── apps/web/              # Next.js 前端
├── packages/go-common/    # Go 共享库
├── deploy/docker/         # Docker Compose 与 Nginx 配置
├── docs/                  # 文档
└── Makefile               # 统一开发命令
../knowledge-engine/       # 同级、独立可复用的 Go module
```

- 后端业务优先放入 `apps/gateway/internal` 对应模块，不新建 `apps/services/*`。
- 跨后端模块复用的基础能力放入 `packages/go-common`；知识领域通用能力放入 sibling `knowledge-engine`。
- 不为单个功能引入新的网络服务、Kafka 事件、protobuf/wire 生成或 Kubernetes manifests。
- 修改功能时同步更新真实路由、迁移和文档；不要把规划能力写成已交付能力。

## 2. Git 与提交

- 推荐从短期功能/修复分支提交 PR，避免直接在主分支开发。
- Commit 使用 `<type>(<scope>): <description>`；常用 type 为 `feat/fix/docs/refactor/perf/test/build/ci/chore`。
- scope 使用真实模块，如 `gateway`、`web`、`common`、`knowledge`、`agent`、`chat`、`deploy`、`docs`。
- 一个提交聚焦一个逻辑变更，不混入无关格式化、生成物、密钥或本地 `.env`。
- PR 说明变更原因、影响范围和验证命令；前端交互变化附必要截图，数据库变化说明迁移与回滚。

## 3. Go 规范

### 3.1 分层与依赖

- 沿用 `handler → service → repository` 和独立 `domain`；Agent 代码位于 `internal/agent` 的对应子包。
- HTTP 参数解析/响应在 handler，业务规则与所有权校验在 service，SQL 在 repository。
- 在 `cmd/gateway/main.go` 显式组装依赖；通过构造函数传入依赖，不在业务包中读取全局环境配置。
- 跨模块调用使用进程内接口或具体 service，不添加内部 REST/gRPC 跳转。
- `context.Context` 作为调用链首个参数； goroutine 需明确 context 生命周期、取消和错误记录。

### 3.2 代码与错误

- 提交前运行 `gofmt`；包名短小，导出标识符有必要注释，避免无意义包装和重复抽象。
- 错误使用 `%w` 或项目 `go-common/errors` 包装，保留底层原因；不向用户返回 SQL、密钥或堆栈。
- 日志使用 zap 结构化字段，至少带 request ID、run ID 或资源 ID 等上下文；禁止记录密码、JWT、provider Key 和附件内容。
- 配置通过 `go-common/config` 增加字段、默认值和环境变量映射；复杂结构（如 MCP servers）明确解析规则。

### 3.3 数据库

- 使用 pgx/v5 和参数化 SQL，禁止字符串拼接用户输入；动态标识符必须严格白名单校验。
- 查询列表必须分页并限制页大小；按 repository 现有约定处理软删除、排序和 `user_id` 所有权。
- 多表写入若要求原子性，使用事务；外部 AI、MinIO、Docker 调用不要无条件包在长事务中。
- JSONB 用于确有动态结构的配置/元数据，固定查询字段仍使用明确列和索引。
- 消息表是分区表，涉及主键、唯一约束或分区维护时先阅读 006 迁移。

## 4. TypeScript / React 规范

- 组件使用 PascalCase，hooks 使用 `useXxx`，工具函数使用 camelCase；类型和 API 输入输出避免 `any`。
- 优先复用 `apps/web/src` 现有 API client、组件和类型，不在页面中重复拼装 fetch、认证头和错误处理。
- 服务端数据优先使用 TanStack Query；仅把真正的本地 UI 状态放入 Zustand 或组件 state。
- 表单沿用 React Hook Form + Zod；在边界校验外部数据，模型流式增量与完整消息要区分处理。
- 使用 Tailwind 和现有 UI 组件保持样式一致；交互需覆盖 loading、empty、error 和取消状态。
- 不重新引入 Monaco IDE 或 workflow 画布；Agent 页面以配置、run 轮询、步骤和取消为边界。

## 5. API 约定

- 业务 API 位于 `/api/v1`，使用名词资源和合适 HTTP 方法；路由统一在 `internal/router/router.go` 注册。
- 除认证/健康入口外默认使用 JWT；service 必须校验资源所有权，不能只依赖隐藏前端按钮。
- 单资源响应沿用 `{"data": ...}`；分页响应使用 `meta.total_count/page/page_size`，查询参数为 `page/page_size`。
- 错误响应沿用 `error.code/message/detail` 及可用的 `request_id`；未定义路由返回结构化 404。
- 聊天流式接口使用 SSE：`POST /conversations/:id/messages/stream`。不要为 Agent 进度新增 WebSocket，当前客户端通过 run API 轮询。
- 破坏性 API 修改需同步 Web client 和文档；不能仅修改 handler 而遗漏路由或类型。

## 6. 常用命令

在仓库根目录执行：

```bash
# 后端
make build
make test
make test-race
make vet
make lint
make fmt

go build ./apps/gateway/... ./packages/go-common/...
go test ./apps/gateway/... ./packages/go-common/... -count=1
SANDBOX_INTEGRATION_TEST=1 go test ./apps/gateway/internal/agent/executor -run TestSandboxExecuteDocker -v

# 前端
pnpm --filter web dev
pnpm --filter web build
pnpm --filter web lint
pnpm --filter web typecheck
pnpm --filter web test

# 本地环境
make setup
make dev-infra
make run-gateway
make web-dev
make sandbox-images
```

修改 sibling knowledge-engine 时，还应在其目录运行 `go test ./...`，并同时验证 gateway 构建，因为 gateway 通过 `replace` 使用本地源码。

## 7. 数据库迁移工作流

1. 在根目录运行 `make db-migrate-create NAME=add_xxx`，工具会在 `apps/gateway/migrations` 生成顺序编号的 up/down SQL。
2. 检查编号顺序，填写可重复执行且约束明确的 up，以及真正可逆的 down；不要修改已部署迁移。
3. 本地执行 `make db-migrate`；需要时以 `make db-migrate-down` 回滚最近一个迁移，并验证再次 up。
4. 测试重启后的初始化、空库和已有数据场景；新增查询同时评估索引、软删除与所有权条件。
5. knowledge-engine 表结构变更放在 `../knowledge-engine/migrations`，不要混入 gateway 编号；嵌入启动会自动执行其迁移。

## 8. 测试要求

- Go 单元测试使用 `*_test.go`，优先覆盖 service 规则、所有权、工具循环、路径逃逸、沙箱参数和 repository 关键 SQL。
- 外部 AI/MCP/MinIO 测试使用可控 HTTP server 或接口替身；不得在默认测试中依赖真实付费 API。
- 需要 Docker/PostgreSQL/Redis 的集成测试应清楚标记环境前置条件；沙箱测试验证无网络、超时和输出截断。
- 前端至少运行 typecheck 和相关 Vitest；流式 UI 测试覆盖追加、完成和错误路径。
- CI 只构建和测试受影响的 gateway、go-common、web 与必要的 knowledge-engine，不维护多服务 protobuf/Kafka 矩阵。

## 9. 安全与审查清单

- 密钥只来自环境或受控密钥管理，不提交 `.env`；生产必须设置强 JWT secret 和有效 `ENCRYPTION_KEY`。
- 新文件能力必须限制类型/大小和存储归属；新工作区工具必须阻止绝对路径、`..` 和符号链接逃逸。
- 修改沙箱时保留无网络、只读 root、cap-drop、no-new-privileges、资源限制、超时和输出上限，除非有明确安全评审。
- 检查列表分页、错误处理、context 取消、资源关闭和日志脱敏。
- 检查文档没有 resurrect 已移除的 IDE、workflow、billing、监控平台、微服务或 Kubernetes 设计。
