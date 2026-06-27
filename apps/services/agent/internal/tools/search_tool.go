package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SearchTool provides text search in files.
type SearchTool struct{}

func NewSearchTool() *SearchTool { return &SearchTool{} }

func (t *SearchTool) Name() string { return "search" }

func (t *SearchTool) Description() string {
	return "Search for text patterns in files within a directory. Returns matching lines with file paths and line numbers."
}

func (t *SearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query (supports substring matching)",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The directory to search in",
			},
			"file_pattern": map[string]interface{}{
				"type":        "string",
				"description": "File extension filter (e.g., '.go', '.ts')",
			},
			"max_results": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of results to return",
			},
		},
		"required": []string{"query", "path"},
	}
}

func (t *SearchTool) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	query, _ := input["query"].(string)
	searchPath, _ := input["path"].(string)
	filePattern, _ := input["file_pattern"].(string)
	maxResults := 50
	if mr, ok := input["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	results := make([]map[string]interface{}, 0)
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if filePattern != "" && !strings.HasSuffix(path, filePattern) {
			return nil
		}

		if len(results) >= maxResults {
			return filepath.SkipDir
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
				results = append(results, map[string]interface{}{
					"file":       path,
					"line":       i + 1,
					"content":    strings.TrimSpace(line),
				})
				if len(results) >= maxResults {
					return filepath.SkipDir
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return map[string]interface{}{
		"results": results,
		"count":   len(results),
	}, nil
}
