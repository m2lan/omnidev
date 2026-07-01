// Package main is the entry point for the Chat Service.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/auth"
	"github.com/omnidev/go-common/cache"
	"github.com/omnidev/go-common/config"
	"github.com/omnidev/go-common/database"
	"github.com/omnidev/go-common/logger"
	"github.com/omnidev/go-common/middleware"
	"github.com/omnidev/go-common/telemetry"

	"github.com/omnidev/services/chat/internal/adapter"
	"github.com/omnidev/services/chat/internal/handler"
	"github.com/omnidev/services/chat/internal/repository"
	"github.com/omnidev/services/chat/internal/service"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(logger.Config{
		Level:  cfg.App.LogLevel,
		Format: "console",
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Log.Info("Starting Chat Service",
		zap.String("version", version),
		zap.String("env", cfg.App.Env),
	)

	ctx := context.Background()

	// Telemetry
	shutdownTelemetry, err := telemetry.Init(ctx, "chat-service", version, cfg.Telemetry.OTLPEndpoint)
	if err != nil {
		logger.Log.Warn("Failed to init telemetry", zap.Error(err))
	} else {
		defer func() {
			if err := shutdownTelemetry(ctx); err != nil {
				logger.Log.Error("Failed to shutdown telemetry", zap.Error(err))
			}
		}()
	}

	// Database
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Redis
	redisClient, err := cache.NewRedis(cfg.Redis)
	if err != nil {
		logger.Log.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	// JWT
	jwtManager := auth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
		cfg.JWT.Issuer,
	)

	// AI Adapters
	adapterRegistry := adapter.NewRegistry()
	adapterRegistry.Register(adapter.NewOpenAIAdapter(cfg.AI.OpenAI))
	adapterRegistry.Register(adapter.NewAnthropicAdapter(cfg.AI.Anthropic))
	adapterRegistry.Register(adapter.NewDeepSeekAdapter(cfg.AI.DeepSeek))
	adapterRegistry.Register(adapter.NewQwenAdapter(cfg.AI.Qwen))
	adapterRegistry.Register(adapter.NewOllamaAdapter(cfg.AI.Ollama))

	// Repositories
	convRepo := repository.NewConversationRepository(db.Pool)
	msgRepo := repository.NewMessageRepository(db.Pool)
	modelRepo := repository.NewModelRepository(db.Pool)
	promptRepo := repository.NewPromptRepository(db.Pool)

	// Services
	chatSvc := service.NewChatService(convRepo, msgRepo, modelRepo, adapterRegistry, redisClient, cfg.AI.DefaultModel)
	promptSvc := service.NewPromptService(promptRepo)

	// Handlers
	chatHandler := handler.NewChatHandler(chatSvc)
	promptHandler := handler.NewPromptHandler(promptSvc)
	modelHandler := handler.NewModelHandler(modelRepo)

	// Gin setup
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS([]string{"http://localhost:3000", "http://localhost:9090"}))

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "chat"})
	})
	r.GET("/ready", func(c *gin.Context) {
		if err := db.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Routes
	v1 := r.Group("/api/v1")
	v1.Use(middleware.JWTAuth(jwtManager))
	{
		// Conversations
		conv := v1.Group("/conversations")
		{
			conv.GET("", chatHandler.ListConversations)
			conv.POST("", chatHandler.CreateConversation)
			conv.GET("/:id", chatHandler.GetConversation)
			conv.PATCH("/:id", chatHandler.UpdateConversation)
			conv.DELETE("/:id", chatHandler.DeleteConversation)
			conv.GET("/:id/messages", chatHandler.ListMessages)
			conv.POST("/:id/messages", chatHandler.SendMessage)
			conv.POST("/:id/messages/stream", chatHandler.StreamMessage)
		}

		// Prompts
		prompts := v1.Group("/prompts")
		{
			prompts.GET("", promptHandler.ListPrompts)
			prompts.POST("", promptHandler.CreatePrompt)
			prompts.GET("/:id", promptHandler.GetPrompt)
			prompts.PATCH("/:id", promptHandler.UpdatePrompt)
			prompts.DELETE("/:id", promptHandler.DeletePrompt)
		}

		// Models
		models := v1.Group("/models")
		{
			models.GET("", modelHandler.ListModels)
		}
	}

	// Server
	addr := fmt.Sprintf(":%d", cfg.App.Port+1) // Chat on 8082
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Log.Info("Chat Service starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Chat Service...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("Server forced shutdown", zap.Error(err))
	}

	logger.Log.Info("Chat Service exited")
}
