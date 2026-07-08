-- Add knowledge_base_ids column to conversations for persistent RAG binding
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS knowledge_base_ids UUID[] NOT NULL DEFAULT '{}';
CREATE INDEX IF NOT EXISTS idx_conversations_knowledge_base_ids ON conversations USING GIN (knowledge_base_ids);
