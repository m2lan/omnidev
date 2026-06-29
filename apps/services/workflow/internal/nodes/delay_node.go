package nodes

import (
	"context"
	"time"
)

// DelayNode introduces a delay in the workflow.
type DelayNode struct{}

func NewDelayNode() *DelayNode { return &DelayNode{} }

func (n *DelayNode) Type() string        { return "delay" }
func (n *DelayNode) Name() string        { return "Delay" }
func (n *DelayNode) Description() string { return "Wait for a specified duration before continuing" }

func (n *DelayNode) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"duration": map[string]interface{}{
				"type":        "number",
				"description": "Duration to wait in seconds",
				"default":     1,
			},
		},
		"required": []string{"duration"},
	}
}

func (n *DelayNode) Execute(ctx context.Context, nodeCtx *NodeContext) (*NodeOutput, error) {
	duration := 1.0
	if d, ok := nodeCtx.Input["duration"].(float64); ok {
		duration = d
	}

	select {
	case <-ctx.Done():
		return &NodeOutput{Status: "failed", Error: "context cancelled"}, ctx.Err()
	case <-time.After(time.Duration(duration * float64(time.Second))):
		return &NodeOutput{
			Status: "success",
			Data: map[string]interface{}{
				"waited_seconds": duration,
			},
		}, nil
	}
}
