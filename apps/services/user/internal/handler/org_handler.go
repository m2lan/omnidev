package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/services/user/internal/domain"
	"github.com/omnidev/services/user/internal/service"
)

// OrganizationHandler handles organization endpoints.
type OrganizationHandler struct {
	orgSvc *service.OrganizationService
}

// NewOrganizationHandler creates a new organization handler.
func NewOrganizationHandler(orgSvc *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{orgSvc: orgSvc}
}

// CreateOrganization creates a new organization.
// POST /api/v1/organizations
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	var input service.CreateOrgInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    400,
				"message": "invalid request body",
				"detail":  err.Error(),
			},
		})
		return
	}

	org, err := h.orgSvc.CreateOrganization(c.Request.Context(), userID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": org,
	})
}

// GetOrganization returns an organization by ID.
// GET /api/v1/organizations/:id
func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid organization ID"},
		})
		return
	}

	org, err := h.orgSvc.GetOrganization(c.Request.Context(), orgID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": org,
	})
}

// UpdateOrganizationInput defines the input for updating an organization.
type UpdateOrganizationInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	AvatarURL   *string `json:"avatar_url"`
}

// UpdateOrganization updates an organization.
// PATCH /api/v1/organizations/:id
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid organization ID"},
		})
		return
	}

	// Check membership
	isMember, _ := h.orgSvc.IsMember(c.Request.Context(), orgID, userID)
	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{"code": 403, "message": "not a member of this organization"},
		})
		return
	}

	var input UpdateOrganizationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    400,
				"message": "invalid request body",
				"detail":  err.Error(),
			},
		})
		return
	}

	update := &domain.OrganizationUpdate{
		Name:        input.Name,
		Description: input.Description,
		AvatarURL:   input.AvatarURL,
	}

	org, err := h.orgSvc.UpdateOrganization(c.Request.Context(), orgID, update)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": org,
	})
}

// ListOrganizations returns organizations the user belongs to.
// GET /api/v1/organizations
func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	orgs, total, err := h.orgSvc.ListOrganizations(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": orgs,
		"meta": gin.H{
			"total_count": total,
			"page":        page,
			"page_size":   pageSize,
		},
	})
}

// InviteMember invites a user to an organization.
// POST /api/v1/organizations/:id/members/invite
func (h *OrganizationHandler) InviteMember(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid organization ID"},
		})
		return
	}

	var input service.InviteMemberInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    400,
				"message": "invalid request body",
				"detail":  err.Error(),
			},
		})
		return
	}

	member, err := h.orgSvc.InviteMember(c.Request.Context(), orgID, userID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": member,
	})
}

// ListMembers returns members of an organization.
// GET /api/v1/organizations/:id/members
func (h *OrganizationHandler) ListMembers(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid organization ID"},
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	members, total, err := h.orgSvc.ListMembers(c.Request.Context(), orgID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": members,
		"meta": gin.H{
			"total_count": total,
			"page":        page,
			"page_size":   pageSize,
		},
	})
}

// UpdateMemberRoleInput defines the input for updating a member's role.
type UpdateMemberRoleInput struct {
	Role string `json:"role" validate:"required,oneof=admin member viewer"`
}

// UpdateMemberRole updates a member's role.
// PATCH /api/v1/organizations/:id/members/:user_id/role
func (h *OrganizationHandler) UpdateMemberRole(c *gin.Context) {
	actorID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid organization ID"},
		})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid user ID"},
		})
		return
	}

	var input UpdateMemberRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    400,
				"message": "invalid request body",
				"detail":  err.Error(),
			},
		})
		return
	}

	if err := h.orgSvc.UpdateMemberRole(c.Request.Context(), orgID, actorID, targetUserID, domain.OrgMemberRole(input.Role)); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"message": "role updated"},
	})
}

// RemoveMember removes a member from an organization.
// DELETE /api/v1/organizations/:id/members/:user_id
func (h *OrganizationHandler) RemoveMember(c *gin.Context) {
	actorID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid organization ID"},
		})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid user ID"},
		})
		return
	}

	if err := h.orgSvc.RemoveMember(c.Request.Context(), orgID, actorID, targetUserID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"message": "member removed"},
	})
}
