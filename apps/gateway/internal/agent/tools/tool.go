// Package tools provides built-in tools for agents.
package tools

import (
	"context"
	"fmt"
	"sort"
)

// Tool defines the interface for agent tools.
type Tool interface {
	// Name returns the tool name.
	Name() string

	// Description returns the tool description.
	Description() string

	// Parameters returns the JSON Schema for the tool parameters.
	Parameters() map[string]interface{}

	// Execute runs the tool with the given input.
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}

// Registry manages available tools.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// NewRunRegistry creates the built-in tool set for a single run workspace.
func NewRunRegistry(workspace string, runner CodeRunner) *Registry {
	r := NewRegistry()
	r.Register(NewFileTool(workspace))
	r.Register(NewSearchTool(workspace))
	r.Register(NewCalculatorTool())
	r.Register(NewCodeExecTool(runner, workspace))
	return r
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Filter returns a registry that keeps only the named tools. An empty list
// keeps every tool.
func (r *Registry) Filter(names []string) *Registry {
	if len(names) == 0 {
		return r
	}
	allowed := make(map[string]struct{}, len(names))
	for _, name := range names {
		allowed[name] = struct{}{}
	}
	filtered := NewRegistry()
	for name, tool := range r.tools {
		if _, ok := allowed[name]; ok {
			filtered.Register(tool)
		}
	}
	return filtered
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// List returns all registered tools in name order.
func (r *Registry) List() []Tool {
	list := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })
	return list
}

// Names returns the registered tool names in name order.
func (r *Registry) Names() []string {
	tools := r.List()
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t.Name())
	}
	return names
}

// ToToolDefinitions converts tools to OpenAI function calling format.
func (r *Registry) ToToolDefinitions() []map[string]interface{} {
	tools := r.List()
	defs := make([]map[string]interface{}, 0, len(tools))
	for _, t := range tools {
		defs = append(defs, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        t.Name(),
				"description": t.Description(),
				"parameters":  t.Parameters(),
			},
		})
	}
	return defs
}
