package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// TransformNode transforms data using JSONPath-like expressions.
type TransformNode struct{}

func NewTransformNode() *TransformNode { return &TransformNode{} }

func (n *TransformNode) Type() string        { return "transform" }
func (n *TransformNode) Name() string        { return "Transform" }
func (n *TransformNode) Description() string { return "Transform and map data between nodes" }

func (n *TransformNode) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"mapping": map[string]interface{}{
				"type":        "object",
				"description": "Key-value mapping where values are field references (e.g., 'node_id.field')",
			},
			"template": map[string]interface{}{
				"type":        "string",
				"description": "String template with {{field}} placeholders",
			},
		},
	}
}

func (n *TransformNode) Execute(ctx context.Context, nodeCtx *NodeContext) (*NodeOutput, error) {
	result := make(map[string]interface{})

	// Apply mapping
	if mapping, ok := nodeCtx.Input["mapping"].(map[string]interface{}); ok {
		for key, fieldRef := range mapping {
			if ref, ok := fieldRef.(string); ok {
				result[key] = resolveField(nodeCtx.Previous, ref)
			} else {
				result[key] = fieldRef
			}
		}
	}

	// Apply template
	if template, ok := nodeCtx.Input["template"].(string); ok && template != "" {
		output := template
		for key, val := range result {
			output = strings.ReplaceAll(output, fmt.Sprintf("{{%s}}", key), fmt.Sprintf("%v", val))
		}
		result["output"] = output
	}

	// If no mapping or template, pass through previous output
	if len(result) == 0 {
		for _, prevOutput := range nodeCtx.Previous {
			for k, v := range prevOutput {
				result[k] = v
			}
			break
		}
	}

	return &NodeOutput{
		Status: "success",
		Data:   result,
	}, nil
}

// toJSON converts an interface to JSON string.
func toJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}
