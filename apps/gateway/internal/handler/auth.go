// Package handler provides HTTP handlers for the API Gateway.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/omnidev/go-common/auth"
	"github.com/omnidev/go-common/cache"
	"github.com/omnidev/go-common/config"
	"github.com/omnidev/go-common/domain"
	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/gateway/internal/repository"
	"github.com/omnidev/gateway/internal/service"

	"go.uber.org/zap"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authSvc  *service.AuthService
	userRepo repository.UserRepository
	cache    *cache.Redis
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(
	userRepo repository.UserRepository,
	oauthRepo repository.OAuthRepository,
	apiKeyRepo repository.APIKeyRepository,
	jwtManager *auth.JWTManager,
	cache *cache.Redis,
	config *config.Config,
) *AuthHandler {
	authSvc := service.NewAuthService(userRepo, oauthRepo, apiKeyRepo, jwtManager, cache, config)
	return &AuthHandler{
		authSvc:  authSvc,
		userRepo: userRepo,
		cache:    cache,
	}
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
	// Register custom password validation
	validate.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		password := fl.Field().String()
		return len(password) >= 8 && len(password) <= 100
	})
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

	logger.Log.Error("Unhandled error", zap.Error(err))

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"code":       500,
			"message":    "internal server error",
			"request_id": c.GetString("X-Request-ID"),
		},
	})
}

// GetProfile handles getting user profile.
// GET /api/v1/users/me
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	// Try cache first
	cacheKey := fmt.Sprintf("user:profile:%s", userID.String())
	if h.cache != nil {
		var cachedUser domain.User
		if err := h.cache.Get(c.Request.Context(), cacheKey, &cachedUser); err == nil {
			c.JSON(http.StatusOK, gin.H{"data": cachedUser})
			return
		}
	}

	// Get from database
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	// Cache for 5 minutes
	if h.cache != nil {
		data, _ := json.Marshal(user)
		_ = h.cache.Set(c.Request.Context(), cacheKey, data, 5*60)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": user,
	})
}

// UpdateProfile handles updating user profile.
// PATCH /api/v1/users/me
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	var input domain.UserUpdate
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

	// Update user
	if err := h.userRepo.Update(c.Request.Context(), userID, &input); err != nil {
		handleError(c, err)
		return
	}

	// Get updated user
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	// Invalidate cache
	if h.cache != nil {
		cacheKey := fmt.Sprintf("user:profile:%s", userID.String())
		_ = h.cache.Delete(c.Request.Context(), cacheKey)
	}

	logger.Log.Info("Profile updated",
		zap.String("user_id", userID.String()),
	)

	c.JSON(http.StatusOK, gin.H{
		"data": user,
	})
}

// ValidateAPIKey validates an API key from Authorization header.
func (h *AuthHandler) ValidateAPIKey(c *gin.Context) {
	key := c.GetHeader("Authorization")
	if key == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "missing API key"},
		})
		c.Abort()
		return
	}

	// Remove "Bearer " prefix if present
	if len(key) > 7 && key[:7] == "Bearer " {
		key = key[7:]
	}

	user, apiKey, err := h.authSvc.ValidateAPIKey(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "invalid API key"},
		})
		c.Abort()
		return
	}

	// Set user info in context
	c.Set("user_id", user.ID)
	c.Set("user_email", user.Email)
	c.Set("user_role", string(user.Role))
	c.Set("api_key_id", apiKey.ID)
	c.Set("api_key_scopes", apiKey.Scopes)

	c.Next()
}

// UserResponse represents a user response without sensitive data.
type UserResponse struct {
	ID        uuid.UUID              `json:"id"`
	Email     string                 `json:"email"`
	Nickname  string                 `json:"nickname"`
	AvatarURL string                 `json:"avatar_url,omitempty"`
	Role      domain.UserRole        `json:"role"`
	Status    domain.UserStatus      `json:"status"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// GetSettings returns the current user's settings.
// GET /api/v1/users/me/settings
func (h *AuthHandler) GetSettings(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	settings := user.Settings
	if settings == nil {
		settings = map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{"data": settings})
}

// UpdateSettingsInput defines the input for updating user settings.
type UpdateSettingsInput struct {
	RAGMode       *string   `json:"rag_mode,omitempty"`
	DefaultKBIDs  []string  `json:"default_kb_ids,omitempty"`
}

// UpdateSettings partially updates the current user's settings.
// PATCH /api/v1/users/me/settings
func (h *AuthHandler) UpdateSettings(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": 401, "message": "unauthorized"},
		})
		return
	}

	var input UpdateSettingsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": 400, "message": "invalid request body", "detail": err.Error()},
		})
		return
	}

	// Get current user to merge settings
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	// Merge into existing settings
	settings := user.Settings
	if settings == nil {
		settings = map[string]interface{}{}
	}

	if input.RAGMode != nil {
		// Validate RAG mode value
		switch *input.RAGMode {
		case "off", "all", "specified":
			settings["rag_mode"] = *input.RAGMode
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": 400, "message": "rag_mode must be one of: off, all, specified"},
			})
			return
		}
	}
	if input.DefaultKBIDs != nil {
		settings["default_kb_ids"] = input.DefaultKBIDs
	}

	// Persist
	update := &domain.UserUpdate{Settings: settings}
	if err := h.userRepo.Update(c.Request.Context(), userID, update); err != nil {
		handleError(c, err)
		return
	}

	// Invalidate cache
	if h.cache != nil {
		cacheKey := fmt.Sprintf("user:profile:%s", userID.String())
		_ = h.cache.Delete(c.Request.Context(), cacheKey)
	}

	c.JSON(http.StatusOK, gin.H{"data": settings})
}
