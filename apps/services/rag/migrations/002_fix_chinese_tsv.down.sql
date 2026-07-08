-- =============================================================================
-- Migration: Revert Chinese text search fix
-- =============================================================================

-- Step 1: Restore original trigger function with 'english' config
CREATE OR REPLACE FUNCTION update_chunk_tsv() RETURNS trigger AS $$
BEGIN
    NEW.content_tsv := to_tsvector('english', NEW.content);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 2: Update existing chunks back to English text search
UPDATE document_chunks
SET content_tsv = to_tsvector('english', COALESCE(content, ''));

-- Step 3: Drop Chinese text search configuration if it exists
DROP TEXT SEARCH CONFIGURATION IF EXISTS chinese;

-- Step 4: Note: We don't drop zhparser extension as it might be used elsewhere
-- If you want to remove it: DROP EXTENSION IF EXISTS zhparser;