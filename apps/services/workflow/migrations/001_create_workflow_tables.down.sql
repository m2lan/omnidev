DROP TABLE IF EXISTS workflow_node_runs CASCADE;
DROP TABLE IF EXISTS workflow_runs CASCADE;
DROP TRIGGER IF EXISTS update_workflows_updated_at ON workflows;
DROP TABLE IF EXISTS workflows CASCADE;
