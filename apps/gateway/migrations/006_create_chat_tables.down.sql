DROP TRIGGER IF EXISTS update_prompt_templates_updated_at ON prompt_templates;
DROP TABLE IF EXISTS prompt_templates CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TRIGGER IF EXISTS update_conversations_updated_at ON conversations;
DROP TABLE IF EXISTS conversations CASCADE;
DROP TRIGGER IF EXISTS update_models_updated_at ON models;
DROP TABLE IF EXISTS models CASCADE;
