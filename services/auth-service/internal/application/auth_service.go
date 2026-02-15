package application

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/infrastructure/security"
)

type AuthService struct {
	userRepo     domain.UserRepository
	tokenService *security.TokenService
}

const minPasswordLength = 8

var (
	ErrUserExists         = errors.New("user with this email already exists")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidRole        = errors.New("invalid role")
	ErrMissingName        = errors.New("missing name")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrUserNotFound       = errors.New("user not found")
)

func NewAuthService(userRepo domain.UserRepository, tokenService *security.TokenService) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		tokenService: tokenService,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password, role, firstName, lastName string) (*domain.User, error) {
	normalizedEmail := normalizeEmail(email)
	if err := validateEmail(normalizedEmail); err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	normalizedFirstName := strings.TrimSpace(firstName)
	normalizedLastName := strings.TrimSpace(lastName)
	if normalizedFirstName == "" || normalizedLastName == "" {
		return nil, ErrMissingName
	}

	normalizedRole, err := parseRole(role)
	if err != nil {
		return nil, err
	}

	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserExists
	}

	// Hash password
	hashedPassword, err := security.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &domain.User{
		ID:           uuid.New(),
		Email:        normalizedEmail,
		PasswordHash: hashedPassword,
		Role:         normalizedRole,
		FirstName:    normalizedFirstName,
		LastName:     normalizedLastName,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	normalizedEmail := normalizeEmail(email)
	if err := validateEmail(normalizedEmail); err != nil {
		return "", err
	}
	if err := validatePassword(password); err != nil {
		return "", err
	}

	user, err := s.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", ErrInvalidCredentials
	}

	if !security.CheckPasswordHash(password, user.PasswordHash) {
		return "", ErrInvalidCredentials
	}

	token, err := s.tokenService.GenerateToken(user)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}

func (s *AuthService) Validate(ctx context.Context, token string) (*domain.User, error) {
	claims, err := s.tokenService.ValidateToken(token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id in token: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidEmail
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return ErrInvalidPassword
	}
	return nil
}

func parseRole(role string) (domain.Role, error) {
	normalizedRole := domain.Role(strings.ToLower(strings.TrimSpace(role)))
	switch normalizedRole {
	case domain.RoleAdmin, domain.RoleStudent, domain.RoleInstructor, domain.RoleProctor:
		return normalizedRole, nil
	default:
		return "", ErrInvalidRole
	}
}
