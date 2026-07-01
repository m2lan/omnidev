// Package main is the entry point for the API Gateway.
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
	"github.com/omnidev/go-common/crypto"
	"github.com/omnidev/go-common/database"
	"github.com/omnidev/go-common/logger"
	"github.com/omnidev/go-common/middleware"
	"github.com/omnidev/go-common/telemetry"

	"github.com/omnidev/gateway/internal/adapter"
	"github.com/omnidev/gateway/internal/handler"
	"github.com/omnidev/gateway/internal/repository"
	"github.com/omnidev/gateway/internal/router"
	"github.com/omnidev/gateway/internal/service"
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

	logger.Log.Info("Starting API Gateway",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("build_time", buildTime),
		zap.String("env", cfg.App.Env),
	)

	// Initialize telemetry
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.Init(ctx, "gateway", version, cfg.Telemetry.OTLPEndpoint)
	if err != nil {
		logger.Log.Warn("Failed to init telemetry, continuing without it", zap.Error(err))
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
		logger.Log.Warn("Failed to connect to Redis, rate limiting disabled", zap.Error(err))
	}

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
		cfg.JWT.Issuer,
	)

	// Initialize repositories
	userRepository := repository.NewUserRepository(db.Pool)
	oauthRepository := repository.NewOAuthRepository(db.Pool)
	apiKeyRepository := repository.NewAPIKeyRepository(db.Pool)

	// Initialize Chat Service repositories
	convRepository := repository.NewConversationRepository(db.Pool)
	msgRepository := repository.NewMessageRepository(db.Pool)
	modelRepository := repository.NewModelRepository(db.Pool)

	// Initialize AI adapters
	adapterRegistry := adapter.NewRegistry()
	adapterRegistry.Register(adapter.NewDeepSeekAdapter(cfg.AI.DeepSeek))
	adapterRegistry.Register(adapter.NewOpenAIAdapter(cfg.AI.OpenAI))

	// Initialize encryption for user AI configs
	var adapterFactory *adapter.Factory
	var userAIConfigRepo repository.UserAIConfigRepository
	if cfg.Security.EncryptionKey != "" {
		encryptor, err := crypto.NewEncryptorFromString(cfg.Security.EncryptionKey)
		if err != nil {
			logger.Log.Warn("Failed to init encryptor, user AI configs disabled", zap.Error(err))
		} else {
			adapterFactory = adapter.NewFactory(encryptor)
			userAIConfigRepo = repository.NewUserAIConfigRepository(db.Pool)
		}
	}

	// Initialize services
	chatService := service.NewChatService(convRepository, msgRepository, modelRepository, adapterRegistry, adapterFactory, userAIConfigRepo, redisClient, cfg.AI.DefaultModel)

	// Initialize handlers
	healthHandler := handler.NewHealthHandler(version, commit, buildTime)
	authHandler := handler.NewAuthHandler(userRepository, oauthRepository, apiKeyRepository, jwtManager, redisClient, cfg)
	chatHandler := handler.NewChatHandler(chatService)
	userAIConfigHandler := handler.NewUserAIConfigProxyHandler("")

	// Setup Gin
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global middleware
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS([]string{"http://localhost:3000", "http://localhost:8080"}))

	// Rate limiter (if Redis is available)
	if redisClient != nil {
		rl := middleware.NewRateLimiter(30, 50)
		r.Use(rl.RateLimit())
	}

	// Setup routes
	router.Setup(r, jwtManager, healthHandler, authHandler, chatHandler, userAIConfigHandler)

	// HTTP server
	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second, // Long for SSE streams
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Log.Info("Server starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("Server forced to shutdown", zap.Error(err))
	}

	if redisClient != nil {
		redisClient.Close()
	}

	logger.Log.Info("Server exited")
}
