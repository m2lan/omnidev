// Package handler provides HTTP handlers for the Agent Service.
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/services/agent/internal/service"
)

// AgentHandler handles agent endpoints.
type AgentHandler struct {
	agentSvc *service.AgentService
}

// NewAgentHandler creates a new agent handler.
func NewAgentHandler(agentSvc *service.AgentService) *AgentHandler {
	return &AgentHandler{agentSvc: agentSvc}
}

// ListAgents returns a paginated list of agents.
// GET /api/v1/agents
func (h *AgentHandler) ListAgents(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	agents, total, err := h.agentSvc.ListAgents(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": agents,
		"meta": gin.H{"total_count": total, "page": page, "page_size": pageSize},
	})
}

// CreateAgent creates a new agent.
// POST /api/v1/agents
func (h *AgentHandler) CreateAgent(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	var input service.CreateAgentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	agent, err := h.agentSvc.CreateAgent(c.Request.Context(), userID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": agent})
}

// GetAgent returns an agent by ID.
// GET /api/v1/agents/:id
func (h *AgentHandler) GetAgent(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid agent ID")
		return
	}

	agent, err := h.agentSvc.GetAgent(c.Request.Context(), userID, agentID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": agent})
}

// UpdateAgent updates an agent.
// PATCH /api/v1/agents/:id
func (h *AgentHandler) UpdateAgent(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": 501, "message": "not implemented"}})
}

// DeleteAgent deletes an agent.
// DELETE /api/v1/agents/:id
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid agent ID")
		return
	}

	if err := h.agentSvc.DeleteAgent(c.Request.Context(), userID, agentID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "agent deleted"}})
}

// RunAgent starts an agent run.
// POST /api/v1/agents/:id/run
func (h *AgentHandler) RunAgent(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid agent ID")
		return
	}

	var input struct {
		Task string `json:"task" validate:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	if input.Task == "" {
		badRequest(c, "task is required")
		return
	}

	run, err := h.agentSvc.RunAgent(c.Request.Context(), userID, agentID, input.Task)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": run})
}

// ListRuns returns runs for an agent.
// GET /api/v1/agents/:id/runs
func (h *AgentHandler) ListRuns(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid agent ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	runs, total, err := h.agentSvc.ListRuns(c.Request.Context(), userID, agentID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": runs,
		"meta": gin.H{"total_count": total, "page": page, "page_size": pageSize},
	})
}

// GetRun returns a run with its steps.
// GET /api/v1/agents/runs/:run_id
func (h *AgentHandler) GetRun(c *gin.Context) {
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

	run, steps, err := h.agentSvc.GetRun(c.Request.Context(), userID, runID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"run":   run,
			"steps": steps,
		},
	})
}

// CancelRun cancels a running agent run.
// POST /api/v1/agents/runs/:run_id/cancel
func (h *AgentHandler) CancelRun(c *gin.Context) {
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

	if err := h.agentSvc.CancelRun(c.Request.Context(), userID, runID); err != nil {
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
		c.JSON(e.Code, gin.H{"error": gin.H{"code": e.Code, "message": e.Message, "detail": e.Detail, "request_id": c.GetString("X-Request-ID")}})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": 500, "message": "internal server error", "request_id": c.GetString("X-Request-ID")}})
}
