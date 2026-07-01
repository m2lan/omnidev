-- =============================================================================
-- Migration: Create models, conversations, messages tables
-- =============================================================================

-- AI Models
CREATE TABLE IF NOT EXISTS models (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider      VARCHAR(50) NOT NULL,
    model_id      VARCHAR(100) NOT NULL,
    display_name  VARCHAR(100) NOT NULL,
    description   TEXT,
    context_window INT NOT NULL DEFAULT 4096,
    max_output    INT NOT NULL DEFAULT 4096,
    supports_streaming BOOLEAN NOT NULL DEFAULT TRUE,
    supports_vision BOOLEAN NOT NULL DEFAULT FALSE,
    supports_tools  BOOLEAN NOT NULL DEFAULT FALSE,
    input_price   DECIMAL(10,6),
    output_price  DECIMAL(10,6),
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    config        JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(provider, model_id)
);

CREATE TRIGGER update_models_updated_at
    BEFORE UPDATE ON models
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Conversations
CREATE TABLE IF NOT EXISTS conversations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL,
    org_id        UUID,
    title         VARCHAR(255),
    model_id      UUID REFERENCES models(id),
    system_prompt TEXT,
    settings      JSONB NOT NULL DEFAULT '{}',
    status        VARCHAR(20) NOT NULL DEFAULT 'active',
    pinned        BOOLEAN NOT NULL DEFAULT FALSE,
    tags          TEXT[] NOT NULL DEFAULT '{}',
    message_count INT NOT NULL DEFAULT 0,
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_conversations_user ON conversations(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_conversations_org ON conversations(org_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_conversations_created ON conversations(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversations_status ON conversations(status) WHERE deleted_at IS NULL;

CREATE TRIGGER update_conversations_updated_at
    BEFORE UPDATE ON conversations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Messages (partitioned by month)
CREATE TABLE IF NOT EXISTS messages (
    id            UUID NOT NULL DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL,
    role          VARCHAR(20) NOT NULL,
    content       TEXT NOT NULL,
    model_id      VARCHAR(100),
    token_input   INT,
    token_output  INT,
    latency_ms    INT,
    tool_calls    JSONB,
    tool_call_id  VARCHAR(255),
    parent_id     UUID,
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create partitions for current and next 3 months
DO $$
DECLARE
    start_date DATE;
    end_date DATE;
    partition_name TEXT;
BEGIN
    FOR i IN 0..3 LOOP
        start_date := DATE_TRUNC('month', CURRENT_DATE) + (i || ' months')::INTERVAL;
        end_date := start_date + '1 month'::INTERVAL;
        partition_name := 'messages_' || TO_CHAR(start_date, 'YYYY_MM');

        EXECUTE FORMAT('
            CREATE TABLE IF NOT EXISTS %I PARTITION OF messages
            FOR VALUES FROM (%L) TO (%L)',
            partition_name, start_date, end_date
        );
    END LOOP;
END $$;

CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_role ON messages(role);

-- Prompt templates
CREATE TABLE IF NOT EXISTS prompt_templates (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL,
    org_id        UUID,
    title         VARCHAR(255) NOT NULL,
    content       TEXT NOT NULL,
    description   TEXT,
    category      VARCHAR(50),
    tags          TEXT[] NOT NULL DEFAULT '{}',
    variables     JSONB NOT NULL DEFAULT '[]',
    visibility    VARCHAR(20) NOT NULL DEFAULT 'private',
    version       INT NOT NULL DEFAULT 1,
    fork_from     UUID REFERENCES prompt_templates(id),
    use_count     INT NOT NULL DEFAULT 0,
    like_count    INT NOT NULL DEFAULT 0,
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_prompts_user ON prompt_templates(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_prompts_visibility ON prompt_templates(visibility) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_prompts_tags ON prompt_templates USING GIN(tags);

CREATE TRIGGER update_prompt_templates_updated_at
    BEFORE UPDATE ON prompt_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
