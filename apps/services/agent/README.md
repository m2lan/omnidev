# Agent Service

AI Agent 执行服务，支持任务规划、工具调用、代码沙箱执行。

## 功能

- Agent 定义与管理
- 任务自动规划（AI Planner）
- 工具调用（File/Calculator/Code/Search）
- 代码沙箱执行（Python/JS/Go/Shell）
- 执行状态追踪
- 步骤日志

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/agents` | 列出 Agents |
| POST | `/api/v1/agents` | 创建 Agent |
| GET | `/api/v1/agents/:id` | 获取 Agent |
| PATCH | `/api/v1/agents/:id` | 更新 Agent |
| DELETE | `/api/v1/agents/:id` | 删除 Agent |
| POST | `/api/v1/agents/:id/run` | 运行 Agent |
| GET | `/api/v1/agents/:id/runs` | 列出执行记录 |
| GET | `/api/v1/agents/runs/:run_id` | 获取执行详情 |
| POST | `/api/v1/agents/runs/:run_id/cancel` | 取消执行 |

## 内置工具

| 工具 | 说明 |
|------|------|
| `file` | 文件读写、列表、删除 |
| `search` | 文件内容搜索 |
| `calculator` | 数学计算 |
| `code_exec` | 代码执行（Python/JS/Go/Shell） |

## 执行流程

```
用户任务 → AI Planner → 步骤列表 → 逐步执行
                                      ↓
                              [工具调用/代码执行/思考]
                                      ↓
                              结果回传 → 下一步规划
                                      ↓
                              完成 → 最终回答
```

## 请求示例

### 创建 Agent

```bash
curl -X POST http://localhost:8084/api/v1/agents \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Code Assistant",
    "description": "Helps with coding tasks",
    "system_prompt": "You are a helpful coding assistant.",
    "tools": [
      {"name": "file", "enabled": true},
      {"name": "code_exec", "enabled": true}
    ],
    "config": {"max_steps": 10}
  }'
```

### 运行 Agent

```bash
curl -X POST http://localhost:8084/api/v1/agents/<id>/run \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"task": "Write a Python script that calculates fibonacci numbers"}'
```

## 后续待办

- [ ] MCP 工具集成
- [ ] Docker 沙箱隔离
- [ ] 多 Agent 协作
- [ ] Agent 模板市场
- [ ] 可视化执行流程
- [ ] 执行历史回放
- [ ] 工具权限控制
