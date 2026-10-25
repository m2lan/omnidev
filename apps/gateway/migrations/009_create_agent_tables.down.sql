DROP TRIGGER IF EXISTS update_agent_steps_updated_at ON agent_steps;
DROP TRIGGER IF EXISTS update_agent_runs_updated_at ON agent_runs;
DROP TRIGGER IF EXISTS update_agents_updated_at ON agents;
DROP TABLE IF EXISTS agent_steps CASCADE;
DROP TABLE IF EXISTS agent_runs CASCADE;
DROP TABLE IF EXISTS agents CASCADE;
