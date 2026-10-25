package tools

import (
	"context"
	"fmt"
	"time"
)

// CodeRunner executes a snippet for one language in an isolated environment.
type CodeRunner interface {
	Execute(ctx context.Context, workspace, language, code string) (map[string]interface{}, error)
}

// CodeExecTool exposes sandboxed code execution to the planner.
type CodeExecTool struct {
	runner    CodeRunner
	workspace string
}

// NewCodeExecTool creates a code execution tool bound to a run workspace.
func NewCodeExecTool(runner CodeRunner, workspace string) *CodeExecTool {
	return &CodeExecTool{runner: runner, workspace: workspace}
}

func (t *CodeExecTool) Name() string { return "code_exec" }

func (t *CodeExecTool) Description() string {
	return "Execute Python, JavaScript, Go or shell code in a disposable Docker sandbox. The workspace is mounted at /workspace and the container has no network access."
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
				"description": "Optional timeout in seconds (capped by the server sandbox timeout)",
			},
		},
		"required": []string{"language", "code"},
	}
}

func (t *CodeExecTool) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	language, _ := input["language"].(string)
	code, _ := input["code"].(string)
	if code == "" {
		return nil, fmt.Errorf("code is required")
	}

	if seconds, ok := input["timeout"].(float64); ok && seconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
		defer cancel()
	}

	return t.runner.Execute(ctx, t.workspace, language, code)
}
