package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileTool provides workspace-scoped file operations. Paths are resolved
// relative to the run workspace; absolute paths are only accepted when they
// point inside /workspace.
type FileTool struct {
	root string
}

// NewFileTool creates a file tool rooted at the run workspace.
func NewFileTool(root string) *FileTool {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	return &FileTool{root: abs}
}

func (t *FileTool) Name() string { return "file" }

func (t *FileTool) Description() string {
	return "Read, write, list, delete and check files inside the run workspace (/workspace)."
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
				"description": "Workspace-relative path, e.g. src/main.go or /workspace/src/main.go",
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

	full, err := t.resolve(path)
	if err != nil {
		return nil, err
	}

	switch operation {
	case "read":
		return t.read(full)
	case "write":
		content, _ := input["content"].(string)
		return t.write(full, content)
	case "list":
		return t.list(full)
	case "delete":
		return t.delete(full)
	case "exists":
		return t.exists(full)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

func (t *FileTool) resolve(path string) (string, error) {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "/workspace") {
		path = strings.TrimPrefix(path, "/workspace")
	}
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return t.root, nil
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute paths outside /workspace are not allowed")
	}

	full := filepath.Join(t.root, path)
	if !withinRoot(t.root, full) {
		return "", fmt.Errorf("path escapes the workspace: %s", path)
	}

	// Reject symlinks that resolve outside the workspace.
	if resolved, err := filepath.EvalSymlinks(full); err == nil {
		if !withinRoot(t.root, resolved) {
			return "", fmt.Errorf("path escapes the workspace: %s", path)
		}
		return resolved, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	if parent, err := filepath.EvalSymlinks(filepath.Dir(full)); err == nil {
		if !withinRoot(t.root, parent) {
			return "", fmt.Errorf("path escapes the workspace: %s", path)
		}
	}
	return full, nil
}

func withinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
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
		info, err := entry.Info()
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(t.root, filepath.Join(path, entry.Name()))
		files = append(files, map[string]interface{}{
			"name":     entry.Name(),
			"path":     rel,
			"is_dir":   entry.IsDir(),
			"size":     info.Size(),
			"mod_time": info.ModTime().Unix(),
		})
	}

	return map[string]interface{}{
		"files": files,
		"count": len(files),
	}, nil
}

func (t *FileTool) delete(path string) (map[string]interface{}, error) {
	if path == t.root {
		return nil, fmt.Errorf("refusing to delete the workspace root")
	}
	if err := os.RemoveAll(path); err != nil {
		return nil, fmt.Errorf("failed to delete path: %w", err)
	}
	return map[string]interface{}{"success": true}, nil
}

func (t *FileTool) exists(path string) (map[string]interface{}, error) {
	_, err := os.Stat(path)
	return map[string]interface{}{
		"exists": err == nil,
	}, nil
}
