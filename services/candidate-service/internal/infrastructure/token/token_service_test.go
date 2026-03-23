package token

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
)

func TestJWTTokenService(t *testing.T) {
	secret := "test-secret"
	service := NewTokenService(secret)

	claims := domain.EnrollmentClaims{
		EnrollmentID: uuid.New(),
		CandidateID:  uuid.New(),
		ExamID:       uuid.New(),
		EnterpriseID: uuid.New(),
	}

	ctx := context.Background()

	// Test GenerateToken
	token, err := service.GenerateToken(ctx, claims)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Test ParseToken
	parsedClaims, err := service.ParseToken(ctx, token)
	assert.NoError(t, err)
	assert.Equal(t, claims.EnrollmentID, parsedClaims.EnrollmentID)
	assert.Equal(t, claims.CandidateID, parsedClaims.CandidateID)
	assert.Equal(t, claims.ExamID, parsedClaims.ExamID)
	assert.Equal(t, claims.EnterpriseID, parsedClaims.EnterpriseID)

	// Test ParseToken with invalid secret
	invalidService := NewTokenService("wrong-secret")
	_, err = invalidService.ParseToken(ctx, token)
	assert.Error(t, err)
}
