-- =============================================================================
-- Migration: Create Agent tables
-- =============================================================================

-- Agents
CREATE TABLE IF NOT EXISTS agents (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL,
    org_id        UUID,
    name          VARCHAR(100) NOT NULL,
    description   TEXT,
    avatar_url    VARCHAR(500),
    system_prompt TEXT NOT NULL,
    model_id      UUID,
    tools         JSONB NOT NULL DEFAULT '[]',
    mcp_servers   JSONB NOT NULL DEFAULT '[]',
    config        JSONB NOT NULL DEFAULT '{"max_steps": 20}',
    visibility    VARCHAR(20) NOT NULL DEFAULT 'private',
    is_template   BOOLEAN NOT NULL DEFAULT FALSE,
    template_category VARCHAR(50),
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_agents_user ON agents(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_agents_template ON agents(is_template, template_category) WHERE deleted_at IS NULL;

CREATE TRIGGER update_agents_updated_at
    BEFORE UPDATE ON agents FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Agent Runs
CREATE TABLE IF NOT EXISTS agent_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        UUID NOT NULL REFERENCES agents(id),
    user_id         UUID NOT NULL,
    task            TEXT NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'created',
    result          TEXT,
    error           TEXT,
    total_steps     INT NOT NULL DEFAULT 0,
    completed_steps INT NOT NULL DEFAULT 0,
    token_input     INT NOT NULL DEFAULT 0,
    token_output    INT NOT NULL DEFAULT 0,
    cost            DECIMAL(10,6) NOT NULL DEFAULT 0,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_agent ON agent_runs(agent_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_runs_user ON agent_runs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_runs_status ON agent_runs(status) WHERE status IN ('created', 'planning', 'executing');

CREATE TRIGGER update_agent_runs_updated_at
    BEFORE UPDATE ON agent_runs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Agent Steps
CREATE TABLE IF NOT EXISTS agent_steps (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id        UUID NOT NULL REFERENCES agent_runs(id),
    step_number   INT NOT NULL,
    type          VARCHAR(30) NOT NULL,
    content       TEXT,
    tool_name     VARCHAR(100),
    tool_input    JSONB,
    tool_output   JSONB,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending',
    error         TEXT,
    token_input   INT,
    token_output  INT,
    latency_ms    INT,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(run_id, step_number)
);

CREATE INDEX IF NOT EXISTS idx_agent_steps_run ON agent_steps(run_id, step_number);
