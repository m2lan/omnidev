package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileTool provides file system operations.
type FileTool struct {
	allowedPaths []string
}

// NewFileTool creates a new file tool.
func NewFileTool() *FileTool {
	return &FileTool{
		allowedPaths: []string{"/workspace", "/tmp"},
	}
}

func (t *FileTool) Name() string { return "file" }

func (t *FileTool) Description() string {
	return "Read, write, and list files in the workspace. Supports read, write, list, and delete operations."
}

func (t *FileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"read", "write", "list", "delete", "exists"},
				"description": "The file operation to perform",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The file path",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write (for write operation)",
			},
		},
		"required": []string{"operation", "path"},
	}
}

func (t *FileTool) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	operation, _ := input["operation"].(string)
	path, _ := input["path"].(string)

	if !t.isPathAllowed(path) {
		return nil, fmt.Errorf("path not allowed: %s", path)
	}

	switch operation {
	case "read":
		return t.read(path)
	case "write":
		content, _ := input["content"].(string)
		return t.write(path, content)
	case "list":
		return t.list(path)
	case "delete":
		return t.delete(path)
	case "exists":
		return t.exists(path)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

func (t *FileTool) isPathAllowed(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	for _, allowed := range t.allowedPaths {
		if strings.HasPrefix(absPath, allowed) {
			return true
		}
	}
	return false
}

func (t *FileTool) read(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return map[string]interface{}{
		"content": string(data),
		"size":    len(data),
	}, nil
}

func (t *FileTool) write(path, content string) (map[string]interface{}, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	return map[string]interface{}{
		"success": true,
		"size":    len(content),
	}, nil
}

func (t *FileTool) list(path string) (map[string]interface{}, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	files := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		info, _ := entry.Info()
		files = append(files, map[string]interface{}{
			"name":      entry.Name(),
			"is_dir":    entry.IsDir(),
			"size":      info.Size(),
			"mod_time":  info.ModTime().Unix(),
		})
	}

	return map[string]interface{}{
		"files": files,
		"count": len(files),
	}, nil
}

func (t *FileTool) delete(path string) (map[string]interface{}, error) {
	if err := os.Remove(path); err != nil {
		return nil, fmt.Errorf("failed to delete file: %w", err)
	}
	return map[string]interface{}{"success": true}, nil
}

func (t *FileTool) exists(path string) (map[string]interface{}, error) {
	_, err := os.Stat(path)
	return map[string]interface{}{
		"exists": err == nil,
	}, nil
}
