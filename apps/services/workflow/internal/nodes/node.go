// Package nodes provides workflow node implementations.
package nodes

import (
	"context"
	"fmt"
)

// NodeContext provides context for node execution.
type NodeContext struct {
	NodeID   string
	Input    map[string]interface{}
	Previous map[string]map[string]interface{} // Previous node outputs by node ID
}

// NodeOutput represents the output of a node execution.
type NodeOutput struct {
	Data   map[string]interface{} `json:"data"`
	Status string                 `json:"status"` // success, failed, skipped
	Error  string                 `json:"error,omitempty"`
}

// Node defines the interface for workflow nodes.
type Node interface {
	// Type returns the node type identifier.
	Type() string

	// Name returns the human-readable name.
	Name() string

	// Description returns the node description.
	Description() string

	// Schema returns the JSON Schema for node configuration.
	Schema() map[string]interface{}

	// Execute runs the node with the given context.
	Execute(ctx context.Context, nodeCtx *NodeContext) (*NodeOutput, error)
}

// Registry manages available node types.
type Registry struct {
	nodes map[string]Node
}

// NewRegistry creates a new node registry.
func NewRegistry() *Registry {
	return &Registry{
		nodes: make(map[string]Node),
	}
}

// Register adds a node type to the registry.
func (r *Registry) Register(node Node) {
	r.nodes[node.Type()] = node
}

// Get returns a node type by name.
func (r *Registry) Get(nodeType string) (Node, error) {
	node, ok := r.nodes[nodeType]
	if !ok {
		return nil, fmt.Errorf("node type not found: %s", nodeType)
	}
	return node, nil
}

// List returns all registered node types.
func (r *Registry) List() []Node {
	nodes := make([]Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// ToNodeDefinitions returns node definitions for the frontend palette.
func (r *Registry) ToNodeDefinitions() []map[string]interface{} {
	defs := make([]map[string]interface{}, 0, len(r.nodes))
	for _, n := range r.nodes {
		defs = append(defs, map[string]interface{}{
			"type":        n.Type(),
			"name":        n.Name(),
			"description": n.Description(),
			"schema":      n.Schema(),
		})
	}
	return defs
}
