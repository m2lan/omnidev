// Package domain defines the core business entities for the Deploy Service.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// DeployStatus represents the status of a deployment.
type DeployStatus string

const (
	DeployStatusPending   DeployStatus = "pending"
	DeployStatusBuilding  DeployStatus = "building"
	DeployStatusDeploying DeployStatus = "deploying"
	DeployStatusRunning   DeployStatus = "running"
	DeployStatusFailed    DeployStatus = "failed"
	DeployStatusStopped   DeployStatus = "stopped"
)

// DeployPlatform represents the deployment target.
type DeployPlatform string

const (
	PlatformDocker     DeployPlatform = "docker"
	PlatformKubernetes DeployPlatform = "kubernetes"
	PlatformAWS        DeployPlatform = "aws"
	PlatformAzure      DeployPlatform = "azure"
	PlatformGCP        DeployPlatform = "gcp"
)

// Environment represents the deployment environment.
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)

// Deployment represents a deployment record.
type Deployment struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	ProjectID   uuid.UUID              `json:"project_id" db:"project_id"`
	UserID      uuid.UUID              `json:"user_id" db:"user_id"`
	Version     string                 `json:"version" db:"version"`
	Environment Environment            `json:"environment" db:"environment"`
	Platform    DeployPlatform         `json:"platform" db:"platform"`
	Status      DeployStatus           `json:"status" db:"status"`
	Config      DeployConfig           `json:"config" db:"config"`
	Domain      *string                `json:"domain,omitempty" db:"domain"`
	URL         *string                `json:"url,omitempty" db:"url"`
	ImageTag    *string                `json:"image_tag,omitempty" db:"image_tag"`
	BuildLogs   *string                `json:"build_logs,omitempty" db:"build_logs"`
	Error       *string                `json:"error,omitempty" db:"error"`
	ResourceUsage *ResourceUsage       `json:"resource_usage,omitempty" db:"resource_usage"`
	StartedAt   *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// DeployConfig holds deployment configuration.
type DeployConfig struct {
	// Build
	BuildCommand  string `json:"build_command,omitempty"`
	Dockerfile    string `json:"dockerfile,omitempty"`
	BuildArgs     map[string]string `json:"build_args,omitempty"`

	// Runtime
	Port          int    `json:"port,omitempty"`
	Replicas      int    `json:"replicas,omitempty"`
	MemoryLimit   string `json:"memory_limit,omitempty"`
	CPULimit      string `json:"cpu_limit,omitempty"`

	// Environment
	EnvVars       map[string]string `json:"env_vars,omitempty"`
	Secrets       map[string]string `json:"secrets,omitempty"`

	// Domain
	CustomDomain  string `json:"custom_domain,omitempty"`
	EnableSSL     bool   `json:"enable_ssl,omitempty"`

	// Health
	HealthCheckPath string `json:"health_check_path,omitempty"`
}

// ResourceUsage represents current resource usage.
type ResourceUsage struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryMB      int     `json:"memory_mb"`
	MemoryLimitMB int     `json:"memory_limit_mb"`
	DiskMB        int     `json:"disk_mb"`
	NetworkInMB   float64 `json:"network_in_mb"`
	NetworkOutMB  float64 `json:"network_out_mb"`
}

// DomainConfig represents domain configuration.
type DomainConfig struct {
	Domain     string `json:"domain"`
	TargetPort int    `json:"target_port"`
	EnableSSL  bool   `json:"enable_ssl"`
	SSLCert    string `json:"ssl_cert,omitempty"`
	SSLKey     string `json:"ssl_key,omitempty"`
}
