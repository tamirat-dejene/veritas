package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
)

// authClaims defines the JWT payload for this service.
type authClaims struct {
	Email        string  `json:"email"`
	Role         string  `json:"role"`
	EnterpriseID *string `json:"enterpriseId"`
	jwt.RegisteredClaims
}

// jwtService generates and validates JWT access tokens.
type jwtService struct {
	secretKey []byte
	ttl       time.Duration
}

// NewJWTService creates a new jwtService using HMAC-SHA256.
func NewJWTService(secretKey string, ttl time.Duration) domain.TokenService {
	return &jwtService{
		secretKey: []byte(secretKey),
		ttl:       ttl,
	}
}

// GenerateAccessToken creates a signed JWT for the provided user.
func (s *jwtService) GenerateAccessToken(user *domain.User) (string, error) {
	now := time.Now()

	var enterpriseID *string
	if user.EnterpriseID != nil {
		id := user.EnterpriseID.String()
		enterpriseID = &id
	}

	claims := authClaims{
		Email:        user.Email,
		Role:         string(user.Role),
		EnterpriseID: enterpriseID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("jwtService.GenerateAccessToken: %w", err)
	}
	return signed, nil
}

// GenerateRefreshToken is implemented by refreshTokenService; this service only handles JWTs.
func (s *jwtService) GenerateRefreshToken() (string, string, error) {
	panic("GenerateRefreshToken must be called on refreshTokenService, not jwtService")
}
