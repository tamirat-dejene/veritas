package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/infrastructure/security"
)

// MockUserRepository is a manual mock for UserRepository
type MockUserRepository struct {
	users map[string]*domain.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{users: make(map[string]*domain.User)}
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	m.users[user.Email] = user
	return nil
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if user, ok := m.users[email]; ok {
		return user, nil
	}
	return nil, nil // Not found
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, nil
}

func TestAuthService_Register(t *testing.T) {
	mockRepo := NewMockUserRepository()
	tokenService := security.NewTokenService("secret")
	service := NewAuthService(mockRepo, tokenService)

	email := "test@example.com"
	password := "password123"
	role := "student"

	// Case 1: Successful Registration
	user, err := service.Register(context.Background(), email, password, role, "Test", "User")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Email != email {
		t.Errorf("expected email %s, got %s", email, user.Email)
	}

	// Case 2: Duplicate Email
	_, err = service.Register(context.Background(), email, password, role, "Test", "User")
	if err == nil {
		t.Error("expected error for duplicate email, got nil")
	}
}

func TestAuthService_Register_Validation(t *testing.T) {
	mockRepo := NewMockUserRepository()
	tokenService := security.NewTokenService("secret")
	service := NewAuthService(mockRepo, tokenService)

	cases := []struct {
		name      string
		email     string
		password  string
		role      string
		firstName string
		lastName  string
		wantErr   error
	}{
		{
			name:      "invalid email",
			email:     "invalid-email",
			password:  "password123",
			role:      "student",
			firstName: "Test",
			lastName:  "User",
			wantErr:   ErrInvalidEmail,
		},
		{
			name:      "short password",
			email:     "shortpass@example.com",
			password:  "short",
			role:      "student",
			firstName: "Test",
			lastName:  "User",
			wantErr:   ErrInvalidPassword,
		},
		{
			name:      "invalid role",
			email:     "role@example.com",
			password:  "password123",
			role:      "invalid",
			firstName: "Test",
			lastName:  "User",
			wantErr:   ErrInvalidRole,
		},
		{
			name:      "missing first name",
			email:     "first@example.com",
			password:  "password123",
			role:      "student",
			firstName: " ",
			lastName:  "User",
			wantErr:   ErrMissingName,
		},
		{
			name:      "missing last name",
			email:     "last@example.com",
			password:  "password123",
			role:      "student",
			firstName: "Test",
			lastName:  " ",
			wantErr:   ErrMissingName,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.Register(context.Background(), tc.email, tc.password, tc.role, tc.firstName, tc.lastName)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestAuthService_Register_NormalizesEmail(t *testing.T) {
	mockRepo := NewMockUserRepository()
	tokenService := security.NewTokenService("secret")
	service := NewAuthService(mockRepo, tokenService)

	user, err := service.Register(context.Background(), " Test@Example.com ", "password123", "student", "Test", "User")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Email != "test@example.com" {
		t.Fatalf("expected normalized email, got %s", user.Email)
	}
}

func TestAuthService_Login(t *testing.T) {
	mockRepo := NewMockUserRepository()
	tokenService := security.NewTokenService("secret")
	service := NewAuthService(mockRepo, tokenService)

	email := "test@example.com"
	password := "password123"
	hashedPassword, _ := security.HashPassword(password)

	// Create user manually in mock
	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         domain.RoleStudent,
	}
	mockRepo.Create(context.Background(), user)

	// Case 1: Successful Login
	token, err := service.Login(context.Background(), email, password)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Error("expected token, got empty string")
	}

	// Case 2: Invalid Password
	_, err = service.Login(context.Background(), email, "wrongpassword")
	if err == nil {
		t.Error("expected error for invalid password, got nil")
	}

	// Case 3: Invalid Email
	_, err = service.Login(context.Background(), "unknown@example.com", password)
	if err == nil {
		t.Error("expected error for unknown email, got nil")
	}
}
