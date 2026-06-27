package domain

import (
	"time"

	"github.com/google/uuid"
)

// OrgPlan represents the subscription plan of an organization.
type OrgPlan string

const (
	OrgPlanFree       OrgPlan = "free"
	OrgPlanPro        OrgPlan = "pro"
	OrgPlanTeam       OrgPlan = "team"
	OrgPlanEnterprise OrgPlan = "enterprise"
)

// OrgMemberRole represents the role of a member in an organization.
type OrgMemberRole string

const (
	OrgMemberRoleOwner  OrgMemberRole = "owner"
	OrgMemberRoleAdmin  OrgMemberRole = "admin"
	OrgMemberRoleMember OrgMemberRole = "member"
	OrgMemberRoleViewer OrgMemberRole = "viewer"
)

// Organization represents a team or company.
type Organization struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Slug        string                 `json:"slug" db:"slug"`
	Description *string                `json:"description,omitempty" db:"description"`
	AvatarURL   *string                `json:"avatar_url,omitempty" db:"avatar_url"`
	Plan        OrgPlan                `json:"plan" db:"plan"`
	Settings    map[string]interface{} `json:"settings" db:"settings"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
	MemberCount int                    `json:"member_count,omitempty"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time             `json:"-" db:"deleted_at"`
}

// OrgMember represents a user's membership in an organization.
type OrgMember struct {
	ID       uuid.UUID     `json:"id" db:"id"`
	OrgID    uuid.UUID     `json:"org_id" db:"org_id"`
	UserID   uuid.UUID     `json:"user_id" db:"user_id"`
	Role     OrgMemberRole `json:"role" db:"role"`
	InvitedBy *uuid.UUID   `json:"invited_by,omitempty" db:"invited_by"`
	JoinedAt time.Time     `json:"joined_at" db:"joined_at"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`

	// Joined fields
	User *User `json:"user,omitempty"`
}

// OrganizationFilter defines filters for listing organizations.
type OrganizationFilter struct {
	UserID *uuid.UUID // Organizations the user belongs to
	Search string
}

// OrganizationUpdate defines fields that can be updated.
type OrganizationUpdate struct {
	Name        *string
	Description *string
	AvatarURL   *string
	Settings    map[string]interface{}
}
