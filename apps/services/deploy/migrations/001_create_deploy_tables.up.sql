-- =============================================================================
-- Migration: Create Deploy tables
-- =============================================================================

CREATE TABLE IF NOT EXISTS deployments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL,
    user_id         UUID NOT NULL,
    version         VARCHAR(100) NOT NULL,
    environment     VARCHAR(20) NOT NULL DEFAULT 'development',
    platform        VARCHAR(20) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    config          JSONB NOT NULL DEFAULT '{}',
    domain          VARCHAR(255),
    url             VARCHAR(500),
    image_tag       VARCHAR(255),
    build_logs      TEXT,
    error           TEXT,
    resource_usage  JSONB,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deployments_project ON deployments(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_deployments_user ON deployments(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status) WHERE status IN ('pending', 'building', 'deploying');

CREATE TRIGGER update_deployments_updated_at
    BEFORE UPDATE ON deployments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
