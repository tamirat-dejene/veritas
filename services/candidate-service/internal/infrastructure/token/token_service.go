package token

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
)

type jwtTokenService struct {
	secret []byte
}

func NewTokenService(secret string) domain.EnrollmentTokenService {
	return &jwtTokenService{
		secret: []byte(secret),
	}
}

func (s *jwtTokenService) GenerateToken(ctx context.Context, claims domain.EnrollmentClaims) (string, error) {
	jwtClaims := jwt.MapClaims{
		"eid": claims.EnrollmentID.String(),
		"cid": claims.CandidateID.String(),
		"xid": claims.ExamID.String(),
		"ent": claims.EnterpriseID.String(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	return token.SignedString(s.secret)
}

func (s *jwtTokenService) ParseToken(ctx context.Context, tokenString string) (*domain.EnrollmentClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		enrollmentID, err := parseUUID(claims["eid"])
		if err != nil {
			return nil, fmt.Errorf("invalid enrollment id in token")
		}
		candidateID, err := parseUUID(claims["cid"])
		if err != nil {
			return nil, fmt.Errorf("invalid candidate id in token")
		}
		examID, err := parseUUID(claims["xid"])
		if err != nil {
			return nil, fmt.Errorf("invalid exam id in token")
		}
		enterpriseID, err := parseUUID(claims["ent"])
		if err != nil {
			return nil, fmt.Errorf("invalid enterprise id in token")
		}

		return &domain.EnrollmentClaims{
			EnrollmentID: enrollmentID,
			CandidateID:  candidateID,
			ExamID:       examID,
			EnterpriseID: enterpriseID,
		}, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func parseUUID(v interface{}) (uuid.UUID, error) {
	s, ok := v.(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("not a string")
	}
	return uuid.Parse(s)
}
