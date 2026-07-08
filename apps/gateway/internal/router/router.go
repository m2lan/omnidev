// Package router sets up HTTP routes for the API Gateway.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/omnidev/go-common/auth"
	"github.com/omnidev/go-common/middleware"

	"github.com/omnidev/gateway/internal/handler"
)

// Setup configures all routes for the API Gateway.
func Setup(
	r *gin.Engine,
	jwtManager *auth.JWTManager,
	healthHandler *handler.HealthHandler,
	authHandler *handler.AuthHandler,
	chatHandler *handler.ChatHandler,
	userAIConfigHandler *handler.UserAIConfigHandler,
	uploadHandler *handler.UploadHandler,
	knowledgeHandler *handler.KnowledgeHandler,
) {
	// Health checks (no auth required)
	r.GET("/health", healthHandler.Health)
	r.GET("/ready", healthHandler.Ready)
	r.GET("/info", healthHandler.Info)

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Auth routes (no JWT required)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.GET("/oauth/:provider", authHandler.OAuthRedirect)
			auth.GET("/callback/:provider", authHandler.OAuthCallback)
		}

		// Protected routes (JWT required)
		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(jwtManager))
		{
			// User
			users := protected.Group("/users")
			{
				users.GET("/me", authHandler.GetProfile)
				users.PATCH("/me", authHandler.UpdateProfile)
				users.GET("/me/settings", authHandler.GetSettings)
				users.PATCH("/me/settings", authHandler.UpdateSettings)
				users.GET("/me/api-keys", authHandler.ListAPIKeys)
				users.POST("/me/api-keys", authHandler.CreateAPIKey)
				users.DELETE("/me/api-keys/:id", authHandler.RevokeAPIKey)
			}

			// Conversations
			conversations := protected.Group("/conversations")
			{
				conversations.GET("", chatHandler.ListConversations)
				conversations.POST("", chatHandler.CreateConversation)
				conversations.GET("/:id", chatHandler.GetConversation)
				conversations.PATCH("/:id", chatHandler.UpdateConversation)
				conversations.DELETE("/:id", chatHandler.DeleteConversation)
				conversations.GET("/:id/messages", chatHandler.ListMessages)
				conversations.POST("/:id/messages", chatHandler.SendMessage)
				conversations.POST("/:id/messages/stream", chatHandler.StreamMessage)
			}

			// Models
			protected.GET("/models", chatHandler.ListModels)

			// Image Generation
			protected.POST("/images/generate", chatHandler.GenerateImage)

			// Upload & Attachments
			protected.POST("/upload", uploadHandler.Upload)
			protected.GET("/attachments/:id", uploadHandler.GetAttachment)
			protected.DELETE("/attachments/:id", uploadHandler.DeleteAttachment)

			// User AI Configs
			aiConfigs := protected.Group("/user/ai-configs")
			{
				aiConfigs.POST("", userAIConfigHandler.Create)
				aiConfigs.GET("", userAIConfigHandler.List)
				aiConfigs.GET("/:id", userAIConfigHandler.Get)
				aiConfigs.PUT("/:id", userAIConfigHandler.Update)
				aiConfigs.DELETE("/:id", userAIConfigHandler.Delete)
				aiConfigs.PUT("/:id/default", userAIConfigHandler.SetDefault)
				aiConfigs.POST("/:id/test", userAIConfigHandler.TestConnection)
			}

			// Knowledge (Knowledge Engine SDK)
			knowledge := protected.Group("/knowledge")
			{
				knowledge.GET("", knowledgeHandler.ListKnowledgeBases)
				knowledge.POST("", knowledgeHandler.CreateKnowledgeBase)
				knowledge.GET("/:id", knowledgeHandler.GetKnowledgeBase)
				knowledge.PATCH("/:id", knowledgeHandler.UpdateKnowledgeBase)
				knowledge.DELETE("/:id", knowledgeHandler.DeleteKnowledgeBase)
				knowledge.GET("/:id/documents", knowledgeHandler.ListDocuments)
				knowledge.POST("/:id/documents", knowledgeHandler.UploadDocument)
				knowledge.DELETE("/:id/documents/:doc_id", knowledgeHandler.DeleteDocument)
				knowledge.POST("/:id/search", knowledgeHandler.Search)
			}
		}

		// Admin routes (admin role required)
		admin := v1.Group("/admin")
		admin.Use(middleware.JWTAuth(jwtManager))
		admin.Use(middleware.RequireRole("admin", "super_admin"))
		{
			// TODO(M3): Implement admin endpoints
		}
	}

	// NoRoute handles undefined routes with a structured 404 response.
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    404,
				"message": "route not found",
				"detail":  c.Request.Method + " " + c.Request.URL.Path + " is not a valid endpoint",
			},
		})
	})
}
