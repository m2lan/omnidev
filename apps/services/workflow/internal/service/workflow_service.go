// Package service contains the business logic for the Workflow Service.
package service

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/services/workflow/internal/domain"
	"github.com/omnidev/services/workflow/internal/engine"
	"github.com/omnidev/services/workflow/internal/repository"
)

// WorkflowService handles workflow operations.
type WorkflowService struct {
	wfRepo     repository.WorkflowRepository
	runRepo    repository.RunRepository
	nodeRunRepo repository.NodeRunRepository
	engine     *engine.Engine
}

// NewWorkflowService creates a new workflow service.
func NewWorkflowService(
	wfRepo repository.WorkflowRepository,
	runRepo repository.RunRepository,
	nodeRunRepo repository.NodeRunRepository,
	engine *engine.Engine,
) *WorkflowService {
	return &WorkflowService{
		wfRepo:      wfRepo,
		runRepo:     runRepo,
		nodeRunRepo: nodeRunRepo,
		engine:      engine,
	}
}

// CreateWorkflowInput defines the input for creating a workflow.
type CreateWorkflowInput struct {
	Name        string                  `json:"name" validate:"required"`
	Description string                  `json:"description"`
	Definition  domain.WorkflowDefinition `json:"definition" validate:"required"`
	Tags        []string                `json:"tags"`
	TriggerType string                  `json:"trigger_type"`
}

// CreateWorkflow creates a new workflow.
func (s *WorkflowService) CreateWorkflow(ctx context.Context, userID uuid.UUID, input *CreateWorkflowInput) (*domain.Workflow, error) {
	triggerType := domain.TriggerManual
	if input.TriggerType != "" {
		triggerType = domain.TriggerType(input.TriggerType)
	}

	wf := &domain.Workflow{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        input.Name,
		Definition:  input.Definition,
		Version:     1,
		TriggerType: triggerType,
		IsActive:    true,
		Visibility:  "private",
		Tags:        input.Tags,
		Metadata:    map[string]interface{}{},
	}

	if input.Description != "" {
		wf.Description = &input.Description
	}
	if wf.Tags == nil {
		wf.Tags = []string{}
	}

	if err := s.wfRepo.Create(ctx, wf); err != nil {
		return nil, appErr.Wrap(err, "failed to create workflow")
	}

	return wf, nil
}

// GetWorkflow returns a workflow by ID.
func (s *WorkflowService) GetWorkflow(ctx context.Context, userID, wfID uuid.UUID) (*domain.Workflow, error) {
	wf, err := s.wfRepo.GetByID(ctx, wfID)
	if err != nil {
		return nil, appErr.NotFound("workflow")
	}
	if wf.UserID != userID {
		return nil, appErr.ErrForbidden
	}
	return wf, nil
}

// ListWorkflows returns a paginated list of workflows.
func (s *WorkflowService) ListWorkflows(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*domain.Workflow, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.wfRepo.List(ctx, userID, offset, pageSize)
}

// UpdateWorkflowInput defines fields for updating a workflow.
type UpdateWorkflowInput struct {
	Name        *string                  `json:"name"`
	Description *string                  `json:"description"`
	Definition  *domain.WorkflowDefinition `json:"definition"`
	IsActive    *bool                    `json:"is_active"`
	Tags        []string                 `json:"tags"`
}

// UpdateWorkflow updates a workflow.
func (s *WorkflowService) UpdateWorkflow(ctx context.Context, userID, wfID uuid.UUID, input *UpdateWorkflowInput) (*domain.Workflow, error) {
	wf, err := s.wfRepo.GetByID(ctx, wfID)
	if err != nil {
		return nil, appErr.NotFound("workflow")
	}
	if wf.UserID != userID {
		return nil, appErr.ErrForbidden
	}

	update := &repository.WorkflowUpdate{
		Name:        input.Name,
		Description: input.Description,
		Definition:  input.Definition,
		IsActive:    input.IsActive,
		Tags:        input.Tags,
	}

	if err := s.wfRepo.Update(ctx, wfID, update); err != nil {
		return nil, appErr.Wrap(err, "failed to update workflow")
	}

	return s.wfRepo.GetByID(ctx, wfID)
}

// DeleteWorkflow deletes a workflow.
func (s *WorkflowService) DeleteWorkflow(ctx context.Context, userID, wfID uuid.UUID) error {
	wf, err := s.wfRepo.GetByID(ctx, wfID)
	if err != nil {
		return appErr.NotFound("workflow")
	}
	if wf.UserID != userID {
		return appErr.ErrForbidden
	}
	return s.wfRepo.Delete(ctx, wfID)
}

// RunWorkflow starts a workflow execution.
func (s *WorkflowService) RunWorkflow(ctx context.Context, userID, wfID uuid.UUID, input map[string]interface{}) (*domain.Run, error) {
	wf, err := s.wfRepo.GetByID(ctx, wfID)
	if err != nil {
		return nil, appErr.NotFound("workflow")
	}
	if wf.UserID != userID {
		return nil, appErr.ErrForbidden
	}

	run := &domain.Run{
		ID:              uuid.New(),
		WorkflowID:      wfID,
		UserID:          userID,
		WorkflowVersion: wf.Version,
		Status:          domain.RunStatusPending,
		TriggerType:     wf.TriggerType,
		Input:           input,
		TotalNodes:      len(wf.Definition.Nodes),
		Metadata:        map[string]interface{}{},
	}

	if err := s.runRepo.Create(ctx, run); err != nil {
		return nil, appErr.Wrap(err, "failed to create run")
	}

	// Execute in background
	go s.executeRun(context.Background(), wf, run)

	return run, nil
}

// executeRun executes a workflow run.
func (s *WorkflowService) executeRun(ctx context.Context, wf *domain.Workflow, run *domain.Run) {
	logger.Log.Info("Starting workflow run",
		zap.String("run_id", run.ID.String()),
		zap.String("workflow_id", wf.ID.String()),
	)

	_ = s.runRepo.UpdateStatus(ctx, run.ID, domain.RunStatusRunning, nil, nil)

	nodeCh, err := s.engine.Run(ctx, wf.Definition, run.Input)
	if err != nil {
		errMsg := err.Error()
		_ = s.runRepo.UpdateStatus(ctx, run.ID, domain.RunStatusFailed, nil, &errMsg)
		return
	}

	completedNodes := 0
	var lastOutput map[string]interface{}
	allSuccess := true

	for nodeResult := range nodeCh {
		completedNodes++

		// Save node run
		nodeRun := domain.NodeRun{
			ID:       uuid.New(),
			RunID:    run.ID,
			NodeID:   nodeResult.NodeID,
			NodeType: nodeResult.NodeType,
			NodeName: nodeResult.NodeName,
			Status:   nodeResult.Status,
			Input:    nodeResult.Input,
			Output:   nodeResult.Output,
			LatencyMs: &nodeResult.LatencyMs,
		}
		if nodeResult.Error != "" {
			nodeRun.Error = &nodeResult.Error
		}
		_ = s.nodeRunRepo.Create(ctx, nodeRun)
		_ = s.runRepo.UpdateProgress(ctx, run.ID, completedNodes)

		lastOutput = nodeResult.Output
		if nodeResult.Status == domain.NodeRunStatusFailed {
			allSuccess = false
		}
	}

	// Update final status
	if allSuccess {
		_ = s.runRepo.UpdateStatus(ctx, run.ID, domain.RunStatusSuccess, lastOutput, nil)
	} else {
		errMsg := "workflow execution failed"
		_ = s.runRepo.UpdateStatus(ctx, run.ID, domain.RunStatusFailed, lastOutput, &errMsg)
	}

	logger.Log.Info("Workflow run completed",
		zap.String("run_id", run.ID.String()),
		zap.Int("nodes", completedNodes),
		zap.Bool("success", allSuccess),
	)
}

// GetRun returns a run with its node runs.
func (s *WorkflowService) GetRun(ctx context.Context, userID, runID uuid.UUID) (*domain.Run, []domain.NodeRun, error) {
	run, err := s.runRepo.GetByID(ctx, runID)
	if err != nil {
		return nil, nil, appErr.NotFound("run")
	}
	if run.UserID != userID {
		return nil, nil, appErr.ErrForbidden
	}

	nodeRuns, err := s.nodeRunRepo.ListByRun(ctx, runID)
	if err != nil {
		return nil, nil, appErr.Wrap(err, "failed to get node runs")
	}

	return run, nodeRuns, nil
}

// ListRuns returns runs for a workflow.
func (s *WorkflowService) ListRuns(ctx context.Context, userID, wfID uuid.UUID, page, pageSize int) ([]*domain.Run, int, error) {
	wf, err := s.wfRepo.GetByID(ctx, wfID)
	if err != nil {
		return nil, 0, appErr.NotFound("workflow")
	}
	if wf.UserID != userID {
		return nil, 0, appErr.ErrForbidden
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	return s.runRepo.ListByWorkflow(ctx, wfID, offset, pageSize)
}

// CancelRun cancels a running workflow.
func (s *WorkflowService) CancelRun(ctx context.Context, userID, runID uuid.UUID) error {
	run, err := s.runRepo.GetByID(ctx, runID)
	if err != nil {
		return appErr.NotFound("run")
	}
	if run.UserID != userID {
		return appErr.ErrForbidden
	}
	if run.Status != domain.RunStatusRunning && run.Status != domain.RunStatusPending {
		return appErr.Validation("run is not cancellable")
	}

	return s.runRepo.UpdateStatus(ctx, runID, domain.RunStatusCancelled, nil, nil)
}
