// Package handler provides HTTP handlers for the Workflow Service.
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/services/workflow/internal/service"
)

// WorkflowHandler handles workflow endpoints.
type WorkflowHandler struct {
	wfSvc *service.WorkflowService
}

// NewWorkflowHandler creates a new workflow handler.
func NewWorkflowHandler(wfSvc *service.WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{wfSvc: wfSvc}
}

// ListWorkflows returns a paginated list of workflows.
// GET /api/v1/workflows
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	wfs, total, err := h.wfSvc.ListWorkflows(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": wfs,
		"meta": gin.H{"total_count": total, "page": page, "page_size": pageSize},
	})
}

// CreateWorkflow creates a new workflow.
// POST /api/v1/workflows
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	var input service.CreateWorkflowInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	wf, err := h.wfSvc.CreateWorkflow(c.Request.Context(), userID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": wf})
}

// GetWorkflow returns a workflow by ID.
// GET /api/v1/workflows/:id
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	wfID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid workflow ID")
		return
	}

	wf, err := h.wfSvc.GetWorkflow(c.Request.Context(), userID, wfID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": wf})
}

// UpdateWorkflow updates a workflow.
// PATCH /api/v1/workflows/:id
func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	wfID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid workflow ID")
		return
	}

	var input service.UpdateWorkflowInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	wf, err := h.wfSvc.UpdateWorkflow(c.Request.Context(), userID, wfID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": wf})
}

// DeleteWorkflow deletes a workflow.
// DELETE /api/v1/workflows/:id
func (h *WorkflowHandler) DeleteWorkflow(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	wfID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid workflow ID")
		return
	}

	if err := h.wfSvc.DeleteWorkflow(c.Request.Context(), userID, wfID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "workflow deleted"}})
}

// RunWorkflow starts a workflow execution.
// POST /api/v1/workflows/:id/run
func (h *WorkflowHandler) RunWorkflow(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	wfID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid workflow ID")
		return
	}

	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		input = map[string]interface{}{}
	}

	run, err := h.wfSvc.RunWorkflow(c.Request.Context(), userID, wfID, input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": run})
}

// ListRuns returns runs for a workflow.
// GET /api/v1/workflows/:id/runs
func (h *WorkflowHandler) ListRuns(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	wfID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid workflow ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	runs, total, err := h.wfSvc.ListRuns(c.Request.Context(), userID, wfID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": runs,
		"meta": gin.H{"total_count": total, "page": page, "page_size": pageSize},
	})
}

// GetRun returns a run with its node runs.
// GET /api/v1/workflows/runs/:run_id
func (h *WorkflowHandler) GetRun(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		badRequest(c, "invalid run ID")
		return
	}

	run, nodeRuns, err := h.wfSvc.GetRun(c.Request.Context(), userID, runID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"run":       run,
			"node_runs": nodeRuns,
		},
	})
}

// CancelRun cancels a running workflow.
// POST /api/v1/workflows/runs/:run_id/cancel
func (h *WorkflowHandler) CancelRun(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		badRequest(c, "invalid run ID")
		return
	}

	if err := h.wfSvc.CancelRun(c.Request.Context(), userID, runID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "run cancelled"}})
}

// --- Helpers ---

func unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": 401, "message": "unauthorized"}})
}

func badRequest(c *gin.Context, detail string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": 400, "message": "bad request", "detail": detail}})
}

func handleError(c *gin.Context, err error) {
	if e, ok := err.(*appErr.AppError); ok {
		c.JSON(e.Code, gin.H{"error": gin.H{"code": e.Code, "message": e.Message, "detail": e.Detail}})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": 500, "message": "internal server error"}})
}
