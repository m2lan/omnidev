// Package planner turns an agent task into the next executable step.
package planner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/omnidev/go-common/config"
)

const (
	defaultBaseURL = "https://api.openai.com/v1"
	defaultModel   = "gpt-4o-mini"
	maxPromptBytes = 12000
)

// Plan represents a task execution plan.
type Plan struct {
	Goal  string        `json:"goal"`
	Steps []PlannedStep `json:"steps"`
}

// PlannedStep represents a single step in a plan.
type PlannedStep struct {
	Number      int                    `json:"number"`
	Description string                 `json:"description"`
	Action      string                 `json:"action"` // think, tool_call, code_exec, response
	ToolName    string                 `json:"tool_name,omitempty"`
	ToolInput   map[string]interface{} `json:"tool_input,omitempty"`
}

// Planner asks an OpenAI-compatible chat model for the next step.
type Planner struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewPlanner creates a planner. An empty API key or base URL falls back to the
// OpenAI provider settings.
func NewPlanner(cfg config.AgentConfig, ai config.AIConfig) *Planner {
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = ai.OpenAI.APIKey
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = ai.OpenAI.BaseURL
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}

	return &Planner{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// NextStep asks the model what to do next. It returns action "response" when
// the task is complete.
func (p *Planner) NextStep(ctx context.Context, systemPrompt, task string, availableTools, completedSteps []string, lastOutput string) (*PlannedStep, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("agent model API key is not configured (set AGENT_API_KEY or OPENAI_API_KEY)")
	}

	completedJSON, _ := json.Marshal(completedSteps)
	if len(lastOutput) > maxPromptBytes {
		lastOutput = lastOutput[:maxPromptBytes] + "\n...[truncated]"
	}

	system := `You are an autonomous engineering agent operating inside a disposable Docker sandbox.
You can inspect and modify files under /workspace and execute code there.
Available tools: ` + strings.Join(availableTools, ", ") + `
Respond with a single JSON object and nothing else:
{"number": N, "description": "...", "action": "tool_call|think|code_exec|response", "tool_name": "...", "tool_input": {...}}
Use action "response" when the task is complete and put the final answer in "description".`
	if strings.TrimSpace(systemPrompt) != "" {
		system += "\n\nAgent instructions:\n" + systemPrompt
	}

	prompt := fmt.Sprintf("Task: %s\n\nCompleted steps: %s\n\nLast output:\n%s\n\nWhat is the next step?", task, completedJSON, lastOutput)
	messages := []map[string]string{
		{"role": "system", "content": system},
		{"role": "user", "content": prompt},
	}

	step, err := p.complete(ctx, messages)
	if err != nil {
		return nil, err
	}

	if step.Action == "" {
		step.Action = "think"
	}
	return step, nil
}

// CreatePlan generates an upfront plan. It is used by callers that want to show
// the intended approach before execution starts.
func (p *Planner) CreatePlan(ctx context.Context, systemPrompt, task string, availableTools []string) (*Plan, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("agent model API key is not configured (set AGENT_API_KEY or OPENAI_API_KEY)")
	}

	system := `You are a task planning assistant. Create a concise step-by-step plan.
Available tools: ` + strings.Join(availableTools, ", ") + `
Respond with JSON only:
{"goal": "...", "steps": [{"number": 1, "description": "...", "action": "tool_call|think|code_exec|response", "tool_name": "...", "tool_input": {}}]}`
	if strings.TrimSpace(systemPrompt) != "" {
		system += "\n\nAgent instructions:\n" + systemPrompt
	}

	content, err := p.chat(ctx, []map[string]string{
		{"role": "system", "content": system},
		{"role": "user", "content": task},
	})
	if err != nil {
		return nil, err
	}

	var plan Plan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return nil, fmt.Errorf("parse plan: %w", err)
	}
	return &plan, nil
}

func (p *Planner) complete(ctx context.Context, messages []map[string]string) (*PlannedStep, error) {
	content, err := p.chat(ctx, messages)
	if err != nil {
		return nil, err
	}

	var step PlannedStep
	if err := json.Unmarshal([]byte(content), &step); err != nil {
		return nil, fmt.Errorf("parse step: %w", err)
	}
	return &step, nil
}

func (p *Planner) chat(ctx context.Context, messages []map[string]string) (string, error) {
	body := map[string]interface{}{
		"model":       p.model,
		"messages":    messages,
		"temperature": 0.2,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("call model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("model error (status %d): %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode model response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("model returned no choices")
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content), nil
}
