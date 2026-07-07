// Package config provides application configuration loading.
package config

import (
	"fmt"
	"time"
)

// Config is the root application configuration.
type Config struct {
	App        AppConfig        `mapstructure:"app"`
	Database   DatabaseConfig   `mapstructure:"db"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Kafka      KafkaConfig      `mapstructure:"kafka"`
	MinIO      MinIOConfig      `mapstructure:"minio"`
	ES         ESConfig         `mapstructure:"es"`
	Temporal   TemporalConfig   `mapstructure:"temporal"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	OAuth      OAuthConfig      `mapstructure:"oauth"`
	AI         AIConfig         `mapstructure:"ai"`
	Security   SecurityConfig   `mapstructure:"security"`
	Sandbox    SandboxConfig    `mapstructure:"sandbox"`
	Telemetry  TelemetryConfig  `mapstructure:"telemetry"`
	Tika       TikaConfig       `mapstructure:"tika"`
}

// AppConfig holds general application settings.
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`     // development, staging, production
	Debug   bool   `mapstructure:"debug"`
	Port    int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host           string        `mapstructure:"host"`
	Port           int           `mapstructure:"port"`
	User           string        `mapstructure:"user"`
	Password       string        `mapstructure:"password"`
	Name           string        `mapstructure:"name"`
	SSLMode        string        `mapstructure:"sslmode"`
	MaxOpenConns   int           `mapstructure:"max_open_conns"`
	MaxIdleConns   int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// DSN returns the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Password   string `mapstructure:"password"`
	DB         int    `mapstructure:"db"`
	MaxRetries int    `mapstructure:"max_retries"`
	PoolSize   int    `mapstructure:"pool_size"`
}

// Addr returns the Redis address.
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// KafkaConfig holds Kafka connection settings.
type KafkaConfig struct {
	Brokers         []string `mapstructure:"brokers"`
	GroupID         string   `mapstructure:"group_id"`
	AutoOffsetReset string   `mapstructure:"auto_offset_reset"`
}

// MinIOConfig holds MinIO connection settings.
type MinIOConfig struct {
	Endpoint   string `mapstructure:"endpoint"`
	AccessKey  string `mapstructure:"access_key"`
	SecretKey  string `mapstructure:"secret_key"`
	UseSSL     bool   `mapstructure:"use_ssl"`
	BucketPrefix string `mapstructure:"bucket_prefix"`
}

// ESConfig holds Elasticsearch connection settings.
type ESConfig struct {
	URL         string `mapstructure:"url"`
	IndexPrefix string `mapstructure:"index_prefix"`
}

// TemporalConfig holds Temporal connection settings.
type TemporalConfig struct {
	Host      string `mapstructure:"host"`
	Namespace string `mapstructure:"namespace"`
	TaskQueue string `mapstructure:"task_queue"`
}

// JWTConfig holds JWT token settings.
type JWTConfig struct {
	Secret        string        `mapstructure:"secret"`
	AccessExpiry  time.Duration `mapstructure:"access_expiry"`
	RefreshExpiry time.Duration `mapstructure:"refresh_expiry"`
	Issuer        string        `mapstructure:"issuer"`
}

// OAuthConfig holds OAuth provider settings.
type OAuthConfig struct {
	GitHub OAuthProvider `mapstructure:"github"`
	Google OAuthProvider `mapstructure:"google"`
}

// OAuthProvider holds settings for a single OAuth provider.
type OAuthProvider struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
}

// AIConfig holds AI provider settings.
type AIConfig struct {
	DefaultModel string             `mapstructure:"default_model"`
	OpenAI       AIProviderConfig   `mapstructure:"openai"`
	Anthropic    AIProviderConfig   `mapstructure:"anthropic"`
	Google       AIProviderConfig   `mapstructure:"google"`
	DeepSeek     AIProviderConfig   `mapstructure:"deepseek"`
	Qwen         AIProviderConfig   `mapstructure:"qwen"`
	Ollama       AIProviderConfig   `mapstructure:"ollama"`
}

// AIProviderConfig holds settings for a single AI provider.
type AIProviderConfig struct {
	APIKey  string   `mapstructure:"api_key"`
	BaseURL string   `mapstructure:"base_url"`
	Models  []string `mapstructure:"models"`
}

// SecurityConfig holds security-related settings.
type SecurityConfig struct {
	EncryptionKey string `mapstructure:"encryption_key"`
}

// SandboxConfig holds sandbox execution settings.
type SandboxConfig struct {
	RuntimeImage string `mapstructure:"runtime_image"`
	CPULimit     string `mapstructure:"cpu_limit"`
	MemoryLimit  string `mapstructure:"memory_limit"`
	Timeout      time.Duration `mapstructure:"timeout"`
}

// TelemetryConfig holds OpenTelemetry settings.
type TelemetryConfig struct {
	OTLPEndpoint string `mapstructure:"otlp_endpoint"`
}

// TikaConfig holds Apache Tika settings.
type TikaConfig struct {
	Endpoint string `mapstructure:"endpoint"` // e.g., http://localhost:9998
	Timeout  int    `mapstructure:"timeout"`  // Request timeout in seconds
}
