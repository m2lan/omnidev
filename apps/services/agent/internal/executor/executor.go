// Package executor provides agent execution capabilities.
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/services/agent/internal/domain"
	"github.com/omnidev/services/agent/internal/planner"
	"github.com/omnidev/services/agent/internal/tools"
)

// ExecuteResult represents the result of an agent execution.
type ExecuteResult struct {
	Steps   []StepResult `json:"steps"`
	Result  string       `json:"result"`
	Success bool         `json:"success"`
	Error   string       `json:"error,omitempty"`
}

// StepResult represents the result of a single step.
type StepResult struct {
	StepNumber   int                    `json:"step_number"`
	Type         domain.StepType        `json:"type"`
	Content      string                 `json:"content"`
	ContentType  string                 `json:"content_type,omitempty"`   // "text", "markdown", "a2ui"
	A2UIMessages []interface{}          `json:"a2ui_messages,omitempty"` // A2UI JSON messages
	ToolName     string                 `json:"tool_name,omitempty"`
	ToolInput    map[string]interface{} `json:"tool_input,omitempty"`
	ToolOutput   map[string]interface{} `json:"tool_output,omitempty"`
	Status       domain.StepStatus      `json:"status"`
	Error        string                 `json:"error,omitempty"`
	LatencyMs    int                    `json:"latency_ms"`
}

// Executor runs agent tasks step by step.
type Executor struct {
	toolRegistry *tools.Registry
	sandboxMgr   *SandboxManager
	planner      *planner.Planner
}

// NewExecutor creates a new executor.
func NewExecutor(toolRegistry *tools.Registry, sandboxMgr *SandboxManager, planner *planner.Planner) *Executor {
	return &Executor{
		toolRegistry: toolRegistry,
		sandboxMgr:   sandboxMgr,
		planner:      planner,
	}
}

// RunOptions contains options for running an agent.
type RunOptions struct {
	AgentID      uuid.UUID
	Task         string
	Tools        []string
	MaxSteps     int
	SystemPrompt string
}

// Run executes an agent task and returns results via the channel.
func (e *Executor) Run(ctx context.Context, opts RunOptions) (<-chan StepResult, error) {
	if opts.MaxSteps <= 0 {
		opts.MaxSteps = 20
	}

	availableTools := make([]string, 0)
	for _, t := range e.toolRegistry.List() {
		availableTools = append(availableTools, t.Name())
	}

	ch := make(chan StepResult, 100)

	go func() {
		defer close(ch)

		completedSteps := make([]string, 0)
		lastOutput := ""
		stepNumber := 0

		for stepNumber < opts.MaxSteps {
			stepNumber++

			nextStep, err := e.planner.NextStep(ctx, opts.Task, completedSteps, lastOutput)
			if err != nil {
				ch <- StepResult{
					StepNumber: stepNumber,
					Type:       domain.StepTypeThink,
					Status:     domain.StepStatusFailed,
					Error:      fmt.Sprintf("planning failed: %v", err),
				}
				return
			}

			if nextStep.Action == "response" {
				ch <- StepResult{
					StepNumber: stepNumber,
					Type:       domain.StepTypeResponse,
					Content:    nextStep.Description,
					Status:     domain.StepStatusSuccess,
				}
				return
			}

			result := e.executeStep(ctx, stepNumber, nextStep)
			ch <- result

			completedSteps = append(completedSteps, nextStep.Description)

			if result.Status == domain.StepStatusFailed {
				lastOutput = fmt.Sprintf("Error: %s", result.Error)
			} else if result.ToolOutput != nil {
				outputJSON, _ := json.Marshal(result.ToolOutput)
				lastOutput = string(outputJSON)
			} else {
				lastOutput = result.Content
			}
		}

		ch <- StepResult{
			StepNumber: stepNumber + 1,
			Type:       domain.StepTypeResponse,
			Content:    "Maximum steps reached. Task incomplete.",
			Status:     domain.StepStatusFailed,
			Error:      "max steps exceeded",
		}
	}()

	return ch, nil
}

func (e *Executor) executeStep(ctx context.Context, stepNumber int, step *planner.PlannedStep) StepResult {
	start := time.Now()

	switch step.Action {
	case "think":
		return StepResult{
			StepNumber: stepNumber,
			Type:       domain.StepTypeThink,
			Content:    step.Description,
			Status:     domain.StepStatusSuccess,
			LatencyMs:  int(time.Since(start).Milliseconds()),
		}
	case "tool_call":
		return e.executeToolCall(ctx, stepNumber, step)
	case "code_exec":
		return e.executeCode(ctx, stepNumber, step)
	default:
		return StepResult{
			StepNumber: stepNumber,
			Type:       domain.StepTypeThink,
			Content:    step.Description,
			Status:     domain.StepStatusSuccess,
			LatencyMs:  int(time.Since(start).Milliseconds()),
		}
	}
}

func (e *Executor) executeToolCall(ctx context.Context, stepNumber int, step *planner.PlannedStep) StepResult {
	start := time.Now()

	tool, err := e.toolRegistry.Get(step.ToolName)
	if err != nil {
		return StepResult{
			StepNumber: stepNumber,
			Type:       domain.StepTypeToolCall,
			ToolName:   step.ToolName,
			ToolInput:  step.ToolInput,
			Status:     domain.StepStatusFailed,
			Error:      fmt.Sprintf("tool not found: %s", step.ToolName),
			LatencyMs:  int(time.Since(start).Milliseconds()),
		}
	}

	output, err := tool.Execute(ctx, step.ToolInput)
	if err != nil {
		return StepResult{
			StepNumber: stepNumber,
			Type:       domain.StepTypeToolCall,
			ToolName:   step.ToolName,
			ToolInput:  step.ToolInput,
			Status:     domain.StepStatusFailed,
			Error:      err.Error(),
			LatencyMs:  int(time.Since(start).Milliseconds()),
		}
	}

	logger.Log.Debug("Tool executed",
		zap.String("tool", step.ToolName),
		zap.Int("latency_ms", int(time.Since(start).Milliseconds())),
	)

	result := StepResult{
		StepNumber: stepNumber,
		Type:       domain.StepTypeToolCall,
		ToolName:   step.ToolName,
		ToolInput:  step.ToolInput,
		ToolOutput: output,
		Status:     domain.StepStatusSuccess,
		LatencyMs:  int(time.Since(start).Milliseconds()),
	}

	// Check if tool output contains A2UI messages
	if a2uiMsgs, ok := output["a2ui_messages"]; ok {
		if msgs, ok := a2uiMsgs.([]interface{}); ok && len(msgs) > 0 {
			result.ContentType = "a2ui"
			result.A2UIMessages = msgs
			// Remove a2ui_messages from ToolOutput to avoid duplication
			delete(output, "a2ui_messages")
		}
	}

	return result
}

func (e *Executor) executeCode(ctx context.Context, stepNumber int, step *planner.PlannedStep) StepResult {
	start := time.Now()

	language, _ := step.ToolInput["language"].(string)
	code, _ := step.ToolInput["code"].(string)

	result, err := e.sandboxMgr.Execute(ctx, language, code)
	if err != nil {
		return StepResult{
			StepNumber: stepNumber,
			Type:       domain.StepTypeCodeExec,
			Status:     domain.StepStatusFailed,
			Error:      err.Error(),
			LatencyMs:  int(time.Since(start).Milliseconds()),
		}
	}

	return StepResult{
		StepNumber: stepNumber,
		Type:       domain.StepTypeCodeExec,
		ToolOutput: result,
		Status:     domain.StepStatusSuccess,
		LatencyMs:  int(time.Since(start).Milliseconds()),
	}
}
