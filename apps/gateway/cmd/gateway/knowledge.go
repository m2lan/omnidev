package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/omnidev/go-common/config"
	"github.com/omnidev/go-common/logger"
	kengine "github.com/omnidev/knowledge-engine"
)

// knowledgeLogger adapts the shared zap logger to the knowledge engine's
// structured logger interface.
type knowledgeLogger struct{}

func (knowledgeLogger) Debug(msg string, kv ...any) { logger.Log.Sugar().Debugw(msg, kv...) }
func (knowledgeLogger) Info(msg string, kv ...any)  { logger.Log.Sugar().Infow(msg, kv...) }
func (knowledgeLogger) Warn(msg string, kv ...any)  { logger.Log.Sugar().Warnw(msg, kv...) }
func (knowledgeLogger) Error(msg string, kv ...any) { logger.Log.Sugar().Errorw(msg, kv...) }

// resolveKnowledgeConfig maps the gateway configuration onto knowledge-engine
// settings, including embedding provider credentials and storage backend.
func resolveKnowledgeConfig(cfg *config.Config) kengine.Config {
	ai := cfg.AI
	apiKey := ai.EmbeddingAPIKey
	if apiKey == "" {
		switch ai.EmbeddingProvider {
		case "gemini":
			apiKey = ai.Google.APIKey
		case "openai":
			apiKey = ai.OpenAI.APIKey
		case "ollama":
			apiKey = ai.Ollama.APIKey
		}
	}
	baseURL := ai.EmbeddingBaseURL
	if baseURL == "" {
		switch ai.EmbeddingProvider {
		case "ollama":
			baseURL = ai.Ollama.BaseURL
		case "openai":
			baseURL = ai.OpenAI.BaseURL
		}
	}

	storageType := cfg.Knowledge.StorageType
	if storageType == "" {
		if cfg.MinIO.Endpoint != "" {
			storageType = "minio"
		} else {
			storageType = "local"
		}
	}
	bucket := cfg.Knowledge.Bucket
	if bucket == "" {
		bucket = "rag-documents"
	}
	localPath := cfg.Knowledge.LocalPath
	if localPath == "" {
		localPath = "./data/knowledge"
	}

	kcfg := kengine.Config{
		Cache: kengine.CacheConfig{SearchTTL: cfg.Knowledge.CacheTTL},
		Storage: kengine.StorageConfig{
			Type:      storageType,
			Bucket:    bucket,
			LocalPath: localPath,
			Endpoint:  cfg.MinIO.Endpoint,
			AccessKey: cfg.MinIO.AccessKey,
			SecretKey: cfg.MinIO.SecretKey,
			UseSSL:    cfg.MinIO.UseSSL,
		},
		Embedding: kengine.EmbeddingConfig{
			Provider:   ai.EmbeddingProvider,
			Model:      ai.EmbeddingModel,
			APIKey:     apiKey,
			BaseURL:    baseURL,
			Dimensions: cfg.Knowledge.EmbeddingDimensions,
		},
		Parser: kengine.ParserConfig{
			Type:         cfg.Knowledge.ParserType,
			TikaEndpoint: cfg.Tika.Endpoint,
			TikaTimeout:  time.Duration(cfg.Tika.Timeout) * time.Second,
		},
		Auth:   kengine.AuthConfig{Type: "noop"},
		Logger: knowledgeLogger{},
	}
	if cfg.Redis.Host != "" {
		kcfg.RedisURL = redisURL(cfg.Redis)
	}
	return kcfg
}

func redisURL(cfg config.RedisConfig) string {
	u := url.URL{Scheme: "redis", Host: cfg.Addr()}
	if cfg.Password != "" {
		u.User = url.UserPassword("", cfg.Password)
	}
	u.Path = fmt.Sprintf("/%d", cfg.DB)
	return u.String()
}
