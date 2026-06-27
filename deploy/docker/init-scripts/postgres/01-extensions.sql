-- =============================================================================
-- OmniDev AI Platform — PostgreSQL Extensions
-- Runs on first initialization only
-- =============================================================================

-- UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Vector search (pgvector)
CREATE EXTENSION IF NOT EXISTS "vector";

-- Fuzzy text matching
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Query performance tracking
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Create application user with limited privileges
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'omnidev_app') THEN
        CREATE ROLE omnidev_app LOGIN PASSWORD 'omnidev_app';
    END IF;
END
$$;

-- Grant permissions
GRANT CONNECT ON DATABASE omnidev TO omnidev_app;
GRANT USAGE ON SCHEMA public TO omnidev_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO omnidev_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO omnidev_app;
