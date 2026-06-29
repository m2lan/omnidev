-- =============================================================================
-- Migration: Create MCP tables
-- =============================================================================

CREATE TABLE IF NOT EXISTS mcp_servers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    org_id          UUID,
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    transport       VARCHAR(20) NOT NULL,
    endpoint        VARCHAR(500),
    command         VARCHAR(500),
    args            TEXT[],
    env             JSONB,
    is_builtin      BOOLEAN NOT NULL DEFAULT FALSE,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    tool_count      INT NOT NULL DEFAULT 0,
    last_health_check TIMESTAMPTZ,
    health_status   VARCHAR(20) DEFAULT 'unknown',
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_mcp_servers_user ON mcp_servers(user_id) WHERE deleted_at IS NULL;

CREATE TRIGGER update_mcp_servers_updated_at
    BEFORE UPDATE ON mcp_servers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS mcp_tools (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id     UUID NOT NULL REFERENCES mcp_servers(id) ON DELETE CASCADE,
    name          VARCHAR(100) NOT NULL,
    description   TEXT,
    input_schema  JSONB NOT NULL DEFAULT '{}',
    output_schema JSONB,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    call_count    BIGINT NOT NULL DEFAULT 0,
    avg_latency_ms INT,
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(server_id, name)
);

CREATE INDEX IF NOT EXISTS idx_mcp_tools_server ON mcp_tools(server_id);

CREATE TRIGGER update_mcp_tools_updated_at
    BEFORE UPDATE ON mcp_tools FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
