# OmniDev AI Platform — 项目开发规范

## 项目概述
OmniDev AI Platform：All-in-One AI 开发平台，融合 IDE、Agent、RAG、Workflow、Deploy 等能力。

## 技术栈
- Backend: Go 1.22+ / Gin / gRPC / Wire
- Frontend: Next.js 15 / React 19 / Tailwind / Shadcn/ui
- Database: PostgreSQL 16 + pgvector / Redis 7 / Kafka (KRaft)
- Storage: MinIO (S3 兼容)
- Search: Elasticsearch 8
- Workflow: Temporal
- Infra: Docker / Kubernetes / Helm / Terraform
- Observability: Prometheus / Grafana / Loki / Jaeger / OpenTelemetry

## 开发规则

### 实现节奏
- 每次只实现一个模块，但必须达到生产级质量
- 按架构文档中的里程碑顺序推进（M0 → M1 → M2 → M3 → M4）

### 质量要求（每个模块必须包含）
1. **完整代码** — 不使用 `...`、`TODO`、`placeholder` 等占位符
2. **单元测试** — 覆盖率 > 80%
3. **集成测试** — 核心流程覆盖
4. **API 文档** — OpenAPI 3.0 规范
5. **数据库迁移脚本** — up + down
6. **README** — 模块说明、启动方式、配置项
7. **部署说明** — Docker / K8s 部署步骤
8. **性能优化** — 关键路径优化说明
9. **安全设计** — 认证、授权、输入验证、日志脱敏
10. **故障排查** — 常见问题及解决方案
11. **后续待办** — 本模块未完成的事项

### 输出规则
- 输出到模型最大长度，不压缩、不省略
- 不使用 `...`、`// 省略`、`/* similar code */` 等占位
- 当达到输出上限时，以 `待续：模块 X，第 N 部分` 结束
- 等待用户发送 `继续` 后再继续输出

### 代码风格
- Go: 遵循 Effective Go，golangci-lint 标准
- TypeScript: ESLint + Prettier，strict 模式
- Commit: `<type>(<scope>): <description>` 格式
- 分支: feature/{module}-{desc} → develop → main

### 目录结构
参考 `docs/architecture/07-DIRECTORY-STRUCTURE.md`

### 架构参考
所有架构文档位于 `docs/architecture/` 目录：
- 00-EXECUTIVE-SUMMARY.md — 项目愿景
- 01-REQUIREMENTS-ANALYSIS.md — 需求分析
- 02-FEATURE-BOUNDARY.md — 功能边界
- 03-NON-FUNCTIONAL-REQUIREMENTS.md — 非功能需求
- 04-TECHNOLOGY-SELECTION.md — 技术选型
- 05-SYSTEM-ARCHITECTURE.md — 系统架构
- 06-DATABASE-DESIGN.md — 数据库设计
- 07-DIRECTORY-STRUCTURE.md — 目录结构
- 08-DEVELOPMENT-STANDARDS.md — 开发规范
- 09-MILESTONE-PLAN.md — 里程碑规划
