// Package service contains the business logic for the User Service.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/omnidev/go-common/auth"
	"github.com/omnidev/go-common/cache"
	"github.com/omnidev/go-common/config"
	"github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/services/user/internal/domain"
	"github.com/omnidev/services/user/internal/repository"
)

// AuthService handles authentication and authorization operations.
type AuthService struct {
	userRepo   repository.UserRepository
	oauthRepo  repository.OAuthRepository
	apiKeyRepo repository.APIKeyRepository
	jwtManager *auth.JWTManager
	cache      *cache.Redis
	config     *config.Config
}

// NewAuthService creates a new auth service.
func NewAuthService(
	userRepo repository.UserRepository,
	oauthRepo repository.OAuthRepository,
	apiKeyRepo repository.APIKeyRepository,
	jwtManager *auth.JWTManager,
	cache *cache.Redis,
	config *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		oauthRepo:  oauthRepo,
		apiKeyRepo: apiKeyRepo,
		jwtManager: jwtManager,
		cache:      cache,
		config:     config,
	}
}

// RegisterInput defines the input for user registration.
type RegisterInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,password"`
	Nickname string `json:"nickname" validate:"required,min=2,max=50"`
}

// RegisterOutput defines the output of user registration.
type RegisterOutput struct {
	User         *domain.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

// Register creates a new user account.
func (s *AuthService) Register(ctx context.Context, input *RegisterInput) (*RegisterOutput, error) {
	// Check if email already exists
	exists, err := s.userRepo.EmailExists(ctx, input.Email)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check email")
	}
	if exists {
		return nil, errors.Conflict("email already registered")
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to hash password")
	}

	// Create user
	user := &domain.User{
		ID:           uuid.New(),
		Email:        input.Email,
		PasswordHash: hashedPassword,
		Nickname:     input.Nickname,
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
		Settings:     map[string]interface{}{},
		Metadata:     map[string]interface{}{},
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(err, "failed to create user")
	}

	// Generate tokens
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, string(user.Role), "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate tokens")
	}

	// Cache user session
	s.cacheSession(ctx, user.ID, tokenPair.RefreshToken)

	logger.Log.Info("User registered",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
	)

	return &RegisterOutput{
		User:         user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}

// LoginInput defines the input for user login.
type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginOutput defines the output of user login.
type LoginOutput struct {
	User         *domain.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

// Login authenticates a user and returns tokens.
func (s *AuthService) Login(ctx context.Context, input *LoginInput, ip string) (*LoginOutput, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, errors.Validation("invalid email or password")
	}

	// Check if user is active
	if !user.IsActive() {
		return nil, errors.Validation("account is suspended or deleted")
	}

	// Verify password
	if !auth.CheckPassword(input.Password, user.PasswordHash) {
		return nil, errors.Validation("invalid email or password")
	}

	// Generate tokens
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, string(user.Role), "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate tokens")
	}

	// Update last login
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID, ip)

	// Cache user session
	s.cacheSession(ctx, user.ID, tokenPair.RefreshToken)

	logger.Log.Info("User logged in",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
		zap.String("ip", ip),
	)

	return &LoginOutput{
		User:         user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

// RefreshTokenInput defines the input for token refresh.
type RefreshTokenInput struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshTokenOutput defines the output of token refresh.
type RefreshTokenOutput struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// RefreshToken refreshes an access token using a refresh token.
func (s *AuthService) RefreshToken(ctx context.Context, input *RefreshTokenInput) (*RefreshTokenOutput, error) {
	// Validate refresh token
	claims, err := s.jwtManager.ValidateToken(input.RefreshToken)
	if err != nil {
		return nil, errors.Validation("invalid refresh token")
	}

	if claims.Type != auth.TokenTypeRefresh {
		return nil, errors.Validation("invalid token type")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, errors.NotFound("user")
	}

	if !user.IsActive() {
		return nil, errors.Validation("account is suspended")
	}

	// Check if refresh token is blacklisted
	blacklisted, _ := s.cache.Exists(ctx, fmt.Sprintf("token:blacklist:%s", input.RefreshToken))
	if blacklisted {
		return nil, errors.Validation("token has been revoked")
	}

	// Generate new tokens
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, string(user.Role), claims.OrgID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate tokens")
	}

	// Blacklist old refresh token
	_ = s.cache.Set(ctx, fmt.Sprintf("token:blacklist:%s", input.RefreshToken), true, 7*24*time.Hour)

	// Cache new session
	s.cacheSession(ctx, user.ID, tokenPair.RefreshToken)

	return &RefreshTokenOutput{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

// CreateAPIKeyInput defines the input for creating an API key.
type CreateAPIKeyInput struct {
	Name      string     `json:"name" validate:"required,min=1,max=100"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// CreateAPIKeyOutput defines the output of creating an API key.
type CreateAPIKeyOutput struct {
	APIKey *domain.APIKey `json:"api_key"`
	Key    string         `json:"key"`
}

// CreateAPIKey creates a new API key for a user.
func (s *AuthService) CreateAPIKey(ctx context.Context, userID uuid.UUID, input *CreateAPIKeyInput) (*CreateAPIKeyOutput, error) {
	// Generate API key
	key, hash, err := auth.GenerateAPIKey("sk-")
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate API key")
	}

	apiKey := &domain.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      input.Name,
		KeyHash:   hash,
		KeyPrefix: key[:10] + "...",
		Scopes:    input.Scopes,
		ExpiresAt: input.ExpiresAt,
		Status:    domain.APIKeyStatusActive,
	}

	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, errors.Wrap(err, "failed to create API key")
	}

	logger.Log.Info("API key created",
		zap.String("user_id", userID.String()),
		zap.String("key_id", apiKey.ID.String()),
	)

	return &CreateAPIKeyOutput{
		APIKey: apiKey,
		Key:    key,
	}, nil
}

// ListAPIKeys returns all API keys for a user.
func (s *AuthService) ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]*domain.APIKey, error) {
	return s.apiKeyRepo.ListByUserID(ctx, userID)
}

// RevokeAPIKey revokes an API key.
func (s *AuthService) RevokeAPIKey(ctx context.Context, userID, keyID uuid.UUID) error {
	key, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return errors.NotFound("API key")
	}

	if key.UserID != userID {
		return errors.ErrForbidden
	}

	return s.apiKeyRepo.Revoke(ctx, keyID)
}

// ValidateAPIKey validates an API key and returns the associated user.
func (s *AuthService) ValidateAPIKey(ctx context.Context, key string) (*domain.User, *domain.APIKey, error) {
	// Hash the key to look up
	hash, err := auth.HashAPIKey(key)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to hash API key")
	}

	apiKey, err := s.apiKeyRepo.GetByHash(ctx, hash)
	if err != nil {
		return nil, nil, errors.Validation("invalid API key")
	}

	if !apiKey.IsValid() {
		return nil, nil, errors.Validation("API key is expired or revoked")
	}

	user, err := s.userRepo.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get user")
	}

	return user, apiKey, nil
}

// cacheSession caches the user session in Redis.
func (s *AuthService) cacheSession(ctx context.Context, userID uuid.UUID, refreshToken string) {
	sessionData, _ := json.Marshal(map[string]interface{}{
		"user_id":       userID.String(),
		"refresh_token": refreshToken,
		"created_at":    time.Now().Unix(),
	})
	_ = s.cache.Set(ctx, fmt.Sprintf("session:%s", userID.String()), sessionData, 7*24*time.Hour)
}
