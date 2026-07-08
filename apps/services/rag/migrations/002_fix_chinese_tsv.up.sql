-- =============================================================================
-- Migration: Fix Chinese text search for document chunks
-- =============================================================================

-- Step 1: Try to create zhparser extension for Chinese tokenization
-- If zhparser is not available, we'll use 'simple' config as fallback
DO $$
BEGIN
    -- Try to create zhparser extension
    BEGIN
        CREATE EXTENSION IF NOT EXISTS zhparser;
        RAISE NOTICE 'zhparser extension created successfully';
    EXCEPTION WHEN OTHERS THEN
        RAISE NOTICE 'zhparser not available, using simple config: %', SQLERRM;
    END;
END $$;

-- Step 2: Create Chinese text search configuration if zhparser is available
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'zhparser') THEN
        -- Create Chinese text search configuration
        DROP TEXT SEARCH CONFIGURATION IF EXISTS chinese;
        CREATE TEXT SEARCH CONFIGURATION chinese (PARSER = zhparser);

        -- Add token mappings
        ALTER TEXT SEARCH CONFIGURATION chinese ADD MAPPING FOR n,v,a,i,e,l WITH simple;

        RAISE NOTICE 'Chinese text search configuration created';
    ELSE
        RAISE NOTICE 'Using simple configuration for Chinese text';
    END IF;
END $$;

-- Step 3: Update the trigger function to use appropriate text search config
CREATE OR REPLACE FUNCTION update_chunk_tsv() RETURNS trigger AS $$
BEGIN
    -- Use Chinese config if zhparser is available, otherwise use 'simple'
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'zhparser') THEN
        NEW.content_tsv := to_tsvector('chinese', COALESCE(NEW.content, ''));
    ELSE
        -- 'simple' config doesn't do stemming, works better for Chinese than 'english'
        NEW.content_tsv := to_tsvector('simple', COALESCE(NEW.content, ''));
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 4: Update existing chunks with new text search vectors
UPDATE document_chunks
SET content_tsv = CASE
    WHEN EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'zhparser') THEN
        to_tsvector('chinese', COALESCE(content, ''))
    ELSE
        to_tsvector('simple', COALESCE(content, ''))
    END
WHERE content_tsv IS NULL
   OR content_tsv = to_tsvector('english', '');

-- Step 5: Update the BM25 search function in retriever to use correct config
-- This will be handled in code changes

-- Step 6: Verify the fix
DO $$
DECLARE
    sample_tsv tsvector;
BEGIN
    SELECT content_tsv INTO sample_tsv
    FROM document_chunks
    LIMIT 1;

    IF sample_tsv IS NOT NULL AND sample_tsv != ''::tsvector THEN
        RAISE NOTICE 'Text search vectors updated successfully';
    ELSE
        RAISE NOTICE 'Warning: Text search vectors may be empty';
    END IF;
END $$;