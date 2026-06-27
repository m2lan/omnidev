// Package handler provides HTTP handlers for the User Service.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/services/user/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authSvc *service.AuthService
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// Register handles user registration.
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var input service.RegisterInput
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

	// Validate
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    400,
				"message": "validation error",
				"detail":  err.Error(),
			},
		})
		return
	}

	result, err := h.authSvc.Register(c.Request.Context(), &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": result,
	})
}

// Login handles user login.
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var input service.LoginInput
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

	result, err := h.authSvc.Login(c.Request.Context(), &input, c.ClientIP())
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// RefreshToken handles token refresh.
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var input service.RefreshTokenInput
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

	result, err := h.authSvc.RefreshToken(c.Request.Context(), &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// OAuthRedirect redirects to OAuth provider.
// GET /api/v1/auth/oauth/:provider
func (h *AuthHandler) OAuthRedirect(c *gin.Context) {
	provider := c.Param("provider")
	// TODO: implement OAuth redirect
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    501,
			"message": "OAuth not yet implemented",
			"detail":  provider,
		},
	})
}

// OAuthCallback handles OAuth callback.
// GET /api/v1/auth/callback/:provider
func (h *AuthHandler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	// TODO: implement OAuth callback
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    501,
			"message": "OAuth not yet implemented",
			"detail":  provider,
		},
	})
}

// CreateAPIKey handles API key creation.
// POST /api/v1/users/me/api-keys
func (h *AuthHandler) CreateAPIKey(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	var input service.CreateAPIKeyInput
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

	result, err := h.authSvc.CreateAPIKey(c.Request.Context(), userID, &input)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": result,
	})
}

// ListAPIKeys handles listing API keys.
// GET /api/v1/users/me/api-keys
func (h *AuthHandler) ListAPIKeys(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	keys, err := h.authSvc.ListAPIKeys(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": keys,
	})
}

// RevokeAPIKey handles API key revocation.
// DELETE /api/v1/users/me/api-keys/:id
func (h *AuthHandler) RevokeAPIKey(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid key ID"},
		})
		return
	}

	if err := h.authSvc.RevokeAPIKey(c.Request.Context(), userID, keyID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"message": "API key revoked"},
	})
}

// handleError handles errors uniformly.
func handleError(c *gin.Context, err error) {
	if appErr, ok := err.(*appErr.AppError); ok {
		c.JSON(appErr.Code, gin.H{
			"error": gin.H{
				"code":       appErr.Code,
				"message":    appErr.Message,
				"detail":     appErr.Detail,
				"request_id": c.GetString("X-Request-ID"),
			},
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"code":       500,
			"message":    "internal server error",
			"request_id": c.GetString("X-Request-ID"),
		},
	})
}
