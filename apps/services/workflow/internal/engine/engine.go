// Package engine provides workflow execution capabilities.
package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/services/workflow/internal/domain"
	"github.com/omnidev/services/workflow/internal/nodes"
)

// RunResult represents the result of a workflow run.
type RunResult struct {
	Status   domain.RunStatus         `json:"status"`
	Output   map[string]interface{}   `json:"output"`
	Error    string                   `json:"error,omitempty"`
	NodeRuns []NodeRunResult          `json:"node_runs"`
}

// NodeRunResult represents the result of a node execution.
type NodeRunResult struct {
	NodeID      string                 `json:"node_id"`
	NodeType    string                 `json:"node_type"`
	NodeName    string                 `json:"node_name"`
	Status      domain.NodeRunStatus   `json:"status"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output"`
	Error       string                 `json:"error,omitempty"`
	LatencyMs   int                    `json:"latency_ms"`
}

// Engine executes workflow definitions.
type Engine struct {
	nodeRegistry *nodes.Registry
}

// NewEngine creates a new workflow engine.
func NewEngine(nodeRegistry *nodes.Registry) *Engine {
	return &Engine{nodeRegistry: nodeRegistry}
}

// Run executes a workflow definition with the given input.
func (e *Engine) Run(ctx context.Context, def domain.WorkflowDefinition, input map[string]interface{}) (<-chan NodeRunResult, error) {
	ch := make(chan NodeRunResult, 100)

	go func() {
		defer close(ch)

		// Build adjacency list and find start nodes
		adjacency := make(map[string][]string)
		inDegree := make(map[string]int)
		nodeMap := make(map[string]domain.NodeDef)

		for _, node := range def.Nodes {
			nodeMap[node.ID] = node
			inDegree[node.ID] = 0
		}
		for _, edge := range def.Edges {
			adjacency[edge.Source] = append(adjacency[edge.Source], edge.Target)
			inDegree[edge.Target]++
		}

		// Topological sort execution
		// Start with nodes that have no incoming edges
		queue := make([]string, 0)
		for nodeID, degree := range inDegree {
			if degree == 0 {
				queue = append(queue, nodeID)
			}
		}

		// Store outputs from each node
		outputs := make(map[string]map[string]interface{})
		outputs["_input"] = input

		executed := make(map[string]bool)

		for len(queue) > 0 {
			nodeID := queue[0]
			queue = queue[1:]

			if executed[nodeID] {
				continue
			}

			nodeDef, ok := nodeMap[nodeID]
			if !ok {
				continue
			}

			// Execute node
			result := e.executeNode(ctx, nodeDef, input, outputs)
			ch <- result

			if result.Status == domain.NodeRunStatusFailed {
				// Workflow failed
				return
			}

			outputs[nodeID] = result.Output
			executed[nodeID] = true

			// Add next nodes to queue
			for _, nextID := range adjacency[nodeID] {
				// Check if all predecessors have been executed
				allDone := true
				for _, edge := range def.Edges {
					if edge.Target == nextID && !executed[edge.Source] {
						allDone = false
						break
					}
				}
				if allDone {
					queue = append(queue, nextID)
				}
			}
		}
	}()

	return ch, nil
}

// executeNode executes a single node.
func (e *Engine) executeNode(ctx context.Context, nodeDef domain.NodeDef, input map[string]interface{}, outputs map[string]map[string]interface{}) NodeRunResult {
	start := time.Now()

	nodeImpl, err := e.nodeRegistry.Get(nodeDef.Type)
	if err != nil {
		return NodeRunResult{
			NodeID:   nodeDef.ID,
			NodeType: nodeDef.Type,
			NodeName: nodeDef.Name,
			Status:   domain.NodeRunStatusFailed,
			Error:    fmt.Sprintf("node type not found: %s", nodeDef.Type),
			LatencyMs: int(time.Since(start).Milliseconds()),
		}
	}

	// Build node context
	nodeCtx := &nodes.NodeContext{
		NodeID:   nodeDef.ID,
		Input:    nodeDef.Config,
		Previous: outputs,
	}

	// Merge config with input references
	for key, val := range nodeDef.Config {
		if ref, ok := val.(string); ok && len(ref) > 2 && ref[0] == '{' && ref[len(ref)-1] == '}' {
			// Resolve reference
			resolved := resolveReference(ref, outputs)
			nodeCtx.Input[key] = resolved
		}
	}

	logger.Log.Debug("Executing node",
		zap.String("node_id", nodeDef.ID),
		zap.String("node_type", nodeDef.Type),
		zap.String("node_name", nodeDef.Name),
	)

	output, err := nodeImpl.Execute(ctx, nodeCtx)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		return NodeRunResult{
			NodeID:   nodeDef.ID,
			NodeType: nodeDef.Type,
			NodeName: nodeDef.Name,
			Status:   domain.NodeRunStatusFailed,
			Input:    nodeCtx.Input,
			Error:    err.Error(),
			LatencyMs: latency,
		}
	}

	status := domain.NodeRunStatusSuccess
	if output.Status == "failed" {
		status = domain.NodeRunStatusFailed
	}

	return NodeRunResult{
		NodeID:   nodeDef.ID,
		NodeType: nodeDef.Type,
		NodeName: nodeDef.Name,
		Status:   status,
		Input:    nodeCtx.Input,
		Output:   output.Data,
		Error:    output.Error,
		LatencyMs: latency,
	}
}

// resolveReference resolves a {{node_id.field}} reference.
func resolveReference(ref string, outputs map[string]map[string]interface{}) interface{} {
	// Simple reference resolution
	// Format: {{node_id.field}} or {{_input.field}}
	// For now, return the raw reference
	return ref
}
