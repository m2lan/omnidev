package nodes

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// CodeNode executes code snippets.
type CodeNode struct{}

func NewCodeNode() *CodeNode { return &CodeNode{} }

func (n *CodeNode) Type() string        { return "code" }
func (n *CodeNode) Name() string        { return "Code Execution" }
func (n *CodeNode) Description() string { return "Execute code in Python, JavaScript, Go, or Shell" }

func (n *CodeNode) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"language": map[string]interface{}{"type": "string", "enum": []string{"python", "javascript", "go", "shell"}, "default": "python"},
			"code":     map[string]interface{}{"type": "string", "description": "Code to execute"},
			"timeout":  map[string]interface{}{"type": "number", "description": "Timeout in seconds", "default": 30},
		},
		"required": []string{"language", "code"},
	}
}

func (n *CodeNode) Execute(ctx context.Context, nodeCtx *NodeContext) (*NodeOutput, error) {
	language, _ := nodeCtx.Input["language"].(string)
	code, _ := nodeCtx.Input["code"].(string)
	timeout := 30.0
	if t, ok := nodeCtx.Input["timeout"].(float64); ok {
		timeout = t
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	switch language {
	case "python":
		cmd = exec.CommandContext(ctx, "python3", "-c", code)
	case "javascript":
		cmd = exec.CommandContext(ctx, "node", "-e", code)
	case "go":
		cmd = exec.CommandContext(ctx, "go", "run", "/dev/stdin")
		cmd.Stdin = bytes.NewReader([]byte(code))
	case "shell":
		cmd = exec.CommandContext(ctx, "sh", "-c", code)
	default:
		return &NodeOutput{Status: "failed", Error: fmt.Sprintf("unsupported language: %s", language)}, nil
	}

	output, err := cmd.CombinedOutput()
	result := &NodeOutput{
		Status: "success",
		Data: map[string]interface{}{
			"output":   string(output),
			"language": language,
		},
	}
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		result.Data["exit_code"] = cmd.ProcessState.ExitCode()
	}
	return result, nil
}
