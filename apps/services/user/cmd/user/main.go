// Package main is the entry point for the User Service.
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

	"github.com/omnidev/services/user/internal/handler"
	"github.com/omnidev/services/user/internal/repository"
	"github.com/omnidev/services/user/internal/service"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(logger.Config{
		Level:  cfg.App.LogLevel,
		Format: "console",
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Log.Info("Starting User Service",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("env", cfg.App.Env),
	)

	ctx := context.Background()

	// Initialize telemetry
	shutdownTelemetry, err := telemetry.Init(ctx, "user-service", version, cfg.Telemetry.OTLPEndpoint)
	if err != nil {
		logger.Log.Warn("Failed to init telemetry", zap.Error(err))
	} else {
		defer func() {
			if err := shutdownTelemetry(ctx); err != nil {
				logger.Log.Error("Failed to shutdown telemetry", zap.Error(err))
			}
		}()
	}

	// Initialize database
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Redis
	redisClient, err := cache.NewRedis(cfg.Redis)
	if err != nil {
		logger.Log.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
		cfg.JWT.Issuer,
	)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.Pool)
	oauthRepo := repository.NewOAuthRepository(db.Pool)
	apiKeyRepo := repository.NewAPIKeyRepository(db.Pool)
	orgRepo := repository.NewOrganizationRepository(db.Pool)

	// Initialize services
	authSvc := service.NewAuthService(userRepo, oauthRepo, apiKeyRepo, jwtManager, redisClient, cfg)
	userSvc := service.NewUserService(userRepo, redisClient)
	orgSvc := service.NewOrganizationService(orgRepo, userRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(userSvc)
	orgHandler := handler.NewOrganizationHandler(orgSvc)

	// Setup Gin
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS([]string{"http://localhost:3000", "http://localhost:8080"}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "user"})
	})
	r.GET("/ready", func(c *gin.Context) {
		if err := db.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Routes
	v1 := r.Group("/api/v1")
	{
		// Public auth routes
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/refresh", authHandler.RefreshToken)
			authGroup.GET("/oauth/:provider", authHandler.OAuthRedirect)
			authGroup.GET("/callback/:provider", authHandler.OAuthCallback)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(jwtManager))
		{
			// User profile
			protected.GET("/users/me", userHandler.GetProfile)
			protected.PATCH("/users/me", userHandler.UpdateProfile)

			// API Keys
			protected.GET("/users/me/api-keys", authHandler.ListAPIKeys)
			protected.POST("/users/me/api-keys", authHandler.CreateAPIKey)
			protected.DELETE("/users/me/api-keys/:id", authHandler.RevokeAPIKey)

			// Organizations
			protected.GET("/organizations", orgHandler.ListOrganizations)
			protected.POST("/organizations", orgHandler.CreateOrganization)
			protected.GET("/organizations/:id", orgHandler.GetOrganization)
			protected.PATCH("/organizations/:id", orgHandler.UpdateOrganization)
			protected.GET("/organizations/:id/members", orgHandler.ListMembers)
			protected.POST("/organizations/:id/members/invite", orgHandler.InviteMember)
			protected.DELETE("/organizations/:id/members/:user_id", orgHandler.RemoveMember)
			protected.PATCH("/organizations/:id/members/:user_id/role", orgHandler.UpdateMemberRole)
		}
	}

	// HTTP server
	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Log.Info("User Service starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down User Service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("Server forced shutdown", zap.Error(err))
	}

	logger.Log.Info("User Service exited")
}
