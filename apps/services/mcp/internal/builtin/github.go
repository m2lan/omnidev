package builtin

import (
	"context"
	"fmt"

	"github.com/omnidev/services/mcp/internal/protocol"
)

// GitHubServer provides GitHub operations via MCP.
type GitHubServer struct{}

func NewGitHubServer() *GitHubServer { return &GitHubServer{} }

func (s *GitHubServer) Name() string       { return "github" }
func (s *GitHubServer) Description() string { return "GitHub operations: repos, issues, PRs, files" }

func (s *GitHubServer) Tools() []protocol.Tool {
	return []protocol.Tool{
		{
			Name:        "gh_list_repos",
			Description: "List GitHub repositories",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]interface{}{"type": "string", "description": "Repository owner"},
				},
			},
		},
		{
			Name:        "gh_get_file",
			Description: "Get a file from a GitHub repository",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner":  map[string]interface{}{"type": "string"},
					"repo":   map[string]interface{}{"type": "string"},
					"path":   map[string]interface{}{"type": "string"},
					"branch": map[string]interface{}{"type": "string", "description": "Branch name (default: main)"},
				},
				"required": []string{"owner", "repo", "path"},
			},
		},
		{
			Name:        "gh_list_issues",
			Description: "List issues in a repository",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner":  map[string]interface{}{"type": "string"},
					"repo":   map[string]interface{}{"type": "string"},
					"state":  map[string]interface{}{"type": "string", "enum": []string{"open", "closed", "all"}},
				},
				"required": []string{"owner", "repo"},
			},
		},
		{
			Name:        "gh_create_issue",
			Description: "Create a new issue",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner":   map[string]interface{}{"type": "string"},
					"repo":    map[string]interface{}{"type": "string"},
					"title":   map[string]interface{}{"type": "string"},
					"body":    map[string]interface{}{"type": "string"},
					"labels":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				},
				"required": []string{"owner", "repo", "title"},
			},
		},
	}
}

func (s *GitHubServer) HandleToolCall(ctx context.Context, params *protocol.ToolCallParams) (*protocol.ToolCallResult, error) {
	switch params.Name {
	case "gh_list_repos":
		return toolSuccess("GitHub integration not configured. Set GITHUB_TOKEN to enable."), nil
	case "gh_get_file":
		return toolSuccess("GitHub integration not configured."), nil
	case "gh_list_issues":
		return toolSuccess("GitHub integration not configured."), nil
	case "gh_create_issue":
		return toolSuccess("GitHub integration not configured."), nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", params.Name)
	}
}
