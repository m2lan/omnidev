// Package builtin provides built-in MCP server implementations.
package builtin

import (
	"context"
	"fmt"

	"github.com/omnidev/services/mcp/internal/protocol"
)

// Server defines the interface for a built-in MCP server.
type Server interface {
	// Name returns the server name.
	Name() string

	// Description returns the server description.
	Description() string

	// Tools returns the tools provided by this server.
	Tools() []protocol.Tool

	// HandleToolCall handles a tool call request.
	HandleToolCall(ctx context.Context, params *protocol.ToolCallParams) (*protocol.ToolCallResult, error)
}

// Registry manages built-in MCP servers.
type Registry struct {
	servers map[string]Server
}

// NewRegistry creates a new built-in server registry.
func NewRegistry() *Registry {
	return &Registry{
		servers: make(map[string]Server),
	}
}

// Register adds a built-in server.
func (r *Registry) Register(server Server) {
	r.servers[server.Name()] = server
}

// Get returns a built-in server by name.
func (r *Registry) Get(name string) (Server, error) {
	server, ok := r.servers[name]
	if !ok {
		return nil, fmt.Errorf("builtin server not found: %s", name)
	}
	return server, nil
}

// List returns all built-in servers.
func (r *Registry) List() []Server {
	servers := make([]Server, 0, len(r.servers))
	for _, s := range r.servers {
		servers = append(servers, s)
	}
	return servers
}

// GetAllTools returns all tools from all built-in servers.
func (r *Registry) GetAllTools() []protocol.Tool {
	tools := make([]protocol.Tool, 0)
	for _, s := range r.servers {
		tools = append(tools, s.Tools()...)
	}
	return tools
}

// FindTool finds which server provides the given tool.
func (r *Registry) FindTool(toolName string) (Server, error) {
	for _, s := range r.servers {
		for _, t := range s.Tools() {
			if t.Name == toolName {
				return s, nil
			}
		}
	}
	return nil, fmt.Errorf("tool not found: %s", toolName)
}
