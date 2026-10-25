-- Rollback: Drop user_ai_configs table
DROP TRIGGER IF EXISTS update_user_ai_configs_updated_at ON user_ai_configs;
DROP TABLE IF EXISTS user_ai_configs CASCADE;
