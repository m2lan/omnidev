# OmniDev AI Platform

[English](README.md)

All-in-One AI 开发平台，融合 IDE、Agent、RAG、Workflow、Deploy 等能力。

## 快速开始

### 前置条件

- Go 1.22+
- Node.js 20+
- pnpm 9+
- Docker & Docker Compose
- Make

### 1. 克隆项目

```bash
git clone git@github.com:m2lan/omnidev.git
cd omnidev
```

### 2. 环境配置

```bash
cp .env.example .env
# 编辑 .env 设置你的 API Key 等配置
```

### 3. 启动基础设施

```bash
# 启动 PostgreSQL, Redis, Kafka, MinIO, Elasticsearch, Temporal
make dev-infra

# 查看状态
make dev-status
```

### 4. 安装依赖

```bash
# Go 依赖
make go-mod

# 前端依赖
make web-install
```

### 5. 运行数据库迁移

```bash
make db-migrate
```

### 6. 启动开发服务器

```bash
# 启动 API Gateway
make run-gateway

# 启动前端 (另一个终端)
make web-dev
```

### 7. 访问

| 服务 | 地址 |
|------|------|
| Web App | http://localhost:3000 |
| API Gateway | http://localhost:9090 |
| Health Check | http://localhost:9090/health |
| API Info | http://localhost:9090/info |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |
| Kafka | localhost:9092 |
| MinIO Console | http://localhost:9001 |
| Elasticsearch | http://localhost:9200 |
| Temporal UI | http://localhost:8088 |
| Grafana | http://localhost:3001 |
| Prometheus | http://localhost:9091 |

## 项目结构

```
omnidev/
├── apps/                    # 应用层
│   ├── web/                 # 前端 Next.js
│   ├── gateway/             # API Gateway (Go)
│   ├── services/            # 后端微服务
│   └── workers/             # 后台工作者
├── packages/                # 共享包
│   ├── proto/               # Protobuf 定义
│   ├── go-common/           # Go 公共库
│   ├── ts-common/           # TypeScript 公共库
│   └── ui/                  # UI 组件库
├── deploy/                  # 部署配置
│   ├── docker/              # Docker Compose
│   ├── helm/                # Helm Charts
│   ├── terraform/           # Terraform 配置
│   └── k8s/                 # K8s 清单
├── docs/                    # 文档
├── scripts/                 # 工具脚本
└── tools/                   # 开发工具
```

## 开发命令

```bash
# 查看所有命令
make help

# 开发环境
make dev              # 启动完整开发环境
make dev-infra        # 仅启动基础设施
make dev-down         # 停止开发环境
make dev-logs         # 查看日志

# 构建
make build-all        # 构建所有服务
make build-gateway    # 构建 API Gateway

# 测试
make test             # 运行所有测试
make test-short       # 运行短测试
make test-integration # 运行集成测试
make test-coverage    # 生成覆盖率报告

# 代码质量
make lint             # 运行 linter
make fmt              # 格式化代码
make check            # 运行所有检查

# 代码生成
make gen-proto        # 生成 Protobuf 代码
make gen-wire         # 生成 Wire 代码
make gen-swagger      # 生成 API 文档

# 数据库
make db-migrate       # 运行迁移
make db-migrate-down  # 回滚迁移
make db-migrate-create NAME=create_xxx  # 创建新迁移

# Docker
make docker-build     # 构建所有 Docker 镜像
make docker-build-gateway  # 构建单个镜像

# Kubernetes
make k8s-apply        # 部署到 K8s
make helm-install     # 安装 Helm Chart
```

## 技术栈

| 层级 | 技术 |
|------|------|
| Frontend | Next.js 15, React 19, Tailwind, Shadcn/ui |
| API Gateway | Go, Gin |
| Backend | Go, gRPC |
| Database | PostgreSQL 16 + pgvector |
| Cache | Redis 7 |
| Queue | Kafka (KRaft) |
| Storage | MinIO |
| Search | Elasticsearch 8 |
| Workflow | Temporal |
| Container | Docker, Kubernetes |
| Observability | Prometheus, Grafana, Loki, Jaeger |

## 架构文档

详细架构文档位于 `docs/architecture/` 目录：

- [执行摘要](docs/architecture/00-EXECUTIVE-SUMMARY.md)
- [需求分析](docs/architecture/01-REQUIREMENTS-ANALYSIS.md)
- [功能边界](docs/architecture/02-FEATURE-BOUNDARY.md)
- [非功能需求](docs/architecture/03-NON-FUNCTIONAL-REQUIREMENTS.md)
- [技术选型](docs/architecture/04-TECHNOLOGY-SELECTION.md)
- [系统架构](docs/architecture/05-SYSTEM-ARCHITECTURE.md)
- [数据库设计](docs/architecture/06-DATABASE-DESIGN.md)
- [目录结构](docs/architecture/07-DIRECTORY-STRUCTURE.md)
- [开发规范](docs/architecture/08-DEVELOPMENT-STANDARDS.md)
- [里程碑规划](docs/architecture/09-MILESTONE-PLAN.md)

## API 文档

API 遵循 RESTful 规范，所有响应格式：

```json
// 成功
{
  "data": { ... },
  "meta": { "page_size": 20, "next_page_token": "...", "total_count": 100 }
}

// 错误
{
  "error": { "code": 400, "message": "...", "detail": "...", "request_id": "..." }
}
```

### 认证

```bash
# Bearer Token
curl -H "Authorization: Bearer <token>" http://localhost:9090/api/v1/users/me

# API Key
curl -H "Authorization: Bearer <api-key>" http://localhost:9090/api/v1/users/me
```

## 故障排查

### 数据库连接失败

```bash
# 检查 PostgreSQL 是否运行
docker compose -f deploy/docker/docker-compose.infra.yml ps postgres

# 查看日志
docker compose -f deploy/docker/docker-compose.infra.yml logs postgres
```

### Redis 连接失败

```bash
# 检查 Redis 是否运行
docker compose -f deploy/docker/docker-compose.infra.yml ps redis

# 测试连接
docker compose -f deploy/docker/docker-compose.infra.yml exec redis redis-cli ping
```

### 端口被占用

```bash
# 查看占用端口的进程
netstat -ano | findstr :9090
# 或
lsof -i :9090

# 杀死进程
taskkill /PID <pid> /F
```

### Go 模块问题

```bash
# 清理模块缓存
go clean -modcache

# 重新下载
go mod download all

# 同步 workspace
go work sync
```

## 贡献指南

1. Fork 项目
2. 创建功能分支: `git checkout -b feature/my-feature`
3. 提交更改: `git commit -m 'feat(scope): add my feature'`
4. 推送分支: `git push origin feature/my-feature`
5. 创建 Pull Request

## 许可证

[MIT License](LICENSE)
