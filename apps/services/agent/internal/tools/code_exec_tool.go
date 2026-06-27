package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CodeExecTool provides code execution capabilities.
type CodeExecTool struct{}

func NewCodeExecTool() *CodeExecTool { return &CodeExecTool{} }

func (t *CodeExecTool) Name() string { return "code_exec" }

func (t *CodeExecTool) Description() string {
	return "Execute code snippets in various languages. Supports Python, JavaScript (Node.js), Go, and Shell scripts."
}

func (t *CodeExecTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"language": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"python", "javascript", "go", "shell"},
				"description": "The programming language",
			},
			"code": map[string]interface{}{
				"type":        "string",
				"description": "The code to execute",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "Timeout in seconds (default: 30)",
			},
		},
		"required": []string{"language", "code"},
	}
}

func (t *CodeExecTool) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	language, _ := input["language"].(string)
	code, _ := input["code"].(string)
	timeout := 30.0
	if t, ok := input["timeout"].(float64); ok {
		timeout = t
	}

	if code == "" {
		return nil, fmt.Errorf("code is required")
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
		// Write to temp file and run
		cmd = exec.CommandContext(ctx, "go", "run", "/dev/stdin")
		cmd.Stdin = strings.NewReader(code)
	case "shell":
		cmd = exec.CommandContext(ctx, "sh", "-c", code)
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	output, err := cmd.CombinedOutput()
	result := map[string]interface{}{
		"output":   string(output),
		"language": language,
		"success":  err == nil,
	}

	if err != nil {
		result["error"] = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		}
	}

	return result, nil
}
