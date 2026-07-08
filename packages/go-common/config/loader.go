package config

import (
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

	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.group_id", "omnidev")
	v.SetDefault("kafka.auto_offset_reset", "earliest")

	v.SetDefault("minio.endpoint", "localhost:9000")
	v.SetDefault("minio.access_key", "minioadmin")
	v.SetDefault("minio.secret_key", "minioadmin")
	v.SetDefault("minio.use_ssl", false)
	v.SetDefault("minio.bucket_prefix", "omnidev")

	v.SetDefault("es.url", "http://localhost:9200")
	v.SetDefault("es.index_prefix", "omnidev")

	v.SetDefault("temporal.host", "localhost:7233")
	v.SetDefault("temporal.namespace", "default")
	v.SetDefault("temporal.task_queue", "omnidev")

	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.access_expiry", "15m")
	v.SetDefault("jwt.refresh_expiry", "168h")
	v.SetDefault("jwt.issuer", "omnidev")

	v.SetDefault("ai.default_model", "deepseek-chat")
	v.SetDefault("ai.embedding_model", "gemini-embedding-2")

	v.SetDefault("sandbox.runtime_image", "omnidev/sandbox:latest")
	v.SetDefault("sandbox.cpu_limit", "2")
	v.SetDefault("sandbox.memory_limit", "2Gi")
	v.SetDefault("sandbox.timeout", "300s")

	v.SetDefault("telemetry.otlp_endpoint", "http://localhost:4317")

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

	return &cfg, nil
}

func bindEnvVars(v *viper.Viper) {
	envMap := map[string]string{
		"APP_ENV":             "app.env",
		"APP_DEBUG":           "app.debug",
		"APP_PORT":            "app.port",
		"APP_LOG_LEVEL":       "app.log_level",
		"DB_HOST":             "db.host",
		"DB_PORT":             "db.port",
		"DB_USER":             "db.user",
		"DB_PASSWORD":         "db.password",
		"DB_NAME":             "db.name",
		"DB_SSLMODE":          "db.sslmode",
		"DB_MAX_OPEN_CONNS":   "db.max_open_conns",
		"DB_MAX_IDLE_CONNS":   "db.max_idle_conns",
		"DB_CONN_MAX_LIFETIME": "db.conn_max_lifetime",
		"REDIS_HOST":          "redis.host",
		"REDIS_PORT":          "redis.port",
		"REDIS_PASSWORD":      "redis.password",
		"REDIS_DB":            "redis.db",
		"KAFKA_BROKERS":       "kafka.brokers",
		"KAFKA_GROUP_ID":      "kafka.group_id",
		"MINIO_ENDPOINT":      "minio.endpoint",
		"MINIO_ACCESS_KEY":    "minio.access_key",
		"MINIO_SECRET_KEY":    "minio.secret_key",
		"MINIO_USE_SSL":       "minio.use_ssl",
		"ES_URL":              "es.url",
		"TEMPORAL_HOST":       "temporal.host",
		"JWT_SECRET":          "jwt.secret",
		"JWT_ACCESS_EXPIRY":   "jwt.access_expiry",
		"JWT_REFRESH_EXPIRY":  "jwt.refresh_expiry",
		"JWT_ISSUER":          "jwt.issuer",
		"OPENAI_API_KEY":      "ai.openai.api_key",
		"OPENAI_BASE_URL":     "ai.openai.base_url",
		"OPENAI_MODELS":       "ai.openai.models",
		"ANTHROPIC_API_KEY":   "ai.anthropic.api_key",
		"GOOGLE_API_KEY":      "ai.google.api_key",
		"AI_DEFAULT_MODEL":    "ai.default_model",
		"RAG_EMBEDDING_MODEL": "ai.embedding_model",
		"DEEPSEEK_API_KEY":    "ai.deepseek.api_key",
		"DEEPSEEK_MODELS":     "ai.deepseek.models",
		"QWEN_API_KEY":        "ai.qwen.api_key",
		"OLLAMA_BASE_URL":     "ai.ollama.base_url",
		"OLLAMA_MODELS":       "ai.ollama.models",
		"ENCRYPTION_KEY":      "security.encryption_key",
		"GITHUB_CLIENT_ID":    "oauth.github.client_id",
		"GITHUB_CLIENT_SECRET": "oauth.github.client_secret",
		"GITHUB_REDIRECT_URL": "oauth.github.redirect_url",
		"GOOGLE_CLIENT_ID":    "oauth.google.client_id",
		"GOOGLE_CLIENT_SECRET": "oauth.google.client_secret",
		"GOOGLE_REDIRECT_URL": "oauth.google.redirect_url",
		"TIKA_ENDPOINT":       "tika.endpoint",
		"TIKA_TIMEOUT":        "tika.timeout",
	}

	for env, key := range envMap {
		_ = v.BindEnv(key, env) //nolint:errcheck
	}
}

// applyEnvFileOverrides copies raw .env file keys (e.g., DEEPSEEK_API_KEY) to their
// mapped config paths (e.g., ai.deepseek.api_key) so Unmarshal can find them.
func applyEnvFileOverrides(v *viper.Viper) {
	envMap := map[string]string{
		"APP_ENV":             "app.env",
		"APP_DEBUG":           "app.debug",
		"APP_PORT":            "app.port",
		"APP_LOG_LEVEL":       "app.log_level",
		"DB_HOST":             "db.host",
		"DB_PORT":             "db.port",
		"DB_USER":             "db.user",
		"DB_PASSWORD":         "db.password",
		"DB_NAME":             "db.name",
		"DB_SSLMODE":          "db.sslmode",
		"DB_MAX_OPEN_CONNS":   "db.max_open_conns",
		"DB_MAX_IDLE_CONNS":   "db.max_idle_conns",
		"DB_CONN_MAX_LIFETIME": "db.conn_max_lifetime",
		"REDIS_HOST":          "redis.host",
		"REDIS_PORT":          "redis.port",
		"REDIS_PASSWORD":      "redis.password",
		"REDIS_DB":            "redis.db",
		"KAFKA_BROKERS":       "kafka.brokers",
		"KAFKA_GROUP_ID":      "kafka.group_id",
		"MINIO_ENDPOINT":      "minio.endpoint",
		"MINIO_ACCESS_KEY":    "minio.access_key",
		"MINIO_SECRET_KEY":    "minio.secret_key",
		"MINIO_USE_SSL":       "minio.use_ssl",
		"ES_URL":              "es.url",
		"TEMPORAL_HOST":       "temporal.host",
		"JWT_SECRET":          "jwt.secret",
		"JWT_ACCESS_EXPIRY":   "jwt.access_expiry",
		"JWT_REFRESH_EXPIRY":  "jwt.refresh_expiry",
		"JWT_ISSUER":          "jwt.issuer",
		"OPENAI_API_KEY":      "ai.openai.api_key",
		"OPENAI_BASE_URL":     "ai.openai.base_url",
		"OPENAI_MODELS":       "ai.openai.models",
		"ANTHROPIC_API_KEY":   "ai.anthropic.api_key",
		"GOOGLE_API_KEY":      "ai.google.api_key",
		"AI_DEFAULT_MODEL":    "ai.default_model",
		"RAG_EMBEDDING_MODEL": "ai.embedding_model",
		"DEEPSEEK_API_KEY":    "ai.deepseek.api_key",
		"DEEPSEEK_MODELS":     "ai.deepseek.models",
		"QWEN_API_KEY":        "ai.qwen.api_key",
		"OLLAMA_BASE_URL":     "ai.ollama.base_url",
		"OLLAMA_MODELS":       "ai.ollama.models",
		"ENCRYPTION_KEY":      "security.encryption_key",
		"GITHUB_CLIENT_ID":    "oauth.github.client_id",
		"GITHUB_CLIENT_SECRET": "oauth.github.client_secret",
		"GITHUB_REDIRECT_URL": "oauth.github.redirect_url",
		"GOOGLE_CLIENT_ID":    "oauth.google.client_id",
		"GOOGLE_CLIENT_SECRET": "oauth.google.client_secret",
		"GOOGLE_REDIRECT_URL": "oauth.google.redirect_url",
		"TIKA_ENDPOINT":       "tika.endpoint",
		"TIKA_TIMEOUT":        "tika.timeout",
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
