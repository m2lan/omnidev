// Package handler provides HTTP handlers for the Deploy Service.
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/services/deploy/internal/service"
)

// DeployHandler handles deployment endpoints.
type DeployHandler struct {
	deploySvc *service.DeployService
}

// NewDeployHandler creates a new deploy handler.
func NewDeployHandler(deploySvc *service.DeployService) *DeployHandler {
	return &DeployHandler{deploySvc: deploySvc}
}

// ListDeployments returns a paginated list of deployments.
// GET /api/v1/deployments
func (h *DeployHandler) ListDeployments(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	deploys, total, err := h.deploySvc.ListDeployments(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": deploys,
		"meta": gin.H{"total_count": total, "page": page, "page_size": pageSize},
	})
}

// CreateDeployment creates a new deployment.
// POST /api/v1/deployments
func (h *DeployHandler) CreateDeployment(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	var input service.CreateDeploymentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	deploy, err := h.deploySvc.CreateDeployment(c.Request.Context(), userID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": deploy})
}

// GetDeployment returns a deployment by ID.
// GET /api/v1/deployments/:id
func (h *DeployHandler) GetDeployment(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	deployID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid deployment ID")
		return
	}

	deploy, err := h.deploySvc.GetDeployment(c.Request.Context(), userID, deployID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": deploy})
}

// RollbackDeployment rolls back a deployment.
// POST /api/v1/deployments/:id/rollback
func (h *DeployHandler) RollbackDeployment(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	deployID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid deployment ID")
		return
	}

	deploy, err := h.deploySvc.RollbackDeployment(c.Request.Context(), userID, deployID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": deploy})
}

// GetDeploymentLogs returns logs for a deployment.
// GET /api/v1/deployments/:id/logs
func (h *DeployHandler) GetDeploymentLogs(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		unauthorized(c)
		return
	}

	deployID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid deployment ID")
		return
	}

	lines, _ := strconv.Atoi(c.DefaultQuery("lines", "100"))
	if lines <= 0 {
		lines = 100
	}

	logs, err := h.deploySvc.GetDeploymentLogs(c.Request.Context(), userID, deployID, lines)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"logs": logs}})
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
