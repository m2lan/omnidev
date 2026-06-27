package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/cache"
	"github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/services/user/internal/domain"
	"github.com/omnidev/services/user/internal/repository"
)

// UserService handles user profile operations.
type UserService struct {
	userRepo repository.UserRepository
	cache    *cache.Redis
}

// NewUserService creates a new user service.
func NewUserService(userRepo repository.UserRepository, cache *cache.Redis) *UserService {
	return &UserService{
		userRepo: userRepo,
		cache:    cache,
	}
}

// GetProfile returns a user's profile, using cache when available.
func (s *UserService) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("user:profile:%s", userID.String())
	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil {
		var user domain.User
		if json.Unmarshal([]byte(cached), &user) == nil {
			return &user, nil
		}
	}

	// Fetch from database
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.NotFound("user")
	}

	// Cache for 5 minutes
	data, _ := json.Marshal(user)
	_ = s.cache.Set(ctx, cacheKey, data, 5*time.Minute)

	return user, nil
}

// UpdateProfile updates a user's profile.
func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, update *domain.UserUpdate) (*domain.User, error) {
	if err := s.userRepo.Update(ctx, userID, update); err != nil {
		return nil, errors.Wrap(err, "failed to update profile")
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:profile:%s", userID.String()))

	logger.Log.Info("Profile updated",
		zap.String("user_id", userID.String()),
	)

	return s.userRepo.GetByID(ctx, userID)
}

// GetUser returns a user by ID.
func (s *UserService) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.NotFound("user")
	}
	return user, nil
}

// ListUsers returns a paginated list of users.
func (s *UserService) ListUsers(ctx context.Context, filter *domain.UserFilter, page, pageSize int) ([]*domain.User, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	return s.userRepo.List(ctx, filter, offset, pageSize)
}

// UpdateUserStatus updates a user's status (admin only).
func (s *UserService) UpdateUserStatus(ctx context.Context, userID uuid.UUID, status domain.UserStatus) error {
	if err := s.userRepo.UpdateStatus(ctx, userID, status); err != nil {
		return errors.Wrap(err, "failed to update user status")
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:profile:%s", userID.String()))

	logger.Log.Info("User status updated",
		zap.String("user_id", userID.String()),
		zap.String("status", string(status)),
	)

	return nil
}
