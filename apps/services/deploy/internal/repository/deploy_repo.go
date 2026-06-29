// Package repository provides data access implementations for the Deploy Service.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/deploy/internal/domain"
)

// DeployRepository defines the interface for deployment data access.
type DeployRepository interface {
	Create(ctx context.Context, deploy *domain.Deployment) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Deployment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DeployStatus, errMsg *string) error
	UpdateBuildInfo(ctx context.Context, id uuid.UUID, imageTag, buildLogs string) error
	UpdateURL(ctx context.Context, id uuid.UUID, url string) error
	ListByProject(ctx context.Context, projectID uuid.UUID, offset, limit int) ([]*domain.Deployment, int, error)
	ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Deployment, int, error)
}

type deployRepository struct {
	pool *pgxpool.Pool
}

func NewDeployRepository(pool *pgxpool.Pool) DeployRepository {
	return &deployRepository{pool: pool}
}

func (r *deployRepository) Create(ctx context.Context, deploy *domain.Deployment) error {
	query := `
		INSERT INTO deployments (id, project_id, user_id, version, environment, platform, status, config, domain, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		deploy.ID, deploy.ProjectID, deploy.UserID, deploy.Version,
		deploy.Environment, deploy.Platform, deploy.Status,
		deploy.Config, deploy.Domain, deploy.Metadata,
	).Scan(&deploy.CreatedAt, &deploy.UpdatedAt)
}

func (r *deployRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Deployment, error) {
	query := `
		SELECT id, project_id, user_id, version, environment, platform, status, config,
		       domain, url, image_tag, build_logs, error, resource_usage,
		       started_at, completed_at, metadata, created_at, updated_at
		FROM deployments WHERE id = $1`

	deploy := &domain.Deployment{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&deploy.ID, &deploy.ProjectID, &deploy.UserID, &deploy.Version,
		&deploy.Environment, &deploy.Platform, &deploy.Status,
		&deploy.Config, &deploy.Domain, &deploy.URL, &deploy.ImageTag,
		&deploy.BuildLogs, &deploy.Error, &deploy.ResourceUsage,
		&deploy.StartedAt, &deploy.CompletedAt, &deploy.Metadata,
		&deploy.CreatedAt, &deploy.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("deployment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}
	return deploy, nil
}

func (r *deployRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DeployStatus, errMsg *string) error {
	now := time.Now()
	var completedAt *time.Time
	var startedAt *time.Time
	if status == domain.DeployStatusBuilding || status == domain.DeployStatusDeploying {
		startedAt = &now
	}
	if status == domain.DeployStatusRunning || status == domain.DeployStatusFailed || status == domain.DeployStatusStopped {
		completedAt = &now
	}

	_, err := r.pool.Exec(ctx,
		`UPDATE deployments SET status = $1, error = $2, started_at = COALESCE(started_at, $3), completed_at = $4 WHERE id = $5`,
		status, errMsg, startedAt, completedAt, id)
	return err
}

func (r *deployRepository) UpdateBuildInfo(ctx context.Context, id uuid.UUID, imageTag, buildLogs string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE deployments SET image_tag = $1, build_logs = $2 WHERE id = $3`,
		imageTag, buildLogs, id)
	return err
}

func (r *deployRepository) UpdateURL(ctx context.Context, id uuid.UUID, url string) error {
	_, err := r.pool.Exec(ctx, `UPDATE deployments SET url = $1 WHERE id = $2`, url, id)
	return err
}

func (r *deployRepository) ListByProject(ctx context.Context, projectID uuid.UUID, offset, limit int) ([]*domain.Deployment, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM deployments WHERE project_id = $1`, projectID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, project_id, user_id, version, environment, platform, status, config,
		       domain, url, image_tag, build_logs, error, resource_usage,
		       started_at, completed_at, metadata, created_at, updated_at
		FROM deployments WHERE project_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, projectID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	deploys := make([]*domain.Deployment, 0)
	for rows.Next() {
		d := &domain.Deployment{}
		if err := rows.Scan(
			&d.ID, &d.ProjectID, &d.UserID, &d.Version,
			&d.Environment, &d.Platform, &d.Status,
			&d.Config, &d.Domain, &d.URL, &d.ImageTag,
			&d.BuildLogs, &d.Error, &d.ResourceUsage,
			&d.StartedAt, &d.CompletedAt, &d.Metadata,
			&d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		deploys = append(deploys, d)
	}

	return deploys, total, nil
}

func (r *deployRepository) ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Deployment, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM deployments WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, project_id, user_id, version, environment, platform, status, config,
		       domain, url, image_tag, build_logs, error, resource_usage,
		       started_at, completed_at, metadata, created_at, updated_at
		FROM deployments WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	deploys := make([]*domain.Deployment, 0)
	for rows.Next() {
		d := &domain.Deployment{}
		if err := rows.Scan(
			&d.ID, &d.ProjectID, &d.UserID, &d.Version,
			&d.Environment, &d.Platform, &d.Status,
			&d.Config, &d.Domain, &d.URL, &d.ImageTag,
			&d.BuildLogs, &d.Error, &d.ResourceUsage,
			&d.StartedAt, &d.CompletedAt, &d.Metadata,
			&d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		deploys = append(deploys, d)
	}

	return deploys, total, nil
}
