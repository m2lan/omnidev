# RAG Service

RAG (Retrieval-Augmented Generation) 服务，提供文档上传、解析、分块、向量化和混合检索能力。

## 功能

- 知识库管理（CRUD）
- 多格式文档上传（PDF/Word/PPT/Excel/Markdown/代码）
- 文档解析（OCR 支持）
- 语义分块（Semantic Chunking）
- 向量嵌入（OpenAI Embedding）
- 混合检索（向量相似度 + BM25 关键词）
- Reciprocal Rank Fusion (RRF) 排序
- 引用溯源

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/knowledge` | 列出知识库 |
| POST | `/api/v1/knowledge` | 创建知识库 |
| GET | `/api/v1/knowledge/:id` | 获取知识库 |
| PATCH | `/api/v1/knowledge/:id` | 更新知识库 |
| DELETE | `/api/v1/knowledge/:id` | 删除知识库 |
| GET | `/api/v1/knowledge/:id/documents` | 列出文档 |
| POST | `/api/v1/knowledge/:id/documents` | 上传文档 |
| DELETE | `/api/v1/knowledge/:id/documents/:doc_id` | 删除文档 |
| POST | `/api/v1/knowledge/:id/search` | 混合检索 |

## 请求示例

### 创建知识库

```bash
curl -X POST http://localhost:8083/api/v1/knowledge \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Knowledge Base",
    "description": "Project documentation",
    "chunk_size": 512,
    "chunk_overlap": 50
  }'
```

### 上传文档

```bash
curl -X POST http://localhost:8083/api/v1/knowledge/<id>/documents \
  -H "Authorization: Bearer <token>" \
  -F "file=@document.pdf"
```

### 搜索

```bash
curl -X POST http://localhost:8083/api/v1/knowledge/<id>/search \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "How to configure authentication?",
    "top_k": 5,
    "min_score": 0.3
  }'
```

## RAG Pipeline

```
文档上传 → MinIO 存储 → 文档解析 → 语义分块 → Embedding → pgvector 存储
                                                              ↓
用户查询 → Query Embedding → 向量检索 (Top-K) ← BM25 检索 → Rerank → 结果
```

## 支持的文件格式

| 格式 | 扩展名 | 解析方式 |
|------|--------|----------|
| 纯文本 | .txt, .md | 直接读取 |
| PDF | .pdf | 文本提取 |
| Word | .docx | XML 解析 |
| PowerPoint | .pptx | XML 解析 |
| Excel | .xlsx, .csv | 直接读取 |
| 代码 | .go, .py, .js, .ts 等 | 直接读取 |
| HTML | .html, .htm | 标签剥离 |
| JSON | .json | 直接读取 |

## 安全设计

- 所有端点需要 JWT 认证
- 用户只能访问自己的知识库
- 文件上传大小限制 100MB
- MinIO 存储隔离

## 性能优化

- 批量 Embedding（OpenAI 支持 2048 条/批）
- HNSW 向量索引（pgvector）
- 批量 Chunk 插入
- 异步文档处理

## 后续待办

- [ ] OCR 支持（扫描件 PDF）
- [ ] GitHub 仓库导入
- [ ] Rerank 重排序模型
- [ ] 多 Embedding 模型支持
- [ ] 文档增量更新
- [ ] 知识库共享
- [ ] 代码 AST 解析
- [ ] 引用高亮
