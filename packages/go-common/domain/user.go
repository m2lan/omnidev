// Package domain defines the core business entities for the User Service.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserStatus represents the status of a user account.
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusDeleted   UserStatus = "deleted"
)

// UserRole represents the role of a user.
type UserRole string

const (
	UserRoleUser       UserRole = "user"
	UserRoleAdmin      UserRole = "admin"
	UserRoleSuperAdmin UserRole = "super_admin"
)

// User represents a user account.
type User struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	Email         string                 `json:"email" db:"email"`
	EmailVerified bool                   `json:"email_verified" db:"email_verified"`
	PasswordHash  string                 `json:"-" db:"password_hash"`
	Nickname      string                 `json:"nickname" db:"nickname"`
	AvatarURL     *string                `json:"avatar_url,omitempty" db:"avatar_url"`
	Bio           *string                `json:"bio,omitempty" db:"bio"`
	Role          UserRole               `json:"role" db:"role"`
	Status        UserStatus             `json:"status" db:"status"`
	LastLoginAt   *time.Time             `json:"last_login_at,omitempty" db:"last_login_at"`
	LastLoginIP   *string                `json:"-" db:"last_login_ip"`
	Settings      map[string]interface{} `json:"settings" db:"settings"`
	Metadata      map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt     *time.Time             `json:"-" db:"deleted_at"`
}

// IsDeleted returns true if the user has been soft-deleted.
func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

// IsActive returns true if the user account is active.
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive && !u.IsDeleted()
}

// IsAdmin returns true if the user has admin privileges.
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin || u.Role == UserRoleSuperAdmin
}

// UserFilter defines filters for listing users.
type UserFilter struct {
	Status *UserStatus
	Role   *UserRole
	Search string // Search by email or nickname
}

// UserUpdate defines fields that can be updated.
type UserUpdate struct {
	Nickname  *string                `json:"nickname,omitempty"`
	AvatarURL *string                `json:"avatar_url,omitempty"`
	Bio       *string                `json:"bio,omitempty"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
}
