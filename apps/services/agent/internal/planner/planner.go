// Package planner provides task planning capabilities for agents.
package planner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/omnidev/go-common/config"
)

// Plan represents a task execution plan.
type Plan struct {
	Goal   string         `json:"goal"`
	Steps  []PlannedStep  `json:"steps"`
}

// PlannedStep represents a single step in a plan.
type PlannedStep struct {
	Number      int                    `json:"number"`
	Description string                 `json:"description"`
	Action      string                 `json:"action"` // think, tool_call, code_exec, response
	ToolName    string                 `json:"tool_name,omitempty"`
	ToolInput   map[string]interface{} `json:"tool_input,omitempty"`
}

// Planner generates execution plans for agent tasks.
type Planner struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewPlanner creates a new planner.
func NewPlanner(cfg config.AIConfig) *Planner {
	baseURL := cfg.OpenAI.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &Planner{
		apiKey:  cfg.OpenAI.APIKey,
		baseURL: baseURL,
		model:   "gpt-4o-mini",
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// CreatePlan generates an execution plan for the given task.
func (p *Planner) CreatePlan(ctx context.Context, task string, availableTools []string) (*Plan, error) {
	systemPrompt := `You are a task planning assistant. Given a task and a list of available tools,
create a step-by-step execution plan. Each step should be actionable and specific.

Available tools: ` + fmt.Sprintf("%v", availableTools) + `

Respond with a JSON plan in this format:
{
  "goal": "The overall goal",
  "steps": [
    {
      "number": 1,
      "description": "What this step does",
      "action": "tool_call|think|code_exec|response",
      "tool_name": "tool name if action is tool_call",
      "tool_input": {"key": "value"}
    }
  ]
}`

	body := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": task},
		},
		"temperature": 0.3,
		"response_format": map[string]string{"type": "json_object"},
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call AI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AI error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no plan generated")
	}

	var plan Plan
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	return &plan, nil
}

// NextStep determines the next step based on current state.
func (p *Planner) NextStep(ctx context.Context, task string, completedSteps []string, lastOutput string) (*PlannedStep, error) {
	completedJSON, _ := json.Marshal(completedSteps)

	prompt := fmt.Sprintf(`Task: %s

Completed steps: %s

Last output: %s

What should be the next step? If the task is complete, respond with action "response" and the final answer.
Respond with a single JSON step:
{"number": N, "description": "...", "action": "tool_call|think|code_exec|response", "tool_name": "...", "tool_input": {...}}`,
		task, string(completedJSON), lastOutput)

	body := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
		"response_format": map[string]string{"type": "json_object"},
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no step generated")
	}

	var step PlannedStep
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &step); err != nil {
		return nil, err
	}

	return &step, nil
}
