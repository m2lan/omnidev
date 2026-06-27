# Chat Service

AI 对话服务，支持多模型、流式输出、会话管理和 Prompt 模板。

## 功能

- 多模型支持（OpenAI, Anthropic, DeepSeek, Qwen, Ollama）
- 流式输出（SSE）
- 多会话管理
- 对话历史
- Prompt 模板（创建/管理/Fork）
- Token 计量
- 自动标题生成

## API 端点

### 会话管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/conversations` | 列出会话 |
| POST | `/api/v1/conversations` | 创建会话 |
| GET | `/api/v1/conversations/:id` | 获取会话 |
| PATCH | `/api/v1/conversations/:id` | 更新会话 |
| DELETE | `/api/v1/conversations/:id` | 删除会话 |
| GET | `/api/v1/conversations/:id/messages` | 列出消息 |
| POST | `/api/v1/conversations/:id/messages` | 发送消息 |
| POST | `/api/v1/conversations/:id/messages/stream` | 流式对话 |

### Prompt 模板

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/prompts` | 列出 Prompt |
| POST | `/api/v1/prompts` | 创建 Prompt |
| GET | `/api/v1/prompts/:id` | 获取 Prompt |
| PATCH | `/api/v1/prompts/:id` | 更新 Prompt |
| DELETE | `/api/v1/prompts/:id` | 删除 Prompt |

### 模型

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/models` | 列出可用模型 |

## 请求示例

### 创建会话

```bash
curl -X POST http://localhost:8082/api/v1/conversations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My Chat",
    "system_prompt": "You are a helpful assistant.",
    "tags": ["coding", "help"]
  }'
```

### 发送消息

```bash
curl -X POST http://localhost:8082/api/v1/conversations/<id>/messages \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Hello, how are you?",
    "model_id": "gpt-4o-mini"
  }'
```

### 流式对话

```bash
curl -X POST http://localhost:8082/api/v1/conversations/<id>/messages/stream \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"content": "Tell me a story"}'
```

## 支持的模型

| Provider | 模型 |
|----------|------|
| OpenAI | gpt-4o, gpt-4o-mini, gpt-4-turbo |
| Anthropic | claude-3-5-sonnet, claude-3-opus |
| DeepSeek | deepseek-chat, deepseek-coder |
| Qwen | qwen-turbo, qwen-plus, qwen-max |
| Ollama | llama3, mistral, codellama |

## 安全设计

- 所有端点需要 JWT 认证
- 用户只能访问自己的会话
- AI API Key 不暴露给客户端
- 请求限流保护

## 后续待办

- [ ] RAG 检索增强（@引用知识库）
- [ ] 多模态输入（图片/文件）
- [ ] 工具调用（Function Calling）
- [ ] 对话分支
- [ ] 消息编辑/重新生成
- [ ] Prompt 版本管理
- [ ] Prompt Compare（多模型对比）
- [ ] 语义缓存
- [ ] 更多模型提供商
