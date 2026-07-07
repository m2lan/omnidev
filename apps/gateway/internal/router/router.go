// Package router sets up HTTP routes for the API Gateway.
package router

import (
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

			// Prompts
			prompts := protected.Group("/prompts")
			{
				prompts.GET("", handler.NotImplemented("list prompts"))
				prompts.POST("", handler.NotImplemented("create prompt"))
				prompts.GET("/:id", handler.NotImplemented("get prompt"))
				prompts.PATCH("/:id", handler.NotImplemented("update prompt"))
				prompts.DELETE("/:id", handler.NotImplemented("delete prompt"))
			}

			// Agents
			agents := protected.Group("/agents")
			{
				agents.GET("", handler.NotImplemented("list agents"))
				agents.POST("", handler.NotImplemented("create agent"))
				agents.GET("/:id", handler.NotImplemented("get agent"))
				agents.PATCH("/:id", handler.NotImplemented("update agent"))
				agents.DELETE("/:id", handler.NotImplemented("delete agent"))
				agents.POST("/:id/run", handler.NotImplemented("run agent"))
				agents.GET("/:id/runs", handler.NotImplemented("list agent runs"))
				agents.GET("/runs/:run_id", handler.NotImplemented("get agent run"))
				agents.POST("/runs/:run_id/cancel", handler.NotImplemented("cancel agent run"))
			}

			// Knowledge Bases
			knowledge := protected.Group("/knowledge")
			{
				knowledge.GET("", handler.NotImplemented("list knowledge bases"))
				knowledge.POST("", handler.NotImplemented("create knowledge base"))
				knowledge.GET("/:id", handler.NotImplemented("get knowledge base"))
				knowledge.PATCH("/:id", handler.NotImplemented("update knowledge base"))
				knowledge.DELETE("/:id", handler.NotImplemented("delete knowledge base"))
				knowledge.GET("/:id/documents", handler.NotImplemented("list documents"))
				knowledge.POST("/:id/documents", handler.NotImplemented("upload document"))
				knowledge.DELETE("/:id/documents/:doc_id", handler.NotImplemented("delete document"))
				knowledge.POST("/:id/search", handler.NotImplemented("search knowledge base"))
			}

			// Projects
			projects := protected.Group("/projects")
			{
				projects.GET("", handler.NotImplemented("list projects"))
				projects.POST("", handler.NotImplemented("create project"))
				projects.GET("/:id", handler.NotImplemented("get project"))
				projects.PATCH("/:id", handler.NotImplemented("update project"))
				projects.DELETE("/:id", handler.NotImplemented("delete project"))
				projects.GET("/:id/files", handler.NotImplemented("list project files"))
				projects.GET("/:id/files/*path", handler.NotImplemented("get file"))
				projects.PUT("/:id/files/*path", handler.NotImplemented("write file"))
				projects.DELETE("/:id/files/*path", handler.NotImplemented("delete file"))
			}

			// Workflows
			workflows := protected.Group("/workflows")
			{
				workflows.GET("", handler.NotImplemented("list workflows"))
				workflows.POST("", handler.NotImplemented("create workflow"))
				workflows.GET("/:id", handler.NotImplemented("get workflow"))
				workflows.PATCH("/:id", handler.NotImplemented("update workflow"))
				workflows.DELETE("/:id", handler.NotImplemented("delete workflow"))
				workflows.POST("/:id/run", handler.NotImplemented("run workflow"))
				workflows.GET("/:id/runs", handler.NotImplemented("list workflow runs"))
				workflows.GET("/runs/:run_id", handler.NotImplemented("get workflow run"))
			}

			// MCP Servers
			mcp := protected.Group("/mcp")
			{
				mcp.GET("/servers", handler.NotImplemented("list mcp servers"))
				mcp.POST("/servers", handler.NotImplemented("add mcp server"))
				mcp.GET("/servers/:id", handler.NotImplemented("get mcp server"))
				mcp.PATCH("/servers/:id", handler.NotImplemented("update mcp server"))
				mcp.DELETE("/servers/:id", handler.NotImplemented("delete mcp server"))
				mcp.GET("/servers/:id/tools", handler.NotImplemented("list mcp tools"))
				mcp.POST("/tools/:id/call", handler.NotImplemented("call mcp tool"))
			}

			// Deployments
			deploy := protected.Group("/deployments")
			{
				deploy.GET("", handler.NotImplemented("list deployments"))
				deploy.POST("", handler.NotImplemented("create deployment"))
				deploy.GET("/:id", handler.NotImplemented("get deployment"))
				deploy.POST("/:id/rollback", handler.NotImplemented("rollback deployment"))
				deploy.GET("/:id/logs", handler.NotImplemented("get deployment logs"))
			}

			// Billing
			billing := protected.Group("/billing")
			{
				billing.GET("/usage", handler.NotImplemented("get usage"))
				billing.GET("/invoices", handler.NotImplemented("list invoices"))
				billing.GET("/invoices/:id", handler.NotImplemented("get invoice"))
				billing.POST("/subscribe", handler.NotImplemented("subscribe"))
				billing.POST("/payment-method", handler.NotImplemented("add payment method"))
			}

			// Organizations
			orgs := protected.Group("/organizations")
			{
				orgs.GET("", handler.NotImplemented("list organizations"))
				orgs.POST("", handler.NotImplemented("create organization"))
				orgs.GET("/:id", handler.NotImplemented("get organization"))
				orgs.PATCH("/:id", handler.NotImplemented("update organization"))
				orgs.GET("/:id/members", handler.NotImplemented("list members"))
				orgs.POST("/:id/members/invite", handler.NotImplemented("invite member"))
				orgs.DELETE("/:id/members/:user_id", handler.NotImplemented("remove member"))
				orgs.PATCH("/:id/members/:user_id/role", handler.NotImplemented("update member role"))
			}

			// Monitoring
			monitoring := protected.Group("/monitoring")
			{
				monitoring.GET("/metrics", handler.NotImplemented("get metrics"))
				monitoring.GET("/logs", handler.NotImplemented("get logs"))
				monitoring.GET("/traces", handler.NotImplemented("get traces"))
				monitoring.GET("/alerts", handler.NotImplemented("list alerts"))
				monitoring.POST("/alerts", handler.NotImplemented("create alert"))
			}
		}

		// Admin routes (admin role required)
		admin := v1.Group("/admin")
		admin.Use(middleware.JWTAuth(jwtManager))
		admin.Use(middleware.RequireRole("admin", "super_admin"))
		{
			admin.GET("/users", handler.NotImplemented("admin list users"))
			admin.PATCH("/users/:id/status", handler.NotImplemented("admin update user status"))
			admin.GET("/models", handler.NotImplemented("admin list models"))
			admin.POST("/models", handler.NotImplemented("admin create model"))
			admin.PATCH("/models/:id", handler.NotImplemented("admin update model"))
			admin.GET("/stats", handler.NotImplemented("admin get stats"))
			admin.GET("/audit-logs", handler.NotImplemented("admin list audit logs"))
			admin.GET("/settings", handler.NotImplemented("admin get settings"))
			admin.PATCH("/settings", handler.NotImplemented("admin update settings"))
		}
	}
}
