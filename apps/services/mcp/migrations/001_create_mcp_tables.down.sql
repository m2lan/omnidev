DROP TRIGGER IF EXISTS update_mcp_tools_updated_at ON mcp_tools;
DROP TRIGGER IF EXISTS update_mcp_servers_updated_at ON mcp_servers;
DROP TABLE IF EXISTS mcp_tools CASCADE;
DROP TABLE IF EXISTS mcp_servers CASCADE;
