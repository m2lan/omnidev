package nodes

import (
	"context"
	"fmt"
	"strings"
)

// ConditionNode evaluates conditions and routes flow.
type ConditionNode struct{}

func NewConditionNode() *ConditionNode { return &ConditionNode{} }

func (n *ConditionNode) Type() string        { return "condition" }
func (n *ConditionNode) Name() string        { return "Condition" }
func (n *ConditionNode) Description() string { return "Evaluate conditions and route workflow flow" }

func (n *ConditionNode) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"field":    map[string]interface{}{"type": "string", "description": "Field to evaluate (dot notation)"},
			"operator": map[string]interface{}{"type": "string", "enum": []string{"equals", "not_equals", "contains", "gt", "lt", "exists", "empty"}},
			"value":    map[string]interface{}{"description": "Value to compare against"},
		},
		"required": []string{"field", "operator"},
	}
}

func (n *ConditionNode) Execute(ctx context.Context, nodeCtx *NodeContext) (*NodeOutput, error) {
	field, _ := nodeCtx.Input["field"].(string)
	operator, _ := nodeCtx.Input["operator"].(string)
	value := nodeCtx.Input["value"]

	// Get field value from previous outputs
	fieldValue := resolveField(nodeCtx.Previous, field)

	result := evaluateCondition(fieldValue, operator, value)

	return &NodeOutput{
		Status: "success",
		Data: map[string]interface{}{
			"result":     result,
			"field":      field,
			"operator":   operator,
			"field_value": fieldValue,
			"branch":     boolToBranch(result),
		},
	}, nil
}

func resolveField(previous map[string]map[string]interface{}, field string) interface{} {
	parts := strings.SplitN(field, ".", 2)
	if len(parts) == 0 {
		return nil
	}

	// Try to find in previous outputs
	for _, output := range previous {
		if val, ok := output[parts[0]]; ok {
			if len(parts) == 1 {
				return val
			}
			if m, ok := val.(map[string]interface{}); ok {
				return resolveField(map[string]map[string]interface{}{"_": m}, parts[1])
			}
		}
	}
	return nil
}

func evaluateCondition(fieldValue interface{}, operator string, compareValue interface{}) bool {
	switch operator {
	case "equals":
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", compareValue)
	case "not_equals":
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", compareValue)
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", fieldValue), fmt.Sprintf("%v", compareValue))
	case "gt":
		return toFloat(fieldValue) > toFloat(compareValue)
	case "lt":
		return toFloat(fieldValue) < toFloat(compareValue)
	case "exists":
		return fieldValue != nil
	case "empty":
		return fieldValue == nil || fmt.Sprintf("%v", fieldValue) == ""
	default:
		return false
	}
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

func boolToBranch(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
