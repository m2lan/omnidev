package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Load reads configuration from environment variables and optional config file.
func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("app.name", "omnidev")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.debug", true)
	v.SetDefault("app.port", 9090)
	v.SetDefault("app.log_level", "debug")

	v.SetDefault("db.host", "localhost")
	v.SetDefault("db.port", 5432)
	v.SetDefault("db.user", "omnidev")
	v.SetDefault("db.password", "omnidev")
	v.SetDefault("db.name", "omnidev")
	v.SetDefault("db.sslmode", "disable")
	v.SetDefault("db.max_open_conns", 25)
	v.SetDefault("db.max_idle_conns", 5)
	v.SetDefault("db.conn_max_lifetime", "5m")

	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.pool_size", 10)

	v.SetDefault("minio.endpoint", "localhost:9000")
	v.SetDefault("minio.access_key", "minioadmin")
	v.SetDefault("minio.secret_key", "minioadmin")
	v.SetDefault("minio.use_ssl", false)
	v.SetDefault("minio.bucket_prefix", "omnidev")

	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.access_expiry", "15m")
	v.SetDefault("jwt.refresh_expiry", "168h")
	v.SetDefault("jwt.issuer", "omnidev")

	v.SetDefault("ai.default_model", "deepseek-chat")
	v.SetDefault("ai.embedding_provider", "openai")
	v.SetDefault("ai.embedding_model", "text-embedding-3-small")

	v.SetDefault("sandbox.enabled", true)
	v.SetDefault("sandbox.workspace_root", "./data/agent-workspaces")
	v.SetDefault("sandbox.network", "none")
	v.SetDefault("sandbox.python_image", "python:3.12-alpine")
	v.SetDefault("sandbox.node_image", "node:22-alpine")
	v.SetDefault("sandbox.go_image", "golang:1.22-alpine")
	v.SetDefault("sandbox.shell_image", "alpine:3.19")
	v.SetDefault("sandbox.cpu_limit", "1")
	v.SetDefault("sandbox.memory_limit", "512m")
	v.SetDefault("sandbox.timeout", "120s")

	v.SetDefault("telemetry.otlp_endpoint", "http://localhost:4317")

	v.SetDefault("knowledge.storage_type", "local")
	v.SetDefault("knowledge.bucket", "rag-documents")
	v.SetDefault("knowledge.local_path", "./data/knowledge")
	v.SetDefault("knowledge.parser_type", "tika")
	v.SetDefault("knowledge.cache_ttl", "60s")
	v.SetDefault("knowledge.embedding_dimensions", 0)

	v.SetDefault("mcp.enabled", false)

	// Environment variables
	v.SetEnvPrefix("OMNIDEV")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Also read standard env vars
	bindEnvVars(v)

	// Config file (optional)
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath(findProjectRoot())
	if err := v.ReadInConfig(); err == nil {
		// .env keys like DEEPSEEK_API_KEY are read as-is, not mapped to ai.deepseek.api_key.
		// Copy raw .env keys to their mapped config paths.
		applyEnvFileOverrides(v)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Parse durations manually since viper doesn't handle them well
	var err error
	cfg.Database.ConnMaxLifetime, err = time.ParseDuration(v.GetString("db.conn_max_lifetime"))
	if err != nil {
		cfg.Database.ConnMaxLifetime = 5 * time.Minute
	}

	cfg.JWT.AccessExpiry, err = time.ParseDuration(v.GetString("jwt.access_expiry"))
	if err != nil {
		cfg.JWT.AccessExpiry = 15 * time.Minute
	}

	cfg.JWT.RefreshExpiry, err = time.ParseDuration(v.GetString("jwt.refresh_expiry"))
	if err != nil {
		cfg.JWT.RefreshExpiry = 168 * time.Hour
	}

	cfg.Sandbox.Timeout, err = time.ParseDuration(v.GetString("sandbox.timeout"))
	if err != nil {
		cfg.Sandbox.Timeout = 300 * time.Second
	}

	cfg.Knowledge.CacheTTL, err = time.ParseDuration(v.GetString("knowledge.cache_ttl"))
	if err != nil {
		cfg.Knowledge.CacheTTL = 60 * time.Second
	}

	// MCP servers are configured as a JSON array; viper cannot unmarshal a
	// heterogeneous list of servers from a single environment variable.
	if raw := strings.TrimSpace(os.Getenv("MCP_SERVERS")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &cfg.MCP.Servers); err != nil {
			return nil, fmt.Errorf("parse MCP_SERVERS: %w", err)
		}
	}

	return &cfg, nil
}

func bindEnvVars(v *viper.Viper) {
	envMap := map[string]string{
		"APP_ENV":                        "app.env",
		"APP_DEBUG":                      "app.debug",
		"APP_PORT":                       "app.port",
		"APP_LOG_LEVEL":                  "app.log_level",
		"DB_HOST":                        "db.host",
		"DB_PORT":                        "db.port",
		"DB_USER":                        "db.user",
		"DB_PASSWORD":                    "db.password",
		"DB_NAME":                        "db.name",
		"DB_SSLMODE":                     "db.sslmode",
		"DB_MAX_OPEN_CONNS":              "db.max_open_conns",
		"DB_MAX_IDLE_CONNS":              "db.max_idle_conns",
		"DB_CONN_MAX_LIFETIME":           "db.conn_max_lifetime",
		"REDIS_HOST":                     "redis.host",
		"REDIS_PORT":                     "redis.port",
		"REDIS_PASSWORD":                 "redis.password",
		"REDIS_DB":                       "redis.db",
		"MINIO_ENDPOINT":                 "minio.endpoint",
		"MINIO_ACCESS_KEY":               "minio.access_key",
		"MINIO_SECRET_KEY":               "minio.secret_key",
		"MINIO_USE_SSL":                  "minio.use_ssl",
		"JWT_SECRET":                     "jwt.secret",
		"JWT_ACCESS_EXPIRY":              "jwt.access_expiry",
		"JWT_REFRESH_EXPIRY":             "jwt.refresh_expiry",
		"JWT_ISSUER":                     "jwt.issuer",
		"OPENAI_API_KEY":                 "ai.openai.api_key",
		"OPENAI_BASE_URL":                "ai.openai.base_url",
		"OPENAI_MODELS":                  "ai.openai.models",
		"ANTHROPIC_API_KEY":              "ai.anthropic.api_key",
		"GOOGLE_API_KEY":                 "ai.google.api_key",
		"AI_DEFAULT_MODEL":               "ai.default_model",
		"RAG_EMBEDDING_PROVIDER":         "ai.embedding_provider",
		"RAG_EMBEDDING_MODEL":            "ai.embedding_model",
		"RAG_EMBEDDING_API_KEY":          "ai.embedding_api_key",
		"RAG_EMBEDDING_BASE_URL":         "ai.embedding_base_url",
		"DEEPSEEK_API_KEY":               "ai.deepseek.api_key",
		"DEEPSEEK_MODELS":                "ai.deepseek.models",
		"QWEN_API_KEY":                   "ai.qwen.api_key",
		"OLLAMA_BASE_URL":                "ai.ollama.base_url",
		"OLLAMA_MODELS":                  "ai.ollama.models",
		"ENCRYPTION_KEY":                 "security.encryption_key",
		"GITHUB_CLIENT_ID":               "oauth.github.client_id",
		"GITHUB_CLIENT_SECRET":           "oauth.github.client_secret",
		"GITHUB_REDIRECT_URL":            "oauth.github.redirect_url",
		"GOOGLE_CLIENT_ID":               "oauth.google.client_id",
		"GOOGLE_CLIENT_SECRET":           "oauth.google.client_secret",
		"GOOGLE_REDIRECT_URL":            "oauth.google.redirect_url",
		"AGENT_MODEL":                    "agent.model",
		"AGENT_BASE_URL":                 "agent.base_url",
		"AGENT_API_KEY":                  "agent.api_key",
		"SANDBOX_ENABLED":                "sandbox.enabled",
		"SANDBOX_WORKSPACE_ROOT":         "sandbox.workspace_root",
		"SANDBOX_HOST_WORKSPACE_ROOT":    "sandbox.host_workspace_root",
		"SANDBOX_NETWORK":                "sandbox.network",
		"SANDBOX_PYTHON_IMAGE":           "sandbox.python_image",
		"SANDBOX_NODE_IMAGE":             "sandbox.node_image",
		"SANDBOX_GO_IMAGE":               "sandbox.go_image",
		"SANDBOX_SHELL_IMAGE":            "sandbox.shell_image",
		"SANDBOX_CPU_LIMIT":              "sandbox.cpu_limit",
		"SANDBOX_MEMORY_LIMIT":           "sandbox.memory_limit",
		"SANDBOX_TIMEOUT":                "sandbox.timeout",
		"TIKA_ENDPOINT":                  "tika.endpoint",
		"TIKA_TIMEOUT":                   "tika.timeout",
		"KNOWLEDGE_STORAGE_TYPE":         "knowledge.storage_type",
		"KNOWLEDGE_BUCKET":               "knowledge.bucket",
		"KNOWLEDGE_LOCAL_PATH":           "knowledge.local_path",
		"KNOWLEDGE_PARSER_TYPE":          "knowledge.parser_type",
		"KNOWLEDGE_CACHE_TTL":            "knowledge.cache_ttl",
		"KNOWLEDGE_EMBEDDING_DIMENSIONS": "knowledge.embedding_dimensions",
		"MCP_ENABLED":                    "mcp.enabled",
	}

	for env, key := range envMap {
		_ = v.BindEnv(key, env) //nolint:errcheck
	}
}

// applyEnvFileOverrides copies raw .env file keys (e.g., DEEPSEEK_API_KEY) to their
// mapped config paths (e.g., ai.deepseek.api_key) so Unmarshal can find them.
func applyEnvFileOverrides(v *viper.Viper) {
	envMap := map[string]string{
		"APP_ENV":                        "app.env",
		"APP_DEBUG":                      "app.debug",
		"APP_PORT":                       "app.port",
		"APP_LOG_LEVEL":                  "app.log_level",
		"DB_HOST":                        "db.host",
		"DB_PORT":                        "db.port",
		"DB_USER":                        "db.user",
		"DB_PASSWORD":                    "db.password",
		"DB_NAME":                        "db.name",
		"DB_SSLMODE":                     "db.sslmode",
		"DB_MAX_OPEN_CONNS":              "db.max_open_conns",
		"DB_MAX_IDLE_CONNS":              "db.max_idle_conns",
		"DB_CONN_MAX_LIFETIME":           "db.conn_max_lifetime",
		"REDIS_HOST":                     "redis.host",
		"REDIS_PORT":                     "redis.port",
		"REDIS_PASSWORD":                 "redis.password",
		"REDIS_DB":                       "redis.db",
		"MINIO_ENDPOINT":                 "minio.endpoint",
		"MINIO_ACCESS_KEY":               "minio.access_key",
		"MINIO_SECRET_KEY":               "minio.secret_key",
		"MINIO_USE_SSL":                  "minio.use_ssl",
		"JWT_SECRET":                     "jwt.secret",
		"JWT_ACCESS_EXPIRY":              "jwt.access_expiry",
		"JWT_REFRESH_EXPIRY":             "jwt.refresh_expiry",
		"JWT_ISSUER":                     "jwt.issuer",
		"OPENAI_API_KEY":                 "ai.openai.api_key",
		"OPENAI_BASE_URL":                "ai.openai.base_url",
		"OPENAI_MODELS":                  "ai.openai.models",
		"ANTHROPIC_API_KEY":              "ai.anthropic.api_key",
		"GOOGLE_API_KEY":                 "ai.google.api_key",
		"AI_DEFAULT_MODEL":               "ai.default_model",
		"RAG_EMBEDDING_PROVIDER":         "ai.embedding_provider",
		"RAG_EMBEDDING_MODEL":            "ai.embedding_model",
		"RAG_EMBEDDING_API_KEY":          "ai.embedding_api_key",
		"RAG_EMBEDDING_BASE_URL":         "ai.embedding_base_url",
		"DEEPSEEK_API_KEY":               "ai.deepseek.api_key",
		"DEEPSEEK_MODELS":                "ai.deepseek.models",
		"QWEN_API_KEY":                   "ai.qwen.api_key",
		"OLLAMA_BASE_URL":                "ai.ollama.base_url",
		"OLLAMA_MODELS":                  "ai.ollama.models",
		"ENCRYPTION_KEY":                 "security.encryption_key",
		"GITHUB_CLIENT_ID":               "oauth.github.client_id",
		"GITHUB_CLIENT_SECRET":           "oauth.github.client_secret",
		"GITHUB_REDIRECT_URL":            "oauth.github.redirect_url",
		"GOOGLE_CLIENT_ID":               "oauth.google.client_id",
		"GOOGLE_CLIENT_SECRET":           "oauth.google.client_secret",
		"GOOGLE_REDIRECT_URL":            "oauth.google.redirect_url",
		"AGENT_MODEL":                    "agent.model",
		"AGENT_BASE_URL":                 "agent.base_url",
		"AGENT_API_KEY":                  "agent.api_key",
		"SANDBOX_ENABLED":                "sandbox.enabled",
		"SANDBOX_WORKSPACE_ROOT":         "sandbox.workspace_root",
		"SANDBOX_HOST_WORKSPACE_ROOT":    "sandbox.host_workspace_root",
		"SANDBOX_NETWORK":                "sandbox.network",
		"SANDBOX_PYTHON_IMAGE":           "sandbox.python_image",
		"SANDBOX_NODE_IMAGE":             "sandbox.node_image",
		"SANDBOX_GO_IMAGE":               "sandbox.go_image",
		"SANDBOX_SHELL_IMAGE":            "sandbox.shell_image",
		"SANDBOX_CPU_LIMIT":              "sandbox.cpu_limit",
		"SANDBOX_MEMORY_LIMIT":           "sandbox.memory_limit",
		"SANDBOX_TIMEOUT":                "sandbox.timeout",
		"TIKA_ENDPOINT":                  "tika.endpoint",
		"TIKA_TIMEOUT":                   "tika.timeout",
		"KNOWLEDGE_STORAGE_TYPE":         "knowledge.storage_type",
		"KNOWLEDGE_BUCKET":               "knowledge.bucket",
		"KNOWLEDGE_LOCAL_PATH":           "knowledge.local_path",
		"KNOWLEDGE_PARSER_TYPE":          "knowledge.parser_type",
		"KNOWLEDGE_CACHE_TTL":            "knowledge.cache_ttl",
		"KNOWLEDGE_EMBEDDING_DIMENSIONS": "knowledge.embedding_dimensions",
		"MCP_ENABLED":                    "mcp.enabled",
	}

	for envKey, cfgKey := range envMap {
		if v.GetString(envKey) != "" {
			v.Set(cfgKey, v.GetString(envKey))
		}
	}

	// Parse comma-separated model lists
	modelKeys := []string{
		"ai.openai.models",
		"ai.deepseek.models",
		"ai.ollama.models",
		"ai.anthropic.models",
		"ai.google.models",
		"ai.qwen.models",
	}
	for _, key := range modelKeys {
		val := v.GetString(key)
		if val != "" && strings.Contains(val, ",") {
			v.Set(key, strings.Split(val, ","))
		}
	}
}

// findProjectRoot walks up from the current working directory to find
// a directory containing .env file.
func findProjectRoot() string {
	dir, _ := filepath.Abs(".")
	for {
		if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}
