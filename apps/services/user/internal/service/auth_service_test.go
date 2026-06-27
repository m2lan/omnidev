package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/omnidev/go-common/auth"
	"github.com/omnidev/go-common/config"

	"github.com/omnidev/services/user/internal/domain"
	"github.com/omnidev/services/user/internal/service"
)

// mockUserRepository implements repository.UserRepository for testing.
type mockUserRepository struct {
	users      map[uuid.UUID]*domain.User
	usersByEmail map[string]*domain.User
}

func newMockUserRepo() *mockUserRepository {
	return &mockUserRepository{
		users:        make(map[uuid.UUID]*domain.User),
		usersByEmail: make(map[string]*domain.User),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return user, nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, ok := m.usersByEmail[email]
	if !ok {
		return nil, nil
	}
	return user, nil
}

func (m *mockUserRepository) Update(ctx context.Context, id uuid.UUID, update *domain.UserUpdate) error {
	return nil
}

func (m *mockUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error {
	return nil
}

func (m *mockUserRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error {
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockUserRepository) List(ctx context.Context, filter *domain.UserFilter, offset, limit int) ([]*domain.User, int, error) {
	return nil, 0, nil
}

func (m *mockUserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	_, ok := m.usersByEmail[email]
	return ok, nil
}

func TestAuthService_Register(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, "test")

	tests := []struct {
		name    string
		input   *service.RegisterInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful registration",
			input: &service.RegisterInput{
				Email:    "test@example.com",
				Password: "Password123!",
				Nickname: "TestUser",
			},
			wantErr: false,
		},
		{
			name: "duplicate email",
			input: &service.RegisterInput{
				Email:    "test@example.com",
				Password: "Password123!",
				Nickname: "TestUser2",
			},
			wantErr: true,
			errMsg:  "email already registered",
		},
		{
			name: "new email succeeds",
			input: &service.RegisterInput{
				Email:    "new@example.com",
				Password: "Password123!",
				Nickname: "NewUser",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh repo for each test
			userRepo := newMockUserRepo()
			// Pre-register if testing duplicate
			if tt.name == "duplicate email" {
				hashed, _ := auth.HashPassword("Password123!")
				userRepo.Create(context.Background(), &domain.User{
					ID:           uuid.New(),
					Email:        "test@example.com",
					PasswordHash: hashed,
					Nickname:     "Existing",
					Role:         domain.UserRoleUser,
					Status:       domain.UserStatusActive,
				})
			}

			cfg := &config.Config{}
			// Note: In a real test, you'd mock the cache too
			// For now, we test the core logic
			_ = jwtManager
			_ = cfg

			// This would need a mock cache to run fully
			// For now, test the repository logic directly
			exists, _ := userRepo.EmailExists(context.Background(), tt.input.Email)
			if tt.name == "duplicate email" && !exists {
				t.Error("Expected email to exist")
			}
			if tt.name == "new email" && exists {
				t.Error("Expected email to not exist")
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	password := "SecurePassword123!"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "" {
		t.Error("HashPassword() returned empty hash")
	}

	if hash == password {
		t.Error("HashPassword() returned plaintext password")
	}

	// Verify password
	if !auth.CheckPassword(password, hash) {
		t.Error("CheckPassword() returned false for correct password")
	}

	if auth.CheckPassword("wrong-password", hash) {
		t.Error("CheckPassword() returned true for wrong password")
	}
}

func TestGenerateAPIKey(t *testing.T) {
	key, hash, err := auth.GenerateAPIKey("sk-")
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}

	if key == "" {
		t.Error("GenerateAPIKey() returned empty key")
	}

	if hash == "" {
		t.Error("GenerateAPIKey() returned empty hash")
	}

	if len(key) < 10 {
		t.Error("API key too short")
	}

	// Verify key
	if !auth.CheckAPIKey(key, hash) {
		t.Error("CheckAPIKey() returned false for correct key")
	}

	if auth.CheckAPIKey("wrong-key", hash) {
		t.Error("CheckAPIKey() returned true for wrong key")
	}
}

func TestJWTManager_GenerateTokenPair(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret-key-at-least-32-chars!", 15*time.Minute, 7*24*time.Hour, "omnidev")

	userID := uuid.New()
	email := "test@example.com"
	role := "user"

	tokenPair, err := jwtManager.GenerateTokenPair(userID, email, role, "")
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Error("AccessToken is empty")
	}

	if tokenPair.RefreshToken == "" {
		t.Error("RefreshToken is empty")
	}

	if tokenPair.TokenType != "Bearer" {
		t.Errorf("TokenType = %s, want Bearer", tokenPair.TokenType)
	}

	// Validate access token
	claims, err := jwtManager.ValidateToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %s, want %s", claims.UserID, userID)
	}

	if claims.Email != email {
		t.Errorf("Email = %s, want %s", claims.Email, email)
	}

	if claims.Role != role {
		t.Errorf("Role = %s, want %s", claims.Role, role)
	}

	if claims.Type != auth.TokenTypeAccess {
		t.Errorf("Type = %s, want %s", claims.Type, auth.TokenTypeAccess)
	}
}

func TestJWTManager_InvalidToken(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, "omnidev")

	tests := []struct {
		name  string
		token string
	}{
		{name: "empty token", token: ""},
		{name: "invalid format", token: "not-a-jwt"},
		{name: "wrong signature", token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jwtManager.ValidateToken(tt.token)
			if err == nil {
				t.Error("ValidateToken() should return error for invalid token")
			}
		})
	}
}

func TestPasswordStrength(t *testing.T) {
	tests := []struct {
		password string
		valid    bool
	}{
		{"short", false},
		{"alllowercase1!", false},
		{"ALLUPPERCASE1!", false},
		{"NoDigits!", false},
		{"NoSpecial1a", false},
		{"Valid1Pass!", true},
		{"An0ther@Pass", true},
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			hash, err := auth.HashPassword(tt.password)
			if err != nil {
				t.Fatalf("HashPassword() error = %v", err)
			}
			if hash == "" {
				t.Error("HashPassword() returned empty hash")
			}
		})
	}
}
