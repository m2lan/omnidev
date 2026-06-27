-- =============================================================================
-- Migration: Create OAuth connections table
-- =============================================================================

CREATE TABLE IF NOT EXISTS oauth_connections (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider      VARCHAR(50) NOT NULL,
    provider_uid  VARCHAR(255) NOT NULL,
    access_token  TEXT,
    refresh_token TEXT,
    expires_at    TIMESTAMPTZ,
    scope         TEXT,
    raw_profile   JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(provider, provider_uid)
);

CREATE INDEX IF NOT EXISTS idx_oauth_user ON oauth_connections(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_provider ON oauth_connections(provider, provider_uid);

CREATE TRIGGER update_oauth_connections_updated_at
    BEFORE UPDATE ON oauth_connections
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
