# OmniDev AI Platform — 执行摘要

## 1. 项目定位

OmniDev 是一个**All-in-One AI 开发平台**，将 AI 编程助手、在线 IDE、Agent 编排、RAG 知识库、CI/CD 部署、监控运维融合为统一产品。

**一句话定位：** 打开浏览器，就能用 AI 写代码、管项目、部署上线。

## 2. 竞品对标矩阵

| 能力域 | 竞品 | OmniDev 差异化 |
|--------|------|----------------|
| AI 编程助手 | Cursor, GitHub Copilot | 多模型切换 + 本地模型 + RAG 增强 |
| 在线 IDE | GitHub Codespaces, Gitpod | 内嵌 AI Agent + 一键部署 |
| Agent 平台 | OpenHands, AutoGPT | 可视化编排 + 沙箱执行 + 插件市场 |
| RAG / 知识库 | ChatGPT Projects, Notion AI | 多格式上传 + Hybrid Search + 代码索引 |
| CI/CD | Jenkins, GitHub Actions | 自然语言配置 + K8s 原生 |
| 部署托管 | Vercel, Railway | 多云 + K8s + 成本优化 |
| 监控 | Grafana Cloud | 内置 OpenTelemetry + AI 异常检测 |
| 数据库 | Supabase, Firebase | PostgreSQL 原生 + 向量扩展 |

## 3. 目标用户

| 用户画像 | 核心场景 |
|----------|----------|
| 独立开发者 | 快速原型 → AI 辅助编码 → 一键上线 |
| 小团队 (2-10 人) | 协作开发 + Agent 自动化 + 共享知识库 |
| 中大型团队 (10-100 人) | RBAC + 审计 + 合规 + 私有部署 |
| AI 应用开发者 | 构建/调试/部署 Agent 和 MCP Server |

## 4. 商业模式

```
Free Tier        → 个人开发者，基础模型额度，公开项目
Pro ($29/月)     → 高级模型，私有项目，更多存储/算力
Team ($19/人/月) → 协作，RBAC，共享资源池
Enterprise       → 私有部署，SLA，专属支持
```

## 5. 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 后端语言 | Go | 高并发、低内存、编译部署简单 |
| 前端框架 | Next.js 15 (App Router) | SSR/SSG、React 生态、API Routes |
| 数据库 | PostgreSQL 16 + pgvector | 关系型 + 向量搜索一体化 |
| 消息队列 | Kafka | 高吞吐、持久化、Exactly-once |
| 对象存储 | MinIO | S3 兼容、自托管、无供应商锁定 |
| 工作流引擎 | Temporal | 可靠长事务、可视化、Go SDK 成熟 |
| 容器编排 | Kubernetes | 行业标准、弹性伸缩、多云统一 |
| AI 接口 | 统一 Adapter 屏蔽差异 | 支持 OpenAI/Anthropic/Gemini/本地 |

## 6. 项目规模估算

| 维度 | 估算 |
|------|------|
| 后端服务数 | 12-15 个微服务 |
| 前端页面数 | 60+ 页面/组件 |
| 数据库表 | 80+ 核心表 |
| API 接口 | 200+ RESTful + 50+ gRPC |
| 开发团队 | 8-15 人（全栈 + AI + DevOps） |
| 首版周期 | 6 个月 MVP → 12 个月完整版 |

## 7. 文档索引

| 文档 | 内容 |
|------|------|
| [01-REQUIREMENTS-ANALYSIS.md](01-REQUIREMENTS-ANALYSIS.md) | 功能/非功能需求详述 |
| [02-FEATURE-BOUNDARY.md](02-FEATURE-BOUNDARY.md) | 模块边界与依赖关系 |
| [03-NON-FUNCTIONAL-REQUIREMENTS.md](03-NON-FUNCTIONAL-REQUIREMENTS.md) | 性能/安全/可用性指标 |
| [04-TECHNOLOGY-SELECTION.md](04-TECHNOLOGY-SELECTION.md) | 技术选型对比分析 |
| [05-SYSTEM-ARCHITECTURE.md](05-SYSTEM-ARCHITECTURE.md) | 系统架构图（Mermaid） |
| [06-DATABASE-DESIGN.md](06-DATABASE-DESIGN.md) | 数据库 ER 设计 |
| [07-DIRECTORY-STRUCTURE.md](07-DIRECTORY-STRUCTURE.md) | 项目目录结构 |
| [08-DEVELOPMENT-STANDARDS.md](08-DEVELOPMENT-STANDARDS.md) | 编码规范与流程 |
| [09-MILESTONE-PLAN.md](09-MILESTONE-PLAN.md) | 里程碑与排期 |
