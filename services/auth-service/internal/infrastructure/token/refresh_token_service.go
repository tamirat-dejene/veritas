package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
)

// refreshTokenService generates and hashes opaque refresh tokens.
type refreshTokenService struct {
	ttl time.Duration
}

// NewRefreshTokenService creates a service for generating opaque refresh tokens.
func NewRefreshTokenService(ttl time.Duration) domain.TokenService {
	return &refreshTokenService{ttl: ttl}
}

// GenerateRefreshToken produces a cryptographically secure 32-byte random token
// (encoded as hex) and its SHA-256 hash for storage.
// The raw token is returned to the client; only the hash is stored in the database.
func (s *refreshTokenService) GenerateRefreshToken() (rawToken string, tokenHash string, err error) {
	b := make([]byte, 32) // 256-bit random token
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("refreshTokenService.GenerateRefreshToken: crypto/rand failed: %w", err)
	}

	rawToken = hex.EncodeToString(b)
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash = hex.EncodeToString(hash[:])
	return rawToken, tokenHash, nil
}

// GenerateAccessToken is not implemented on this service.
func (s *refreshTokenService) GenerateAccessToken(user *domain.User) (string, error) {
	panic("GenerateAccessToken must be called on jwtService, not refreshTokenService")
}

// HashToken returns the hex-encoded SHA-256 hash of a raw token string.
// This is used to hash an incoming raw token before looking it up in the database.
func HashToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}
