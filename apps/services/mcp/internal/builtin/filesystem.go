package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/omnidev/services/mcp/internal/protocol"
)

// FilesystemServer provides file system operations via MCP.
type FilesystemServer struct {
	allowedPaths []string
}

// NewFilesystemServer creates a new filesystem MCP server.
func NewFilesystemServer() *FilesystemServer {
	return &FilesystemServer{
		allowedPaths: []string{"/workspace", "/tmp"},
	}
}

func (s *FilesystemServer) Name() string        { return "filesystem" }
func (s *FilesystemServer) Description() string  { return "File system operations: read, write, list, delete files" }

func (s *FilesystemServer) Tools() []protocol.Tool {
	return []protocol.Tool{
		{
			Name:        "fs_read_file",
			Description: "Read the contents of a file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{"type": "string", "description": "File path to read"},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "fs_write_file",
			Description: "Write content to a file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path":    map[string]interface{}{"type": "string", "description": "File path to write"},
					"content": map[string]interface{}{"type": "string", "description": "Content to write"},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			Name:        "fs_list_directory",
			Description: "List files and directories",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{"type": "string", "description": "Directory path"},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "fs_delete_file",
			Description: "Delete a file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{"type": "string", "description": "File path to delete"},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (s *FilesystemServer) HandleToolCall(ctx context.Context, params *protocol.ToolCallParams) (*protocol.ToolCallResult, error) {
	switch params.Name {
	case "fs_read_file":
		return s.readFile(params.Arguments)
	case "fs_write_file":
		return s.writeFile(params.Arguments)
	case "fs_list_directory":
		return s.listDirectory(params.Arguments)
	case "fs_delete_file":
		return s.deleteFile(params.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

func (s *FilesystemServer) readFile(args map[string]interface{}) (*protocol.ToolCallResult, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return toolError("path is required"), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return toolError(fmt.Sprintf("failed to read file: %v", err)), nil
	}

	return toolSuccess(string(data)), nil
}

func (s *FilesystemServer) writeFile(args map[string]interface{}) (*protocol.ToolCallResult, error) {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)

	if path == "" {
		return toolError("path is required"), nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return toolError(fmt.Sprintf("failed to create directory: %v", err)), nil
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return toolError(fmt.Sprintf("failed to write file: %v", err)), nil
	}

	return toolSuccess(fmt.Sprintf("File written: %s (%d bytes)", path, len(content))), nil
}

func (s *FilesystemServer) listDirectory(args map[string]interface{}) (*protocol.ToolCallResult, error) {
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return toolError(fmt.Sprintf("failed to list directory: %v", err)), nil
	}

	var result strings.Builder
	for _, entry := range entries {
		info, _ := entry.Info()
		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("[DIR]  %s/\n", entry.Name()))
		} else {
			result.WriteString(fmt.Sprintf("[FILE] %s (%d bytes)\n", entry.Name(), info.Size()))
		}
	}

	return toolSuccess(result.String()), nil
}

func (s *FilesystemServer) deleteFile(args map[string]interface{}) (*protocol.ToolCallResult, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return toolError("path is required"), nil
	}

	if err := os.Remove(path); err != nil {
		return toolError(fmt.Sprintf("failed to delete file: %v", err)), nil
	}

	return toolSuccess(fmt.Sprintf("File deleted: %s", path)), nil
}

func toolSuccess(text string) *protocol.ToolCallResult {
	return &protocol.ToolCallResult{
		Content: []protocol.ToolContent{{Type: "text", Text: text}},
	}
}

func toolError(text string) *protocol.ToolCallResult {
	return &protocol.ToolCallResult{
		Content: []protocol.ToolContent{{Type: "text", Text: text}},
		IsError: true,
	}
}
