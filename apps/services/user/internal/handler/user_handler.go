package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/services/user/internal/domain"
	"github.com/omnidev/services/user/internal/service"
)

// UserHandler handles user profile endpoints.
type UserHandler struct {
	userSvc *service.UserService
}

// NewUserHandler creates a new user handler.
func NewUserHandler(userSvc *service.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

// GetProfile returns the authenticated user's profile.
// GET /api/v1/users/me
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	user, err := h.userSvc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": user,
	})
}

// UpdateProfileInput defines the input for updating a profile.
type UpdateProfileInput struct {
	Nickname  *string `json:"nickname"`
	AvatarURL *string `json:"avatar_url"`
	Bio       *string `json:"bio"`
}

// UpdateProfile updates the authenticated user's profile.
// PATCH /api/v1/users/me
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	var input UpdateProfileInput
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

	update := &domain.UserUpdate{
		Nickname:  input.Nickname,
		AvatarURL: input.AvatarURL,
		Bio:       input.Bio,
	}

	user, err := h.userSvc.UpdateProfile(c.Request.Context(), userID, update)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": user,
	})
}
