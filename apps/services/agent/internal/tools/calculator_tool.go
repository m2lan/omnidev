package tools

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// CalculatorTool provides mathematical calculations.
type CalculatorTool struct{}

func NewCalculatorTool() *CalculatorTool { return &CalculatorTool{} }

func (t *CalculatorTool) Name() string { return "calculator" }

func (t *CalculatorTool) Description() string {
	return "Perform mathematical calculations. Supports basic arithmetic, powers, square roots, and common math functions."
}

func (t *CalculatorTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"expression": map[string]interface{}{
				"type":        "string",
				"description": "The mathematical expression to evaluate (e.g., '2 + 3 * 4', 'sqrt(16)', 'pow(2, 10)')",
			},
		},
		"required": []string{"expression"},
	}
}

func (t *CalculatorTool) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	expression, _ := input["expression"].(string)
	if expression == "" {
		return nil, fmt.Errorf("expression is required")
	}

	result, err := evaluateExpression(expression)
	if err != nil {
		return nil, fmt.Errorf("calculation error: %w", err)
	}

	return map[string]interface{}{
		"expression": expression,
		"result":     result,
	}, nil
}

func evaluateExpression(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)

	// Handle special functions
	if strings.HasPrefix(expr, "sqrt(") && strings.HasSuffix(expr, ")") {
		val, err := strconv.ParseFloat(expr[5:len(expr)-1], 64)
		if err != nil {
			return 0, err
		}
		return math.Sqrt(val), nil
	}

	if strings.HasPrefix(expr, "pow(") && strings.HasSuffix(expr, ")") {
		parts := strings.Split(expr[4:len(expr)-1], ",")
		if len(parts) != 2 {
			return 0, fmt.Errorf("pow requires two arguments")
		}
		base, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, err
		}
		exp, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return 0, err
		}
		return math.Pow(base, exp), nil
	}

	// Simple arithmetic: try to parse as a number first
	if val, err := strconv.ParseFloat(expr, 64); err == nil {
		return val, nil
	}

	// Handle basic operations
	for _, op := range []string{"+", "-", "*", "/"} {
		idx := strings.LastIndex(expr, op)
		if idx > 0 {
			left, err := evaluateExpression(expr[:idx])
			if err != nil {
				continue
			}
			right, err := evaluateExpression(expr[idx+1:])
			if err != nil {
				continue
			}
			switch op {
			case "+":
				return left + right, nil
			case "-":
				return left - right, nil
			case "*":
				return left * right, nil
			case "/":
				if right == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				return left / right, nil
			}
		}
	}

	return 0, fmt.Errorf("cannot evaluate expression: %s", expr)
}
