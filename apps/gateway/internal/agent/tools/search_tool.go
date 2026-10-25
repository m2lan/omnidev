package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SearchTool searches text files inside the run workspace.
type SearchTool struct {
	root string
}

// NewSearchTool creates a workspace-scoped search tool.
func NewSearchTool(root string) *SearchTool {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	return &SearchTool{root: abs}
}

func (t *SearchTool) Name() string { return "search" }

func (t *SearchTool) Description() string {
	return "Search for text patterns in files inside the run workspace. Returns matching lines with file paths and line numbers."
}

func (t *SearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query (substring, case-insensitive)",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Workspace-relative directory to search in (default: workspace root)",
			},
			"file_pattern": map[string]interface{}{
				"type":        "string",
				"description": "File extension filter (e.g., '.go', '.ts')",
			},
			"max_results": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of results to return (default: 50)",
			},
		},
		"required": []string{"query"},
	}
}

func (t *SearchTool) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	query, _ := input["query"].(string)
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is required")
	}
	searchPath, _ := input["path"].(string)
	filePattern, _ := input["file_pattern"].(string)
	maxResults := 50
	if mr, ok := input["max_results"].(float64); ok && mr > 0 {
		maxResults = int(mr)
	}

	full, err := t.resolve(searchPath)
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0)
	walkErr := filepath.Walk(full, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if filePattern != "" && !strings.HasSuffix(path, filePattern) {
			return nil
		}
		if len(results) >= maxResults {
			return filepath.SkipAll
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(t.root, path)
		for i, line := range strings.Split(string(data), "\n") {
			if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
				results = append(results, map[string]interface{}{
					"file":    rel,
					"line":    i + 1,
					"content": strings.TrimSpace(line),
				})
				if len(results) >= maxResults {
					return filepath.SkipAll
				}
			}
		}
		return nil
	})

	if walkErr != nil {
		return nil, fmt.Errorf("search failed: %w", walkErr)
	}

	return map[string]interface{}{
		"results": results,
		"count":   len(results),
	}, nil
}

func (t *SearchTool) resolve(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return t.root, nil
	}
	if strings.HasPrefix(raw, "/workspace") {
		raw = strings.TrimPrefix(raw, "/workspace")
	}
	raw = strings.TrimPrefix(raw, "/")
	if filepath.IsAbs(raw) {
		return "", fmt.Errorf("absolute paths outside /workspace are not allowed")
	}
	full := filepath.Join(t.root, raw)
	if !withinRoot(t.root, full) {
		return "", fmt.Errorf("path escapes the workspace: %s", raw)
	}
	return full, nil
}
