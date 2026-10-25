-- =============================================================================
-- Migration: Create user_ai_configs table
-- =============================================================================

CREATE TABLE IF NOT EXISTS user_ai_configs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL,
    provider     VARCHAR(50) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    api_key      TEXT NOT NULL,
    base_url     VARCHAR(500) NOT NULL,
    protocol     VARCHAR(20) NOT NULL DEFAULT 'openai',
    models       JSONB NOT NULL DEFAULT '[]',
    is_default   BOOLEAN NOT NULL DEFAULT FALSE,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_user_ai_configs_user_id ON user_ai_configs(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_ai_configs_user_provider ON user_ai_configs(user_id, provider) WHERE deleted_at IS NULL;

-- Updated at trigger
CREATE TRIGGER update_user_ai_configs_updated_at
    BEFORE UPDATE ON user_ai_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
