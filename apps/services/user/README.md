# User Service

用户中心服务，负责认证、授权、用户管理和组织管理。

## 功能

- 用户注册/登录（邮箱 + 密码）
- OAuth 登录（GitHub, Google）
- JWT Token 管理（签发/刷新/撤销）
- API Key 管理（创建/列表/撤销）
- RBAC 角色权限
- 组织管理（创建/成员邀请/角色管理）
- 用户个人设置

## API 端点

### 认证（公开）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/register` | 用户注册 |
| POST | `/api/v1/auth/login` | 用户登录 |
| POST | `/api/v1/auth/refresh` | 刷新 Token |
| GET | `/api/v1/auth/oauth/:provider` | OAuth 跳转 |
| GET | `/api/v1/auth/callback/:provider` | OAuth 回调 |

### 用户（需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/users/me` | 获取个人资料 |
| PATCH | `/api/v1/users/me` | 更新个人资料 |
| GET | `/api/v1/users/me/api-keys` | 列出 API Key |
| POST | `/api/v1/users/me/api-keys` | 创建 API Key |
| DELETE | `/api/v1/users/me/api-keys/:id` | 撤销 API Key |

### 组织（需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/organizations` | 列出我的组织 |
| POST | `/api/v1/organizations` | 创建组织 |
| GET | `/api/v1/organizations/:id` | 获取组织详情 |
| PATCH | `/api/v1/organizations/:id` | 更新组织 |
| GET | `/api/v1/organizations/:id/members` | 列出成员 |
| POST | `/api/v1/organizations/:id/members/invite` | 邀请成员 |
| PATCH | `/api/v1/organizations/:id/members/:user_id/role` | 更新成员角色 |
| DELETE | `/api/v1/organizations/:id/members/:user_id` | 移除成员 |

## 请求示例

### 注册

```bash
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!",
    "nickname": "TestUser"
  }'
```

### 登录

```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!"
  }'
```

### 获取个人资料

```bash
curl http://localhost:8081/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"
```

## 配置

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| APP_PORT | 8081 | 服务端口 |
| DB_HOST | localhost | PostgreSQL 地址 |
| DB_PORT | 5432 | PostgreSQL 端口 |
| DB_USER | omnidev | 数据库用户 |
| DB_PASSWORD | omnidev | 数据库密码 |
| DB_NAME | omnidev | 数据库名 |
| REDIS_HOST | localhost | Redis 地址 |
| JWT_SECRET | - | JWT 签名密钥 |
| JWT_ACCESS_EXPIRY | 15m | Access Token 有效期 |
| JWT_REFRESH_EXPIRY | 168h | Refresh Token 有效期 |

## 运行

```bash
# 本地运行
go run cmd/user/main.go

# Docker
docker build -t omnidev/user-service .
docker run -p 8081:8081 omnidev/user-service
```

## 测试

```bash
# 单元测试
go test ./... -v

# 集成测试
go test ./... -v -tags=integration -count=1

# 覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 安全设计

- 密码使用 bcrypt (cost=12) 哈希存储
- JWT 使用 HS256 签名，Access Token 15分钟，Refresh Token 7天
- API Key 使用 bcrypt 哈希存储，仅首次创建时返回明文
- Refresh Token 支持黑名单机制（Redis）
- 用户会话缓存在 Redis 中

## 故障排查

### 数据库连接失败
```
检查 PostgreSQL 是否运行：docker compose ps postgres
检查连接配置：DB_HOST, DB_PORT, DB_USER, DB_PASSWORD
```

### JWT 验证失败
```
确保 JWT_SECRET 在所有服务间一致
检查 Token 是否过期
检查 Token 类型（access vs refresh）
```

### 邮箱已注册
```
用户尝试使用已注册的邮箱注册
引导用户使用登录或找回密码功能
```

## 后续待办

- [ ] 邮箱验证流程
- [ ] 密码重置功能
- [ ] OAuth 完整实现（GitHub, Google）
- [ ] 双因素认证（2FA）
- [ ] 登录设备管理
- [ ] 用户头像上传（MinIO）
- [ ] 更细粒度的 RBAC 权限
- [ ] 审计日志集成
