# OmniDev — 执行摘要

## 1. 项目定位

OmniDev 是一个**面向团队的 AI 知识与编码工作台**，聚焦三条已经可用的主线：

1. **Chat**：多模型对话，支持流式输出、附件、多模态、用户自带 Key（BYOK）与外部 MCP 工具。
2. **知识库**：文档上传 → Tika 解析 → 分块 → 向量化 → pgvector 检索，并可注入对话上下文。
3. **后台 Agent**：异步执行任务，代码在 Docker 沙箱中运行，步骤落库、可轮询、可取消。

**一句话定位：** 用一套可私有部署的系统，把团队知识、模型对话和自动化编码任务串起来。

## 2. 为什么收缩

此前仓库按"All-in-One 平台"规划了 16 个微服务、在线 IDE、工作流编排、支付与监控体系，但真实可运行的只有一个 Gateway。把规划写成现状会误导使用者，因此当前版本明确：

- **保留并做深**：Chat、知识引擎、后台 Agent。
- **直接移除**：在线 IDE（Monaco + 假终端）、可视化工作流画布、自建支付、监控告警全家桶、一键部署、通知服务。
- **架构降级为现实形态**：模块化单体，删除未使用的 Kafka、Elasticsearch、Temporal、Helm/Kubernetes 清单。

## 3. 目标用户与场景

| 用户 | 场景 |
|------|------|
| 小团队（2–20 人） | 私有部署，接入自己的模型 Key，沉淀团队文档并在对话中检索 |
| 独立开发者 | 用自然语言驱动后台 Agent 跑脚本、整理文件、验证代码 |
| AI 应用开发者 | 复用 knowledge-engine 作为 Go 模块或 MCP server，接入自己的工具链 |

## 4. 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 部署形态 | 单 Go 二进制 + Next.js | 运维成本低，符合真实代码结构 |
| 知识能力 | knowledge-engine 作为可嵌入模块 | 同一份代码同时服务自家平台与 MCP 生态 |
| 向量存储 | PostgreSQL + pgvector | 与业务数据同库，按知识库匹配向量维度 |
| Agent 执行 | Docker CLI 启动一次性容器 | 无网络、只读根fs、资源限额，避免在宿主机执行生成代码 |
| 外部工具 | MCP（streamable HTTP）客户端 | 复用现有 MCP 生态，不自建插件协议 |
| 文档解析 | Apache Tika | 复用成熟解析能力，覆盖 PDF/Office 等格式 |
| 商业模式 | 暂不做支付与订阅 | 先私有部署与自用，支付可由第三方集成 |

## 5. 当前规模（真实）

| 维度 | 现状 |
|------|------|
| 后端进程 | 1 个（`apps/gateway`），另有独立的 knowledge-engine 仓库 |
| 前端页面 | Chat、知识库、Agent、设置、登录/注册 |
| 基础设施 | PostgreSQL、Redis、MinIO、Tika、Docker（沙箱） |
| 数据库迁移 | `apps/gateway/migrations` 001–009，外加 knowledge-engine 自维护的表 |

## 6. 文档索引

| 文档 | 内容 |
|------|------|
| [01-REQUIREMENTS-ANALYSIS.md](01-REQUIREMENTS-ANALYSIS.md) | 功能范围与非目标 |
| [02-FEATURE-BOUNDARY.md](02-FEATURE-BOUNDARY.md) | 模块边界与依赖 |
| [03-NON-FUNCTIONAL-REQUIREMENTS.md](03-NON-FUNCTIONAL-REQUIREMENTS.md) | 安全、隔离、限额 |
| [04-TECHNOLOGY-SELECTION.md](04-TECHNOLOGY-SELECTION.md) | 技术选型与移除项 |
| [05-SYSTEM-ARCHITECTURE.md](05-SYSTEM-ARCHITECTURE.md) | 运行拓扑与请求链路 |
| [06-DATABASE-DESIGN.md](06-DATABASE-DESIGN.md) | 真实表结构 |
| [07-DIRECTORY-STRUCTURE.md](07-DIRECTORY-STRUCTURE.md) | 目录结构 |
| [08-DEVELOPMENT-STANDARDS.md](08-DEVELOPMENT-STANDARDS.md) | 编码规范与流程 |
| [09-MILESTONE-PLAN.md](09-MILESTONE-PLAN.md) | 已完成 / 规划中 / 已取消 |
