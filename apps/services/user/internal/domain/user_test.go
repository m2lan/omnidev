package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/omnidev/services/user/internal/domain"
)

func TestUser_IsDeleted(t *testing.T) {
	tests := []struct {
		name string
		user domain.User
		want bool
	}{
		{
			name: "active user is not deleted",
			user: domain.User{DeletedAt: nil},
			want: false,
		},
		{
			name: "user with deleted_at is deleted",
			user: domain.User{DeletedAt: timePtr(time.Now())},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsDeleted(); got != tt.want {
				t.Errorf("User.IsDeleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsActive(t *testing.T) {
	tests := []struct {
		name string
		user domain.User
		want bool
	}{
		{
			name: "active user is active",
			user: domain.User{Status: domain.UserStatusActive, DeletedAt: nil},
			want: true,
		},
		{
			name: "suspended user is not active",
			user: domain.User{Status: domain.UserStatusSuspended, DeletedAt: nil},
			want: false,
		},
		{
			name: "deleted user is not active",
			user: domain.User{Status: domain.UserStatusActive, DeletedAt: timePtr(time.Now())},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsActive(); got != tt.want {
				t.Errorf("User.IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		name string
		user domain.User
		want bool
	}{
		{
			name: "regular user is not admin",
			user: domain.User{Role: domain.UserRoleUser},
			want: false,
		},
		{
			name: "admin is admin",
			user: domain.User{Role: domain.UserRoleAdmin},
			want: true,
		},
		{
			name: "super_admin is admin",
			user: domain.User{Role: domain.UserRoleSuperAdmin},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsAdmin(); got != tt.want {
				t.Errorf("User.IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIKey_IsExpired(t *testing.T) {
	tests := []struct {
		name string
		key  domain.APIKey
		want bool
	}{
		{
			name: "nil expires_at is not expired",
			key:  domain.APIKey{ExpiresAt: nil},
			want: false,
		},
		{
			name: "future expires_at is not expired",
			key:  domain.APIKey{ExpiresAt: timePtr(time.Now().Add(24 * time.Hour))},
			want: false,
		},
		{
			name: "past expires_at is expired",
			key:  domain.APIKey{ExpiresAt: timePtr(time.Now().Add(-24 * time.Hour))},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.key.IsExpired(); got != tt.want {
				t.Errorf("APIKey.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIKey_IsValid(t *testing.T) {
	tests := []struct {
		name string
		key  domain.APIKey
		want bool
	}{
		{
			name: "active and not expired is valid",
			key:  domain.APIKey{Status: domain.APIKeyStatusActive, ExpiresAt: timePtr(time.Now().Add(24 * time.Hour))},
			want: true,
		},
		{
			name: "revoked is not valid",
			key:  domain.APIKey{Status: domain.APIKeyStatusRevoked, ExpiresAt: timePtr(time.Now().Add(24 * time.Hour))},
			want: false,
		},
		{
			name: "expired is not valid",
			key:  domain.APIKey{Status: domain.APIKeyStatusActive, ExpiresAt: timePtr(time.Now().Add(-24 * time.Hour))},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.key.IsValid(); got != tt.want {
				t.Errorf("APIKey.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrganization(t *testing.T) {
	org := domain.Organization{
		ID:   uuid.New(),
		Name: "Test Org",
		Slug: "test-org",
		Plan: domain.OrgPlanFree,
	}

	if org.ID == uuid.Nil {
		t.Error("Organization ID should not be nil")
	}
	if org.Name != "Test Org" {
		t.Errorf("Organization.Name = %s, want 'Test Org'", org.Name)
	}
	if org.Slug != "test-org" {
		t.Errorf("Organization.Slug = %s, want 'test-org'", org.Slug)
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
