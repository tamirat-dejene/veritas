package security

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
)

type TokenService struct {
	secretKey string
}

func NewTokenService(secretKey string) *TokenService {
	return &TokenService{secretKey: secretKey}
}

type UserClaims struct {
	UserID       string      `json:"sub"`
	Role         domain.Role `json:"role"`
	EnterpriseID *string     `json:"enterpriseId,omitempty"`
	Tier         string      `json:"tier,omitempty"`
	jwt.RegisteredClaims
}

func (s *TokenService) GenerateToken(user *domain.User) (string, error) {
	claims := UserClaims{
		UserID:       user.ID.String(),
		Role:         user.Role,
		EnterpriseID: user.EnterpriseID,
		Tier:         user.Tier,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "veritas-auth-service",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

func (s *TokenService) ValidateToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
