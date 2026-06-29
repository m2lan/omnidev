-- =============================================================================
-- Migration: Create Workflow tables
-- =============================================================================

CREATE TABLE IF NOT EXISTS workflows (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL,
    org_id        UUID,
    name          VARCHAR(100) NOT NULL,
    description   TEXT,
    definition    JSONB NOT NULL,
    version       INT NOT NULL DEFAULT 1,
    trigger_type  VARCHAR(20) NOT NULL DEFAULT 'manual',
    trigger_config JSONB,
    is_active     BOOLEAN NOT NULL DEFAULT FALSE,
    visibility    VARCHAR(20) NOT NULL DEFAULT 'private',
    tags          TEXT[] NOT NULL DEFAULT '{}',
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_workflows_user ON workflows(user_id) WHERE deleted_at IS NULL;

CREATE TRIGGER update_workflows_updated_at
    BEFORE UPDATE ON workflows FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS workflow_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id     UUID NOT NULL REFERENCES workflows(id),
    user_id         UUID NOT NULL,
    workflow_version INT NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    trigger_type    VARCHAR(20) NOT NULL,
    input           JSONB,
    output          JSONB,
    error           TEXT,
    total_nodes     INT NOT NULL DEFAULT 0,
    completed_nodes INT NOT NULL DEFAULT 0,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wf_runs_workflow ON workflow_runs(workflow_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wf_runs_status ON workflow_runs(status) WHERE status IN ('pending', 'running');

CREATE TABLE IF NOT EXISTS workflow_node_runs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id        UUID NOT NULL REFERENCES workflow_runs(id),
    node_id       VARCHAR(100) NOT NULL,
    node_type     VARCHAR(50) NOT NULL,
    node_name     VARCHAR(100),
    status        VARCHAR(20) NOT NULL DEFAULT 'pending',
    input         JSONB,
    output        JSONB,
    error         TEXT,
    retry_count   INT NOT NULL DEFAULT 0,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    latency_ms    INT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(run_id, node_id)
);

CREATE INDEX IF NOT EXISTS idx_wf_node_runs_run ON workflow_node_runs(run_id);
