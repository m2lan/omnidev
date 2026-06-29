// Package service contains the business logic for the Deploy Service.
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/services/deploy/internal/builder"
	"github.com/omnidev/services/deploy/internal/domain"
	"github.com/omnidev/services/deploy/internal/platforms"
	"github.com/omnidev/services/deploy/internal/repository"
)

// DeployService handles deployment operations.
type DeployService struct {
	deployRepo  repository.DeployRepository
	builder     builder.Builder
	platformReg *platforms.Registry
}

// NewDeployService creates a new deploy service.
func NewDeployService(
	deployRepo repository.DeployRepository,
	builder builder.Builder,
	platformReg *platforms.Registry,
) *DeployService {
	return &DeployService{
		deployRepo:  deployRepo,
		builder:     builder,
		platformReg: platformReg,
	}
}

// CreateDeploymentInput defines the input for creating a deployment.
type CreateDeploymentInput struct {
	ProjectID   uuid.UUID          `json:"project_id" validate:"required"`
	Version     string             `json:"version" validate:"required"`
	Environment string             `json:"environment"`
	Platform    string             `json:"platform" validate:"required"`
	Config      domain.DeployConfig `json:"config"`
	Domain      string             `json:"domain"`
}

// CreateDeployment creates and starts a new deployment.
func (s *DeployService) CreateDeployment(ctx context.Context, userID uuid.UUID, input *CreateDeploymentInput) (*domain.Deployment, error) {
	env := domain.EnvDevelopment
	if input.Environment != "" {
		env = domain.Environment(input.Environment)
	}

	deploy := &domain.Deployment{
		ID:          uuid.New(),
		ProjectID:   input.ProjectID,
		UserID:      userID,
		Version:     input.Version,
		Environment: env,
		Platform:    domain.DeployPlatform(input.Platform),
		Status:      domain.DeployStatusPending,
		Config:      input.Config,
		Metadata:    map[string]interface{}{},
	}

	if input.Domain != "" {
		deploy.Domain = &input.Domain
	}

	if err := s.deployRepo.Create(ctx, deploy); err != nil {
		return nil, appErr.Wrap(err, "failed to create deployment")
	}

	// Start deployment in background
	go s.executeDeployment(context.Background(), deploy)

	return deploy, nil
}

// executeDeployment runs the full deployment pipeline.
func (s *DeployService) executeDeployment(ctx context.Context, deploy *domain.Deployment) {
	logger.Log.Info("Starting deployment",
		zap.String("deploy_id", deploy.ID.String()),
		zap.String("platform", string(deploy.Platform)),
	)

	// Step 1: Build
	_ = s.deployRepo.UpdateStatus(ctx, deploy.ID, domain.DeployStatusBuilding, nil)

	buildOpts := builder.BuildOptions{
		ProjectPath:  fmt.Sprintf("/workspace/%s", deploy.ProjectID.String()),
		ImageName:    fmt.Sprintf("omnidev/%s", deploy.ProjectID.String()),
		ImageTag:     "",
		Dockerfile:   deploy.Config.Dockerfile,
		BuildCommand: deploy.Config.BuildCommand,
		BuildArgs:    deploy.Config.BuildArgs,
	}

	buildResult, err := s.builder.Build(ctx, buildOpts)
	if err != nil {
		errMsg := err.Error()
		_ = s.deployRepo.UpdateStatus(ctx, deploy.ID, domain.DeployStatusFailed, &errMsg)
		return
	}

	_ = s.deployRepo.UpdateBuildInfo(ctx, deploy.ID, buildResult.ImageTag, buildResult.BuildLogs)

	// Step 2: Deploy
	_ = s.deployRepo.UpdateStatus(ctx, deploy.ID, domain.DeployStatusDeploying, nil)

	platform, err := s.platformReg.Get(string(deploy.Platform))
	if err != nil {
		errMsg := err.Error()
		_ = s.deployRepo.UpdateStatus(ctx, deploy.ID, domain.DeployStatusFailed, &errMsg)
		return
	}

	deployOpts := platforms.DeployOptions{
		DeploymentID: deploy.ID.String(),
		ImageTag:     buildResult.ImageTag,
		Port:         deploy.Config.Port,
		Replicas:     deploy.Config.Replicas,
		EnvVars:      deploy.Config.EnvVars,
		MemoryLimit:  deploy.Config.MemoryLimit,
		CPULimit:     deploy.Config.CPULimit,
		Domain:       deploy.Config.CustomDomain,
		EnableSSL:    deploy.Config.EnableSSL,
	}

	result, err := platform.Deploy(ctx, deployOpts)
	if err != nil {
		errMsg := err.Error()
		_ = s.deployRepo.UpdateStatus(ctx, deploy.ID, domain.DeployStatusFailed, &errMsg)
		return
	}

	if result.URL != "" {
		_ = s.deployRepo.UpdateURL(ctx, deploy.ID, result.URL)
	}

	_ = s.deployRepo.UpdateStatus(ctx, deploy.ID, domain.DeployStatusRunning, nil)

	logger.Log.Info("Deployment succeeded",
		zap.String("deploy_id", deploy.ID.String()),
		zap.String("url", result.URL),
	)
}

// GetDeployment returns a deployment by ID.
func (s *DeployService) GetDeployment(ctx context.Context, userID, deployID uuid.UUID) (*domain.Deployment, error) {
	deploy, err := s.deployRepo.GetByID(ctx, deployID)
	if err != nil {
		return nil, appErr.NotFound("deployment")
	}
	if deploy.UserID != userID {
		return nil, appErr.ErrForbidden
	}
	return deploy, nil
}

// ListDeployments returns deployments for a user.
func (s *DeployService) ListDeployments(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*domain.Deployment, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.deployRepo.ListByUser(ctx, userID, offset, pageSize)
}

// RollbackDeployment rolls back to a previous version.
func (s *DeployService) RollbackDeployment(ctx context.Context, userID, deployID uuid.UUID) (*domain.Deployment, error) {
	deploy, err := s.deployRepo.GetByID(ctx, deployID)
	if err != nil {
		return nil, appErr.NotFound("deployment")
	}
	if deploy.UserID != userID {
		return nil, appErr.ErrForbidden
	}

	// Create a new deployment with the same config
	rollback := &domain.Deployment{
		ID:          uuid.New(),
		ProjectID:   deploy.ProjectID,
		UserID:      userID,
		Version:     deploy.Version + "-rollback",
		Environment: deploy.Environment,
		Platform:    deploy.Platform,
		Status:      domain.DeployStatusPending,
		Config:      deploy.Config,
		Metadata:    map[string]interface{}{"rollback_from": deploy.ID.String()},
	}

	if err := s.deployRepo.Create(ctx, rollback); err != nil {
		return nil, appErr.Wrap(err, "failed to create rollback")
	}

	go s.executeDeployment(context.Background(), rollback)

	return rollback, nil
}

// GetDeploymentLogs returns logs for a deployment.
func (s *DeployService) GetDeploymentLogs(ctx context.Context, userID, deployID uuid.UUID, lines int) (string, error) {
	deploy, err := s.deployRepo.GetByID(ctx, deployID)
	if err != nil {
		return "", appErr.NotFound("deployment")
	}
	if deploy.UserID != userID {
		return "", appErr.ErrForbidden
	}

	platform, err := s.platformReg.Get(string(deploy.Platform))
	if err != nil {
		return "", appErr.Wrap(err, "platform not found")
	}

	return platform.GetLogs(ctx, deploy.ID.String(), lines)
}
