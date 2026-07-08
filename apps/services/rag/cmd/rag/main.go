// Package main is the entry point for the RAG Service.
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
	"github.com/omnidev/go-common/storage"
	"github.com/omnidev/go-common/telemetry"

	"github.com/omnidev/services/rag/internal/chunker"
	"github.com/omnidev/services/rag/internal/embedder"
	"github.com/omnidev/services/rag/internal/handler"
	"github.com/omnidev/services/rag/internal/parser"
	"github.com/omnidev/services/rag/internal/repository"
	"github.com/omnidev/services/rag/internal/retriever"
	"github.com/omnidev/services/rag/internal/service"
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

	logger.Log.Info("Starting RAG Service", zap.String("version", version), zap.String("env", cfg.App.Env))

	ctx := context.Background()

	// Telemetry
	shutdownTel, err := telemetry.Init(ctx, "rag-service", version, cfg.Telemetry.OTLPEndpoint)
	if err != nil {
		logger.Log.Warn("Telemetry init failed", zap.Error(err))
	} else {
		defer func() { _ = shutdownTel(ctx) }()
	}

	// Database
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		logger.Log.Fatal("DB connect failed", zap.Error(err))
	}
	defer db.Close()

	// Redis
	redisClient, err := cache.NewRedis(cfg.Redis)
	if err != nil {
		logger.Log.Fatal("Redis connect failed", zap.Error(err))
	}
	defer redisClient.Close()

	// MinIO
	minioClient, err := storage.NewMinIO(cfg.MinIO)
	if err != nil {
		logger.Log.Fatal("MinIO connect failed", zap.Error(err))
	}

	// JWT
	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry, cfg.JWT.Issuer)

	// Components
	docParser := parser.NewDocParser()
	chunker := chunker.NewSemanticChunker(512, 50)

	// Select embedding provider based on config
	var emb embedder.Embedder
	switch cfg.AI.EmbeddingModel {
	case "gemini-embedding-2":
		emb = embedder.NewGeminiEmbedder(cfg.AI.Google)
		logger.Log.Info("Using Gemini embedding", zap.String("model", cfg.AI.EmbeddingModel))
	default:
		emb = embedder.NewOpenAIEmbedder(cfg.AI.OpenAI)
		logger.Log.Info("Using OpenAI embedding", zap.String("model", cfg.AI.EmbeddingModel))
	}

	retriever := retriever.NewHybridRetriever(db.Pool, emb)

	// Repositories
	kbRepo := repository.NewKnowledgeBaseRepository(db.Pool)
	docRepo := repository.NewDocumentRepository(db.Pool)
	chunkRepo := repository.NewChunkRepository(db.Pool)

	// Services
	kbSvc := service.NewKnowledgeBaseService(kbRepo, docRepo, chunkRepo, minioClient, docParser, chunker, emb)
	searchSvc := service.NewSearchService(retriever, chunkRepo)

	// Handlers
	kbHandler := handler.NewKnowledgeBaseHandler(kbSvc)
	searchHandler := handler.NewSearchHandler(searchSvc)

	// Gin
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Logger(), middleware.Recovery())
	r.Use(middleware.CORS([]string{"*"}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "rag"})
	})

	v1 := r.Group("/api/v1")
	v1.Use(middleware.JWTAuth(jwtManager))
	{
		kb := v1.Group("/knowledge")
		{
			kb.GET("", kbHandler.ListKnowledgeBases)
			kb.POST("", kbHandler.CreateKnowledgeBase)
			kb.GET("/:id", kbHandler.GetKnowledgeBase)
			kb.PATCH("/:id", kbHandler.UpdateKnowledgeBase)
			kb.DELETE("/:id", kbHandler.DeleteKnowledgeBase)
			kb.GET("/:id/documents", kbHandler.ListDocuments)
			kb.POST("/:id/documents", kbHandler.UploadDocument)
			kb.DELETE("/:id/documents/:doc_id", kbHandler.DeleteDocument)
			kb.POST("/:id/search", searchHandler.Search)
		}
	}

	addr := fmt.Sprintf(":%d", cfg.App.Port+2) // RAG on 8083
	srv := &http.Server{Addr: addr, Handler: r, ReadTimeout: 30 * time.Second, WriteTimeout: 300 * time.Second}

	go func() {
		logger.Log.Info("RAG Service starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down RAG Service...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Log.Info("RAG Service exited")
}
