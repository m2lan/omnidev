package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/services/user/internal/domain"
	"github.com/omnidev/services/user/internal/repository"
)

// OrganizationService handles organization operations.
type OrganizationService struct {
	orgRepo  repository.OrganizationRepository
	userRepo repository.UserRepository
}

// NewOrganizationService creates a new organization service.
func NewOrganizationService(orgRepo repository.OrganizationRepository, userRepo repository.UserRepository) *OrganizationService {
	return &OrganizationService{
		orgRepo:  orgRepo,
		userRepo: userRepo,
	}
}

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// CreateOrgInput defines the input for creating an organization.
type CreateOrgInput struct {
	Name        string  `json:"name" validate:"required,min=2,max=100"`
	Slug        string  `json:"slug" validate:"required,min=2,max=100"`
	Description *string `json:"description"`
}

// CreateOrganization creates a new organization and adds the creator as owner.
func (s *OrganizationService) CreateOrganization(ctx context.Context, userID uuid.UUID, input *CreateOrgInput) (*domain.Organization, error) {
	// Validate slug format
	if !slugRegex.MatchString(input.Slug) {
		return nil, errors.Validation("slug must contain only lowercase letters, numbers, and hyphens")
	}

	// Check slug uniqueness
	exists, err := s.orgRepo.SlugExists(ctx, input.Slug)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check slug")
	}
	if exists {
		return nil, errors.Conflict("slug already taken")
	}

	// Create organization
	org := &domain.Organization{
		ID:          uuid.New(),
		Name:        input.Name,
		Slug:        input.Slug,
		Description: input.Description,
		Plan:        domain.OrgPlanFree,
		Settings:    map[string]interface{}{},
		Metadata:    map[string]interface{}{},
	}

	if err := s.orgRepo.Create(ctx, org); err != nil {
		return nil, errors.Wrap(err, "failed to create organization")
	}

	// Add creator as owner
	member := &domain.OrgMember{
		ID:     uuid.New(),
		OrgID:  org.ID,
		UserID: userID,
		Role:   domain.OrgMemberRoleOwner,
	}

	if err := s.orgRepo.AddMember(ctx, member); err != nil {
		return nil, errors.Wrap(err, "failed to add owner to organization")
	}

	logger.Log.Info("Organization created",
		zap.String("org_id", org.ID.String()),
		zap.String("slug", org.Slug),
		zap.String("owner_id", userID.String()),
	)

	return org, nil
}

// GetOrganization returns an organization by ID.
func (s *OrganizationService) GetOrganization(ctx context.Context, orgID uuid.UUID) (*domain.Organization, error) {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, errors.NotFound("organization")
	}

	// Get member count
	count, _ := s.orgRepo.CountMembers(ctx, orgID)
	org.MemberCount = count

	return org, nil
}

// UpdateOrganization updates an organization.
func (s *OrganizationService) UpdateOrganization(ctx context.Context, orgID uuid.UUID, update *domain.OrganizationUpdate) (*domain.Organization, error) {
	if err := s.orgRepo.Update(ctx, orgID, update); err != nil {
		return nil, errors.Wrap(err, "failed to update organization")
	}
	return s.orgRepo.GetByID(ctx, orgID)
}

// ListOrganizations returns organizations the user belongs to.
func (s *OrganizationService) ListOrganizations(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*domain.Organization, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	filter := &domain.OrganizationFilter{
		UserID: &userID,
	}

	return s.orgRepo.List(ctx, filter, offset, pageSize)
}

// InviteMemberInput defines the input for inviting a member.
type InviteMemberInput struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=admin member viewer"`
}

// InviteMember invites a user to an organization by email.
func (s *OrganizationService) InviteMember(ctx context.Context, orgID uuid.UUID, inviterID uuid.UUID, input *InviteMemberInput) (*domain.OrgMember, error) {
	// Check if inviter has permission
	inviterMember, err := s.orgRepo.GetMember(ctx, orgID, inviterID)
	if err != nil {
		return nil, errors.ErrForbidden
	}
	if inviterMember.Role != domain.OrgMemberRoleOwner && inviterMember.Role != domain.OrgMemberRoleAdmin {
		return nil, errors.ErrForbidden
	}

	// Find user by email
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, errors.NotFound("user with this email")
	}

	// Check if already a member
	isMember, _ := s.orgRepo.IsMember(ctx, orgID, user.ID)
	if isMember {
		return nil, errors.Conflict("user is already a member")
	}

	// Add member
	member := &domain.OrgMember{
		ID:        uuid.New(),
		OrgID:     orgID,
		UserID:    user.ID,
		Role:      domain.OrgMemberRole(input.Role),
		InvitedBy: &inviterID,
	}

	if err := s.orgRepo.AddMember(ctx, member); err != nil {
		return nil, errors.Wrap(err, "failed to add member")
	}

	logger.Log.Info("Member invited to organization",
		zap.String("org_id", orgID.String()),
		zap.String("user_id", user.ID.String()),
		zap.String("inviter_id", inviterID.String()),
	)

	return member, nil
}

// ListMembers returns members of an organization.
func (s *OrganizationService) ListMembers(ctx context.Context, orgID uuid.UUID, page, pageSize int) ([]*domain.OrgMember, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	return s.orgRepo.ListMembers(ctx, orgID, offset, pageSize)
}

// UpdateMemberRole updates a member's role.
func (s *OrganizationService) UpdateMemberRole(ctx context.Context, orgID, actorID, targetUserID uuid.UUID, role domain.OrgMemberRole) error {
	// Check actor permission
	actorMember, err := s.orgRepo.GetMember(ctx, orgID, actorID)
	if err != nil {
		return errors.ErrForbidden
	}
	if actorMember.Role != domain.OrgMemberRoleOwner {
		return errors.ErrForbidden
	}

	// Cannot change own role
	if actorID == targetUserID {
		return errors.Validation("cannot change your own role")
	}

	return s.orgRepo.UpdateMemberRole(ctx, orgID, targetUserID, role)
}

// RemoveMember removes a member from an organization.
func (s *OrganizationService) RemoveMember(ctx context.Context, orgID, actorID, targetUserID uuid.UUID) error {
	// Check actor permission
	actorMember, err := s.orgRepo.GetMember(ctx, orgID, actorID)
	if err != nil {
		return errors.ErrForbidden
	}

	// Only owner/admin can remove others; anyone can remove themselves
	if actorID != targetUserID &&
		actorMember.Role != domain.OrgMemberRoleOwner &&
		actorMember.Role != domain.OrgMemberRoleAdmin {
		return errors.ErrForbidden
	}

	// Cannot remove the owner
	targetMember, err := s.orgRepo.GetMember(ctx, orgID, targetUserID)
	if err != nil {
		return errors.NotFound("member")
	}
	if targetMember.Role == domain.OrgMemberRoleOwner && actorID != targetUserID {
		return errors.Validation("cannot remove the organization owner")
	}

	return s.orgRepo.RemoveMember(ctx, orgID, targetUserID)
}

// IsMember checks if a user is a member of an organization.
func (s *OrganizationService) IsMember(ctx context.Context, orgID, userID uuid.UUID) (bool, error) {
	return s.orgRepo.IsMember(ctx, orgID, userID)
}

// sanitizeSlug converts a name to a valid slug.
func sanitizeSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	// Remove invalid characters
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	slug = reg.ReplaceAllString(slug, "")
	// Remove consecutive hyphens
	reg2 := regexp.MustCompile(`-+`)
	slug = reg2.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 100 {
		slug = slug[:100]
	}
	return slug
}

// generateSlug generates a unique slug from a name.
func (s *OrganizationService) generateSlug(ctx context.Context, name string) string {
	base := sanitizeSlug(name)
	if base == "" {
		base = "org"
	}

	slug := base
	counter := 1
	for {
		exists, _ := s.orgRepo.SlugExists(ctx, slug)
		if !exists {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", base, counter)
		counter++
	}
}
