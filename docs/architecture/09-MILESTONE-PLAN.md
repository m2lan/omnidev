# OmniDev — 里程碑与真实状态

> 更新基准：2026-10-25。本文记录代码仓库中的实际交付状态，不再把旧产品蓝图标为完成。

## 1. 状态定义

- `[x]`：代码已经实现；只表示对应代码和自动检查存在，不等于通过生产验收或提供 SLA。
- `[ ]`：尚未实现或尚未完成验证，不能写入对外“已支持”清单。
- `~~删除线~~`：明确取消，不再作为后续默认路线。

当前最重要的范围决策是：集中做 **Chat + 知识库 + 后台 Agent**，不再同时建设 IDE、工作流、支付、部署托管和监控平台。

## 2. M0：可用基础与 Chat

**目标：** 用户能注册登录、配置模型并完成真实对话，开发环境可以重复启动。

- [x] Next.js Web 与 Go/Gin gateway 项目骨架。
- [x] PostgreSQL、Redis、MinIO、Tika 的 Docker Compose 开发环境。
- [x] 单库数据库迁移和本地初始化脚本。
- [x] JWT 登录、刷新令牌、API Key 与 OAuth 接入点。
- [x] 用户资源隔离与用户模型配置（BYOK）。
- [x] OpenAI、DeepSeek、Qwen、Anthropic 普通对话适配器。
- [x] 流式/非流式 Chat、会话与消息持久化。
- [x] MinIO 附件、Tika 解析、图片输入与生成链路。
- [x] Go build/vet/test 与 Web typecheck 的 CI 基础检查。
- [ ] 完整的注册 → 配模型 → 附件 → 流式对话浏览器 E2E 自动测试。
- [ ] 生产级备份恢复、高可用、发布回滚与 SLA 验证。

## 3. M1：知识引擎真正可用并接入平台

**目标：** 文档能入库、检索并进入 Chat，同时可以作为独立能力被其他工具调用。

- [x] 修复不同 embedding 模型的向量维度冲突，支持按模型/知识库确定维度。
- [x] knowledge-engine 接入 JWT 或 API Key 认证与用户权限校验。
- [x] 文档解析统一接入 Tika。
- [x] Redis 检索/嵌入缓存接线。
- [x] 知识库 chunk 配置在入库流程中生效。
- [x] knowledge-engine 入库与检索集成测试。
- [x] knowledge-engine 独立 HTTP API 与 MCP server。
- [x] gateway 直接 import knowledge-engine，删除重复 RAG 服务代码。
- [x] Chat 使用知识检索结果并保留引用上下文。
- [x] Web 知识库与文档管理页面。
- [ ] 对更多真实中文文档格式、扫描件和大文件进行系统性质量评估。
- [ ] OCR、代码仓库索引、自动 rerank 等增强能力；未实现前不得列为现状。

## 4. M2：工具调用与后台 Agent

**目标：** Chat 能调用外部工具，Agent 能异步执行编码相关任务，并把不可信代码关进真实 Docker 沙箱。

### Chat 工具调用

- [x] Streamable HTTP MCP client：initialize、tools/list、tools/call。
- [x] MCP server 配置、启动连接、工具名冲突处理与失败降级。
- [x] OpenAI 兼容协议的非流式与流式 tool call 解析。
- [x] 最多 5 轮工具循环、8000 字符工具输出限制与 `tools_used` 元数据。
- [x] DeepSeek、Qwen 适配器传递 tools 并返回 tool calls。
- [ ] Anthropic `tool_use` / `tool_result` 映射；当前只支持普通对话，不支持 MCP 工具循环。
- [ ] 面向多个真实 MCP server 的端到端兼容性与超时/重连测试矩阵。

### 后台 Agent 与沙箱

- [x] Agent 配置、运行和步骤的 PostgreSQL 模型与迁移。
- [x] 自然语言任务、planner/executor 循环和最大步数控制。
- [x] file、search、calculator、code_exec 工具。
- [x] 每次运行独立工作区及文件路径约束。
- [x] 异步启动、状态轮询、步骤落库、结果查询与取消 API。
- [x] Web Agent 配置、任务提交、运行历史、步骤展示和取消。
- [x] Docker 沙箱：独立 `/workspace`、默认无网络、只读 rootfs、tmpfs `/tmp`、drop capabilities、`no-new-privileges`、资源与输出限制、超时清理。
- [x] gateway 容器内调用宿主机 Docker 时的 host/container 工作区路径映射。
- [ ] 在目标部署环境完成真实沙箱运行的自动化 smoke test，并覆盖取消与资源超限。
- [ ] 将进程内异步执行升级为可恢复、可水平扩展的持久任务机制；当前不承诺进程重启后的 run 恢复。
- [ ] Git 仓库授权、分支/提交、Diff 和凭据隔离。
- [ ] Agent 自动创建或更新 PR，以及检查状态回传。**这是规划项，当前没有实现。**
- [ ] Agent 端到端验收：从真实任务到可审查的代码交付，而不是只返回文字结果。

## 5. M3：聚焦重构与文档纠偏

**目标：** 删除伪需求和重复服务，让代码、部署、Web 与公开描述一致。

- [x] 将旧多服务骨架收缩为 `apps/gateway` 模块化单体。
- [x] 迁移集中到 `apps/gateway/migrations`。
- [x] 删除未使用的 Kafka、Elasticsearch、Temporal、Protobuf 与 Kafka event 包。
- [x] Compose、Makefile、CI/CD 和 `.env.example` 改为当前运行拓扑。
- [x] 删除在线 IDE、拖拽工作流和设置页中的伪计费/通知界面。
- [x] 首页和 README 只描述真实能力，明确 PR 与 Anthropic 工具调用的缺口。
- [x] 架构文档按“已实现 / 规划中 / 已取消”重写。
- [ ] 对所有公开文档与示例执行一次完整链接、命令和 API 路径核验。
- [ ] 在干净环境按 README 完成一次全新安装验收。

## 6. 下一阶段：主线验收，而非重新扩 scope

下一阶段没有预设日历或虚假发布日期。按以下顺序推进，每一项都要有可重复验证：

1. **Agent 真实交付闭环**：真实仓库任务、受控 Git 凭据、Diff 审查、PR 交付与取消/失败恢复。
2. **运行可靠性**：持久任务状态、重启恢复、并发限制、工作区清理和沙箱安全测试。
3. **知识质量**：中文文档评测集、检索质量指标、引用正确性与大数据量测试。
4. **工具生态**：Anthropic 工具协议映射、更多真实 MCP server 的兼容性测试。
5. **发布质量**：浏览器 E2E、干净环境安装、生产备份恢复和最小必要运维手册。

只有实际验收通过的项目才能改为 `[x]`。不得为了勾选里程碑降低“真实可用”的标准。

## 7. 已取消范围

以下旧路线不再排期；如果未来确有用户需求，应作为新决策重新评审，而不是默认恢复：

- [ ] ~~在线 IDE、Monaco 编辑器、浏览器假终端与内置 Git/Code Review 平台~~（已取消）。
- [ ] ~~React Flow / Temporal 可视化拖拽工作流画布~~（已取消）。
- [ ] ~~自建 Billing、订阅、Stripe/微信/支付宝支付~~（已取消）。
- [ ] ~~一键应用部署、多云托管与域名管理~~（已取消）。
- [ ] ~~Prometheus/Grafana/Loki/Jaeger 全家桶和 AI 异常检测产品~~（已取消；不排斥集成外部运维工具）。
- [ ] ~~通知微服务与内置消息中心~~（已取消）。
- [ ] ~~16 微服务、gRPC 服务间调用和 Kafka 事件总线~~（已由模块化单体替代）。
- [ ] ~~Elasticsearch 混合检索与 Temporal 工作流运行时~~（当前无对应依赖）。
- [ ] ~~Kubernetes/Helm/Terraform 作为默认部署目标~~（当前仅维护 Docker Compose）。
- [ ] ~~Agent 可视化编排与模板市场~~（已取消，优先完成单 Agent 的真实交付闭环）。

## 8. 当前发布门禁

在声称“生产就绪”之前，至少还需完成：

- [ ] Chat、知识库、Agent 的核心浏览器 E2E 全部通过。
- [ ] 数据库从空库迁移、重复启动和回滚流程经过验证。
- [ ] Docker 沙箱在支持的目标平台通过真实运行、取消、超时、网络隔离和路径逃逸测试。
- [ ] 对进程重启、模型/MCP/存储不可用等故障行为有明确说明和测试。
- [ ] 安全配置、密钥管理、Docker socket 权限和备份恢复形成可执行手册。
- [ ] 公开文档中的每条命令、API 和状态表与实际版本一致。

当前版本应描述为**核心代码可用的聚焦重构版**，而不是已经完成安全审计、性能验收和 SLA 验证的 GA 产品。
