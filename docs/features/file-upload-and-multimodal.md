# 文件上传与多模态消息功能

## 功能概述

文件上传功能允许用户在聊天中发送图片、文档等文件，并通过多模态消息格式传递给 AI 模型进行分析。

## 架构设计

```
┌──────────────────────────────────────────────────────────────────┐
│                          Frontend                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ FileUpload  │  │ Attachment  │  │     ChatArea            │  │
│  │   Button    │→ │   Preview   │→ │  (with attachments)     │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└────────────────────────────┬─────────────────────────────────────┘
                             │ POST /api/v1/upload
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│                         API Gateway                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Upload    │  │  Attachment │  │     ChatService         │  │
│  │   Handler   │  │  Repository │  │  (multimodal messages)  │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└────────────────────────────┬─────────────────────────────────────┘
                             │
        ┌────────────────────┼────────────────────┐
        ▼                    ▼                    ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│    MinIO     │    │  PostgreSQL  │    │   Apache     │
│  (文件存储)   │    │  (元数据)    │    │   Tika       │
│              │    │              │    │  (文档解析)   │
└──────────────┘    └──────────────┘    └──────────────┘
```

## 支持的文件类型

### 图片类型（多模态）
- JPEG/JPG
- PNG
- GIF
- WebP

### 文档类型（文本提取）
- PDF
- Word (.doc, .docx)
- Excel (.xls, .xlsx)
- PowerPoint (.ppt, .pptx)
- 文本文件 (.txt)
- Markdown (.md)
- CSV

## API 接口

### 1. 上传文件

**POST** `/api/v1/upload`

**请求**：`multipart/form-data`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | File | 是 | 要上传的文件 |

**响应**：
```json
{
  "data": {
    "id": "uuid",
    "user_id": "uuid",
    "filename": "example.pdf",
    "mime_type": "application/pdf",
    "file_size": 12345,
    "storage_url": "https://...",
    "width": null,
    "height": null,
    "created_at": "2026-07-07T10:00:00Z"
  }
}
```

### 2. 发送消息（带附件）

**POST** `/api/v1/conversations/:id/messages/stream`

**请求体**：
```json
{
  "content": "分析这个文档",
  "model_id": "gpt-4o",
  "attachment_ids": ["uuid1", "uuid2"]
}
```

### 3. 获取附件

**GET** `/api/v1/attachments/:id`

### 4. 删除附件

**DELETE** `/api/v1/attachments/:id`

## 数据库设计

### attachments 表

```sql
CREATE TABLE attachments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    conversation_id UUID REFERENCES conversations(id),
    message_id      UUID,
    filename        VARCHAR(255) NOT NULL,
    mime_type       VARCHAR(100) NOT NULL,
    file_size       BIGINT NOT NULL,
    storage_key     VARCHAR(500) NOT NULL,
    storage_url     VARCHAR(500) NOT NULL,
    thumbnail_key   VARCHAR(500),
    width           INT,
    height          INT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
```

## 文件处理流程

### 图片处理

```
1. 用户上传图片 → MinIO 存储
2. 保存元数据到 attachments 表
3. 发送消息时：
   - 从 MinIO 下载图片
   - 转换为 base64 编码
   - 使用多模态格式传给 AI：
     {
       "role": "user",
       "content": [
         {"type": "text", "text": "分析这张图片"},
         {"type": "image_url", "image_url": {"url": "data:image/png;base64,..."}}
       ]
     }
```

### 文档处理

```
1. 用户上传文档 → MinIO 存储
2. 保存元数据到 attachments 表
3. 发送消息时：
   - 从 MinIO 下载文档
   - 调用 Apache Tika 提取文本
   - 缓存解析结果到 Redis（7天 TTL）
   - 将提取的文本附加到消息内容：
     "分析这个文档\n\n[附件: report.pdf]\n提取的文本内容..."
```

## 配置项

### .env 文件

```env
# Apache Tika 文档解析服务
TIKA_ENDPOINT=http://localhost:9998
TIKA_TIMEOUT=60

# MinIO 对象存储
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET_PREFIX=omnidev
```

## Docker 部署

### 启动 Tika 服务

```bash
docker compose -f deploy/docker/docker-compose.infra.yml up -d tika
```

### 验证 Tika 运行

```bash
curl http://localhost:9998/version
# 应返回: Welcome to the Apache Tika 3.3.1 Server
```

## 前端组件

### FileUploadButton

文件上传按钮组件，支持：
- 多文件选择
- 文件类型验证
- 文件大小限制（20MB）
- 上传进度显示

### AttachmentPreviewList

附件预览列表组件：
- 图片显示缩略图
- 文档显示文件图标和大小
- 支持删除单个附件

### MessageBubble（扩展）

消息气泡组件扩展：
- 图片内联显示（点击可放大）
- 文档显示为可点击卡片

## 性能优化

1. **Redis 缓存**：文档解析结果缓存 7天，避免重复解析
2. **图片压缩**：上传时可选压缩（待实现）
3. **懒加载**：历史消息中的图片懒加载
4. **预签名 URL**：MinIO 文件访问使用预签名 URL（7天有效）

## 安全设计

1. **文件类型白名单**：只允许指定的 MIME 类型
2. **文件大小限制**：单文件最大 20MB
3. **用户隔离**：用户只能访问自己上传的文件
4. **预签名 URL**：文件访问通过预签名 URL，不暴露 MinIO 内部地址

## 故障排查

### 问题：上传失败

**检查项**：
1. MinIO 服务是否运行：`docker ps | grep minio`
2. MinIO 配置是否正确：检查 `.env` 中的 MINIO_* 配置
3. 文件大小是否超限：最大 20MB
4. 文件类型是否支持：检查白名单

### 问题：文档解析失败

**检查项**：
1. Tika 服务是否运行：`curl http://localhost:9998/version`
2. Tika 配置是否正确：检查 `.env` 中的 TIKA_ENDPOINT
3. 查看 Gateway 日志中的错误信息

### 问题：AI 无法理解图片

**检查项**：
1. 模型是否支持 vision（如 GPT-4o、Claude 3.5 Sonnet）
2. 查看日志中是否有 `Parsing document with Tika` 或 `imageToBase64` 相关日志
3. 检查附件是否正确关联到消息

## 后续待办

- [ ] 图片压缩和缩略图生成
- [ ] 支持更多文档格式（EPUB、RTF）
- [ ] OCR 支持（扫描件 PDF）
- [ ] 文件预览页面
- [ ] 批量上传
- [ ] 上传进度条
- [ ] 断点续传
- [ ] 文件版本管理

## 相关文件

### 后端

| 文件 | 说明 |
|------|------|
| `packages/go-common/parser/parser.go` | Parser 接口定义 |
| `packages/go-common/parser/tika.go` | Tika 客户端实现 |
| `packages/go-common/parser/cache.go` | 解析结果缓存 |
| `apps/gateway/internal/handler/upload.go` | 上传 API Handler |
| `apps/gateway/internal/service/upload_service.go` | 上传服务 |
| `apps/gateway/internal/repository/attachment_repo.go` | 附件数据访问 |
| `apps/gateway/internal/domain/conversation.go` | Attachment 实体 |
| `apps/gateway/internal/service/chat_service.go` | 多模态消息处理 |
| `apps/gateway/internal/adapter/adapter.go` | 多模态消息格式 |

### 前端

| 文件 | 说明 |
|------|------|
| `apps/web/src/components/chat/file-upload-button.tsx` | 文件上传组件 |
| `apps/web/src/components/chat/chat-area.tsx` | 聊天区域（集成上传） |
| `apps/web/src/components/chat/message-bubble.tsx` | 消息气泡（附件展示） |
| `apps/web/src/stores/chat-store.ts` | 聊天状态管理 |
| `apps/web/src/lib/api/client.ts` | API 客户端 |

### 数据库

| 文件 | 说明 |
|------|------|
| `apps/services/chat/migrations/002_create_attachments.up.sql` | 建表脚本 |
| `apps/services/chat/migrations/002_create_attachments.down.sql` | 回滚脚本 |

### 基础设施

| 文件 | 说明 |
|------|------|
| `deploy/docker/docker-compose.infra.yml` | Tika Docker 配置 |
| `packages/go-common/config/config.go` | Tika 配置定义 |
| `packages/go-common/config/loader.go` | 环境变量映射 |
