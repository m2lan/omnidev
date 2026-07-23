// Package service contains the business logic for the Agent Service.
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/services/agent/internal/domain"
	"github.com/omnidev/services/agent/internal/executor"
	"github.com/omnidev/services/agent/internal/planner"
	"github.com/omnidev/services/agent/internal/repository"
)

// AgentService handles agent operations.
type AgentService struct {
	agentRepo repository.AgentRepository
	runRepo   repository.RunRepository
	stepRepo  repository.StepRepository
	executor  *executor.Executor
	planner   *planner.Planner
}

// NewAgentService creates a new agent service.
func NewAgentService(
	agentRepo repository.AgentRepository,
	runRepo repository.RunRepository,
	stepRepo repository.StepRepository,
	executor *executor.Executor,
	planner *planner.Planner,
) *AgentService {
	return &AgentService{
		agentRepo: agentRepo,
		runRepo:   runRepo,
		stepRepo:  stepRepo,
		executor:  executor,
		planner:   planner,
	}
}

// CreateAgentInput defines the input for creating an agent.
type CreateAgentInput struct {
	Name         string                `json:"name" validate:"required"`
	Description  string                `json:"description"`
	SystemPrompt string                `json:"system_prompt" validate:"required"`
	ModelID      string                `json:"model_id"`
	Tools        []domain.ToolConfig   `json:"tools"`
	Config       domain.AgentConfig    `json:"config"`
	Visibility   string                `json:"visibility"`
}

// CreateAgent creates a new agent.
func (s *AgentService) CreateAgent(ctx context.Context, userID uuid.UUID, input *CreateAgentInput) (*domain.Agent, error) {
	agent := &domain.Agent{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         input.Name,
		SystemPrompt: input.SystemPrompt,
		Tools:        input.Tools,
		Config:       input.Config,
		Visibility:   input.Visibility,
		Metadata:     map[string]interface{}{},
	}

	if input.Description != "" {
		agent.Description = &input.Description
	}
	if agent.Visibility == "" {
		agent.Visibility = "private"
	}
	if agent.Tools == nil {
		agent.Tools = []domain.ToolConfig{}
	}
	if agent.Config.MaxSteps == 0 {
		agent.Config.MaxSteps = 20
	}

	if err := s.agentRepo.Create(ctx, agent); err != nil {
		return nil, appErr.Wrap(err, "failed to create agent")
	}

	return agent, nil
}

// GetAgent returns an agent by ID.
func (s *AgentService) GetAgent(ctx context.Context, userID, agentID uuid.UUID) (*domain.Agent, error) {
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, appErr.NotFound("agent")
	}
	if agent.UserID != userID {
		return nil, appErr.ErrForbidden
	}
	return agent, nil
}

// ListAgents returns a paginated list of agents.
func (s *AgentService) ListAgents(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*domain.Agent, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.agentRepo.List(ctx, userID, offset, pageSize)
}

// DeleteAgent deletes an agent.
func (s *AgentService) DeleteAgent(ctx context.Context, userID, agentID uuid.UUID) error {
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return appErr.NotFound("agent")
	}
	if agent.UserID != userID {
		return appErr.ErrForbidden
	}
	return s.agentRepo.Delete(ctx, agentID)
}

// RunAgent starts an agent run.
func (s *AgentService) RunAgent(ctx context.Context, userID, agentID uuid.UUID, task string) (*domain.Run, error) {
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, appErr.NotFound("agent")
	}
	if agent.UserID != userID {
		return nil, appErr.ErrForbidden
	}

	run := &domain.Run{
		ID:         uuid.New(),
		AgentID:    agentID,
		UserID:     userID,
		Task:       task,
		Status:     domain.RunStatusCreated,
		TotalSteps: agent.Config.MaxSteps,
		Metadata:   map[string]interface{}{},
	}

	if err := s.runRepo.Create(ctx, run); err != nil {
		return nil, appErr.Wrap(err, "failed to create run")
	}

	// Start execution in background
	go s.executeRun(context.Background(), agent, run)

	return run, nil
}

// executeRun runs the agent task.
func (s *AgentService) executeRun(ctx context.Context, agent *domain.Agent, run *domain.Run) {
	logger.Log.Info("Starting agent run",
		zap.String("run_id", run.ID.String()),
		zap.String("agent_id", agent.ID.String()),
		zap.String("task", run.Task),
	)

	// Update status to planning
	_ = s.runRepo.UpdateStatus(ctx, run.ID, domain.RunStatusPlanning, nil, nil)

	// Build tool list
	toolNames := make([]string, 0)
	for _, t := range agent.Tools {
		if t.Enabled {
			toolNames = append(toolNames, t.Name)
		}
	}

	// Execute
	opts := executor.RunOptions{
		AgentID:      agent.ID,
		Task:         run.Task,
		Tools:        toolNames,
		MaxSteps:     agent.Config.MaxSteps,
		SystemPrompt: agent.SystemPrompt,
	}

	_ = s.runRepo.UpdateStatus(ctx, run.ID, domain.RunStatusExecuting, nil, nil)

	stepCh, err := s.executor.Run(ctx, opts)
	if err != nil {
		errMsg := err.Error()
		_ = s.runRepo.UpdateStatus(ctx, run.ID, domain.RunStatusFailed, nil, &errMsg)
		return
	}

	completedSteps := 0
	var finalResult string

	for stepResult := range stepCh {
		completedSteps++

		// Save step
		step := domain.Step{
			ID:         uuid.New(),
			RunID:      run.ID,
			StepNumber: stepResult.StepNumber,
			Type:       stepResult.Type,
			Status:     stepResult.Status,
			LatencyMs:  &stepResult.LatencyMs,
		}
		if stepResult.Content != "" {
			step.Content = &stepResult.Content
		}
		if stepResult.ToolName != "" {
			step.ToolName = &stepResult.ToolName
		}
		if stepResult.ToolInput != nil {
			step.ToolInput = stepResult.ToolInput
		}
		if stepResult.ToolOutput != nil {
			step.ToolOutput = stepResult.ToolOutput
		}
		if stepResult.Error != "" {
			step.Error = &stepResult.Error
		}

		// Store A2UI messages in step metadata
		if stepResult.ContentType == "a2ui" && len(stepResult.A2UIMessages) > 0 {
			if step.Metadata == nil {
				step.Metadata = map[string]interface{}{}
			}
			step.Metadata["content_type"] = "a2ui"
			step.Metadata["a2ui_messages"] = stepResult.A2UIMessages
		}

		_ = s.stepRepo.Create(ctx, step)
		_ = s.runRepo.UpdateProgress(ctx, run.ID, completedSteps)

		if stepResult.Type == domain.StepTypeResponse {
			finalResult = stepResult.Content
		}
	}

	// Determine final status
	if finalResult != "" {
		_ = s.runRepo.UpdateStatus(ctx, run.ID, domain.RunStatusSuccess, &finalResult, nil)
	} else {
		errMsg := "no final response generated"
		_ = s.runRepo.UpdateStatus(ctx, run.ID, domain.RunStatusFailed, nil, &errMsg)
	}

	logger.Log.Info("Agent run completed",
		zap.String("run_id", run.ID.String()),
		zap.Int("steps", completedSteps),
	)
}

// GetRun returns a run by ID.
func (s *AgentService) GetRun(ctx context.Context, userID, runID uuid.UUID) (*domain.Run, []domain.Step, error) {
	run, err := s.runRepo.GetByID(ctx, runID)
	if err != nil {
		return nil, nil, appErr.NotFound("run")
	}
	if run.UserID != userID {
		return nil, nil, appErr.ErrForbidden
	}

	steps, err := s.stepRepo.ListByRun(ctx, runID)
	if err != nil {
		return nil, nil, appErr.Wrap(err, "failed to get steps")
	}

	return run, steps, nil
}

// ListRuns returns runs for an agent.
func (s *AgentService) ListRuns(ctx context.Context, userID, agentID uuid.UUID, page, pageSize int) ([]*domain.Run, int, error) {
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, 0, appErr.NotFound("agent")
	}
	if agent.UserID != userID {
		return nil, 0, appErr.ErrForbidden
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	return s.runRepo.ListByAgent(ctx, agentID, offset, pageSize)
}

// CancelRun cancels a running agent run.
func (s *AgentService) CancelRun(ctx context.Context, userID, runID uuid.UUID) error {
	run, err := s.runRepo.GetByID(ctx, runID)
	if err != nil {
		return appErr.NotFound("run")
	}
	if run.UserID != userID {
		return appErr.ErrForbidden
	}

	if run.Status != domain.RunStatusExecuting && run.Status != domain.RunStatusPlanning {
		return appErr.Validation("run is not in a cancellable state")
	}

	return s.runRepo.UpdateStatus(ctx, runID, domain.RunStatusCancelled, nil, nil)
}
