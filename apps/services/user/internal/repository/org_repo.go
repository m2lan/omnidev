package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnidev/services/user/internal/domain"
)

// OrganizationRepository defines the interface for organization data access.
type OrganizationRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Organization, error)
	Update(ctx context.Context, id uuid.UUID, update *domain.OrganizationUpdate) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *domain.OrganizationFilter, offset, limit int) ([]*domain.Organization, int, error)
	SlugExists(ctx context.Context, slug string) (bool, error)

	// Members
	AddMember(ctx context.Context, member *domain.OrgMember) error
	GetMember(ctx context.Context, orgID, userID uuid.UUID) (*domain.OrgMember, error)
	ListMembers(ctx context.Context, orgID uuid.UUID, offset, limit int) ([]*domain.OrgMember, int, error)
	UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role domain.OrgMemberRole) error
	RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error
	CountMembers(ctx context.Context, orgID uuid.UUID) (int, error)
	IsMember(ctx context.Context, orgID, userID uuid.UUID) (bool, error)
}

type organizationRepository struct {
	pool *pgxpool.Pool
}

// NewOrganizationRepository creates a new organization repository.
func NewOrganizationRepository(pool *pgxpool.Pool) OrganizationRepository {
	return &organizationRepository{pool: pool}
}

func (r *organizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	query := `
		INSERT INTO organizations (id, name, slug, description, avatar_url, plan, settings, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		org.ID, org.Name, org.Slug, org.Description, org.AvatarURL, org.Plan, org.Settings, org.Metadata,
	).Scan(&org.CreatedAt, &org.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("slug already exists: %w", err)
		}
		return fmt.Errorf("failed to create organization: %w", err)
	}
	return nil
}

func (r *organizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	query := `
		SELECT id, name, slug, description, avatar_url, plan, settings, metadata, created_at, updated_at
		FROM organizations
		WHERE id = $1 AND deleted_at IS NULL`

	org := &domain.Organization{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Description, &org.AvatarURL,
		&org.Plan, &org.Settings, &org.Metadata, &org.CreatedAt, &org.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return org, nil
}

func (r *organizationRepository) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	query := `
		SELECT id, name, slug, description, avatar_url, plan, settings, metadata, created_at, updated_at
		FROM organizations
		WHERE slug = $1 AND deleted_at IS NULL`

	org := &domain.Organization{}
	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Description, &org.AvatarURL,
		&org.Plan, &org.Settings, &org.Metadata, &org.CreatedAt, &org.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return org, nil
}

func (r *organizationRepository) Update(ctx context.Context, id uuid.UUID, update *domain.OrganizationUpdate) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if update.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *update.Name)
		argIdx++
	}
	if update.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *update.Description)
		argIdx++
	}
	if update.AvatarURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("avatar_url = $%d", argIdx))
		args = append(args, *update.AvatarURL)
		argIdx++
	}
	if update.Settings != nil {
		setClauses = append(setClauses, fmt.Sprintf("settings = $%d", argIdx))
		args = append(args, update.Settings)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE organizations SET %s WHERE id = $%d AND deleted_at IS NULL",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("organization not found")
	}
	return nil
}

func (r *organizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE organizations SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("organization not found")
	}
	return nil
}

func (r *organizationRepository) List(ctx context.Context, filter *domain.OrganizationFilter, offset, limit int) ([]*domain.Organization, int, error) {
	whereClauses := []string{"o.deleted_at IS NULL"}
	args := []interface{}{}
	argIdx := 1

	if filter != nil && filter.UserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf(
			"EXISTS(SELECT 1 FROM org_members om WHERE om.org_id = o.id AND om.user_id = $%d)", argIdx))
		args = append(args, *filter.UserID)
		argIdx++
	}
	if filter != nil && filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(o.name ILIKE $%d OR o.slug ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	where := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM organizations o WHERE %s", where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count organizations: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT o.id, o.name, o.slug, o.description, o.avatar_url, o.plan, o.settings, o.metadata, o.created_at, o.updated_at
		FROM organizations o
		WHERE %s
		ORDER BY o.created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list organizations: %w", err)
	}
	defer rows.Close()

	orgs := make([]*domain.Organization, 0)
	for rows.Next() {
		org := &domain.Organization{}
		if err := rows.Scan(
			&org.ID, &org.Name, &org.Slug, &org.Description, &org.AvatarURL,
			&org.Plan, &org.Settings, &org.Metadata, &org.CreatedAt, &org.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan organization: %w", err)
		}
		orgs = append(orgs, org)
	}

	return orgs, total, nil
}

func (r *organizationRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM organizations WHERE slug = $1 AND deleted_at IS NULL)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, slug).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check slug existence: %w", err)
	}
	return exists, nil
}

// --- Members ---

func (r *organizationRepository) AddMember(ctx context.Context, member *domain.OrgMember) error {
	query := `
		INSERT INTO org_members (id, org_id, user_id, role, invited_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING joined_at, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		member.ID, member.OrgID, member.UserID, member.Role, member.InvitedBy,
	).Scan(&member.JoinedAt, &member.CreatedAt, &member.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("user is already a member")
		}
		return fmt.Errorf("failed to add member: %w", err)
	}
	return nil
}

func (r *organizationRepository) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*domain.OrgMember, error) {
	query := `
		SELECT id, org_id, user_id, role, invited_by, joined_at, created_at, updated_at
		FROM org_members
		WHERE org_id = $1 AND user_id = $2`

	member := &domain.OrgMember{}
	err := r.pool.QueryRow(ctx, query, orgID, userID).Scan(
		&member.ID, &member.OrgID, &member.UserID, &member.Role,
		&member.InvitedBy, &member.JoinedAt, &member.CreatedAt, &member.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("member not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	return member, nil
}

func (r *organizationRepository) ListMembers(ctx context.Context, orgID uuid.UUID, offset, limit int) ([]*domain.OrgMember, int, error) {
	countQuery := `SELECT COUNT(*) FROM org_members WHERE org_id = $1`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, orgID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count members: %w", err)
	}

	query := `
		SELECT id, org_id, user_id, role, invited_by, joined_at, created_at, updated_at
		FROM org_members
		WHERE org_id = $1
		ORDER BY joined_at ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list members: %w", err)
	}
	defer rows.Close()

	members := make([]*domain.OrgMember, 0)
	for rows.Next() {
		m := &domain.OrgMember{}
		if err := rows.Scan(
			&m.ID, &m.OrgID, &m.UserID, &m.Role,
			&m.InvitedBy, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, m)
	}

	return members, total, nil
}

func (r *organizationRepository) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role domain.OrgMemberRole) error {
	query := `UPDATE org_members SET role = $1 WHERE org_id = $2 AND user_id = $1`
	_, err := r.pool.Exec(ctx, query, role, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}
	return nil
}

func (r *organizationRepository) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM org_members WHERE org_id = $1 AND user_id = $2`, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

func (r *organizationRepository) CountMembers(ctx context.Context, orgID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM org_members WHERE org_id = $1`
	var count int
	if err := r.pool.QueryRow(ctx, query, orgID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count members: %w", err)
	}
	return count, nil
}

func (r *organizationRepository) IsMember(ctx context.Context, orgID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM org_members WHERE org_id = $1 AND user_id = $2)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, orgID, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}
	return exists, nil
}
