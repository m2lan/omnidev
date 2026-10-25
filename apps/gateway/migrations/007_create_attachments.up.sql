-- Attachments table for file uploads in chat
CREATE TABLE IF NOT EXISTS attachments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    conversation_id UUID REFERENCES conversations(id),
    message_id      UUID,
    filename        VARCHAR(255) NOT NULL,
    mime_type       VARCHAR(100) NOT NULL,
    file_size       BIGINT NOT NULL,
    storage_key     VARCHAR(1024) NOT NULL,
    storage_url     TEXT NOT NULL,
    thumbnail_key   VARCHAR(1024),
    width           INT,
    height          INT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_attachments_user ON attachments(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_attachments_conversation ON attachments(conversation_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_attachments_message ON attachments(message_id) WHERE deleted_at IS NULL;
