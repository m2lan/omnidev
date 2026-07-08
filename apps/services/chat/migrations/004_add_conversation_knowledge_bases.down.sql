-- Remove knowledge_base_ids column from conversations
DROP INDEX IF EXISTS idx_conversations_knowledge_base_ids;
ALTER TABLE conversations DROP COLUMN IF EXISTS knowledge_base_ids;
