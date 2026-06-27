// Package middleware provides HTTP middleware for Gin.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omnidev/go-common/auth"
	"github.com/omnidev/go-common/errors"
)

const (
	// ContextKeyUserID is the context key for the authenticated user ID.
	ContextKeyUserID = "user_id"
	// ContextKeyEmail is the context key for the authenticated user email.
	ContextKeyEmail = "email"
	// ContextKeyRole is the context key for the authenticated user role.
	ContextKeyRole = "role"
	// ContextKeyOrgID is the context key for the organization ID.
	ContextKeyOrgID = "org_id"
	// ContextKeyClaims is the context key for the full JWT claims.
	ContextKeyClaims = "claims"
)

// JWTAuth returns a middleware that validates JWT tokens.
func JWTAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c)
		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    401,
					"message": "missing authorization token",
				},
			})
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    401,
					"message": "invalid or expired token",
				},
			})
			c.Abort()
			return
		}

		if claims.Type != auth.TokenTypeAccess {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    401,
					"message": "invalid token type",
				},
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyEmail, claims.Email)
		c.Set(ContextKeyRole, claims.Role)
		c.Set(ContextKeyOrgID, claims.OrgID)
		c.Set(ContextKeyClaims, claims)

		c.Next()
	}
}

// OptionalAuth is like JWTAuth but doesn't abort if no token is present.
func OptionalAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c)
		if tokenStr == "" {
			c.Next()
			return
		}

		claims, err := jwtManager.ValidateToken(tokenStr)
		if err != nil {
			c.Next()
			return
		}

		if claims.Type == auth.TokenTypeAccess {
			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyEmail, claims.Email)
			c.Set(ContextKeyRole, claims.Role)
			c.Set(ContextKeyOrgID, claims.OrgID)
			c.Set(ContextKeyClaims, claims)
		}

		c.Next()
	}
}

// RequireRole returns a middleware that checks if the user has one of the required roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextKeyRole)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    403,
					"message": "access denied",
				},
			})
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    403,
					"message": "access denied",
				},
			})
			c.Abort()
			return
		}

		for _, r := range roles {
			if r == roleStr {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    403,
				"message": "insufficient permissions",
			},
		})
		c.Abort()
	}
}

// GetUserID extracts the user ID from the Gin context.
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get(ContextKeyUserID)
	if !exists {
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}

// MustGetUserID extracts the user ID from the context or panics.
func MustGetUserID(c *gin.Context) uuid.UUID {
	id, ok := GetUserID(c)
	if !ok {
		panic("user_id not found in context")
	}
	return id
}

// GetUserEmail extracts the user email from the Gin context.
func GetUserEmail(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextKeyEmail)
	if !exists {
		return "", false
	}
	email, ok := val.(string)
	return email, ok
}

// GetUserRole extracts the user role from the Gin context.
func GetUserRole(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextKeyRole)
	if !exists {
		return "", false
	}
	role, ok := val.(string)
	return role, ok
}

// GetOrgID extracts the organization ID from the Gin context.
func GetOrgID(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextKeyOrgID)
	if !exists {
		return "", false
	}
	orgID, ok := val.(string)
	return orgID, ok
}

func extractToken(c *gin.Context) string {
	// Check Authorization header
	bearer := c.GetHeader("Authorization")
	if bearer != "" {
		parts := strings.SplitN(bearer, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}

	// Check query parameter
	if token := c.Query("token"); token != "" {
		return token
	}

	// Check cookie
	if token, err := c.Cookie("access_token"); err == nil {
		return token
	}

	return ""
}

// ErrorHandler is a middleware that handles errors uniformly.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			var appErr *errors.AppError
			if e, ok := err.(*errors.AppError); ok {
				appErr = e
			} else {
				appErr = errors.Wrap(err, "internal server error")
			}

			c.JSON(appErr.Code, gin.H{
				"error": gin.H{
					"code":       appErr.Code,
					"message":    appErr.Message,
					"detail":     appErr.Detail,
					"request_id": c.GetString("X-Request-ID"),
				},
			})
		}
	}
}
