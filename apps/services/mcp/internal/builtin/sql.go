package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/mcp/internal/protocol"
)

// SQLServer provides database query operations via MCP.
type SQLServer struct {
	pool *pgxpool.Pool
}

func NewSQLServer(pool *pgxpool.Pool) *SQLServer { return &SQLServer{pool: pool} }

func (s *SQLServer) Name() string       { return "sql" }
func (s *SQLServer) Description() string { return "SQL database operations: query, list tables, describe table" }

func (s *SQLServer) Tools() []protocol.Tool {
	return []protocol.Tool{
		{
			Name:        "sql_query",
			Description: "Execute a read-only SQL query",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query":  map[string]interface{}{"type": "string", "description": "SQL query to execute"},
					"params": map[string]interface{}{"type": "array", "description": "Query parameters"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "sql_list_tables",
			Description: "List all tables in the database",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "sql_describe_table",
			Description: "Describe the structure of a table",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"table": map[string]interface{}{"type": "string", "description": "Table name"},
				},
				"required": []string{"table"},
			},
		},
	}
}

func (s *SQLServer) HandleToolCall(ctx context.Context, params *protocol.ToolCallParams) (*protocol.ToolCallResult, error) {
	switch params.Name {
	case "sql_query":
		return s.executeQuery(ctx, params.Arguments)
	case "sql_list_tables":
		return s.listTables(ctx)
	case "sql_describe_table":
		return s.describeTable(ctx, params.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

func (s *SQLServer) executeQuery(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return toolError("query is required"), nil
	}

	// Safety: only allow SELECT queries
	upperQuery := strings.ToUpper(strings.TrimSpace(query))
	if !strings.HasPrefix(upperQuery, "SELECT") && !strings.HasPrefix(upperQuery, "WITH") {
		return toolError("only SELECT queries are allowed"), nil
	}

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return toolError(fmt.Sprintf("query failed: %v", err)), nil
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	columns := make([]string, len(fields))
	for i, f := range fields {
		columns[i] = string(f.Name)
	}

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return toolError(fmt.Sprintf("failed to read row: %v", err)), nil
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	jsonData, _ := json.MarshalIndent(map[string]interface{}{
		"columns": columns,
		"rows":    results,
		"count":   len(results),
	}, "", "  ")

	return toolSuccess(string(jsonData)), nil
}

func (s *SQLServer) listTables(ctx context.Context) (*protocol.ToolCallResult, error) {
	query := `
		SELECT table_name, table_type
		FROM information_schema.tables
		WHERE table_schema = 'public'
		ORDER BY table_name`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return toolError(fmt.Sprintf("failed to list tables: %v", err)), nil
	}
	defer rows.Close()

	var result strings.Builder
	for rows.Next() {
		var tableName, tableType string
		if err := rows.Scan(&tableName, &tableType); err != nil {
			continue
		}
		result.WriteString(fmt.Sprintf("%s (%s)\n", tableName, tableType))
	}

	return toolSuccess(result.String()), nil
}

func (s *SQLServer) describeTable(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
	table, _ := args["table"].(string)
	if table == "" {
		return toolError("table is required"), nil
	}

	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position`

	rows, err := s.pool.Query(ctx, query, table)
	if err != nil {
		return toolError(fmt.Sprintf("failed to describe table: %v", err)), nil
	}
	defer rows.Close()

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Table: %s\n\n", table))
	result.WriteString(fmt.Sprintf("%-30s %-20s %-10s %s\n", "Column", "Type", "Nullable", "Default"))
	result.WriteString(strings.Repeat("-", 80) + "\n")

	for rows.Next() {
		var colName, dataType, nullable string
		var defaultVal *string
		if err := rows.Scan(&colName, &dataType, &nullable, &defaultVal); err != nil {
			continue
		}
		defaultStr := ""
		if defaultVal != nil {
			defaultStr = *defaultVal
		}
		result.WriteString(fmt.Sprintf("%-30s %-20s %-10s %s\n", colName, dataType, nullable, defaultStr))
	}

	return toolSuccess(result.String()), nil
}
