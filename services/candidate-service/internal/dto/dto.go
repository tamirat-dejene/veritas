package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type CandidateCreateRequest struct {
	ExternalID       string  `json:"externalId" binding:"required"`
	FirstName        string  `json:"firstName" binding:"required"`
	LastName         string  `json:"lastName" binding:"required"`
	Email            *string `json:"email"`
	FaceReferenceURL *string `json:"faceReferenceUrl"`
}

type CandidateUpdateRequest struct {
	FirstName        string  `json:"firstName" binding:"required"`
	LastName         string  `json:"lastName" binding:"required"`
	Email            *string `json:"email"`
	FaceReferenceURL *string `json:"faceReferenceUrl"`
}

type CandidateResponse struct {
	Data *domain.CandidateProfile `json:"data"`
}

type CandidateListResponse struct {
	Data []*domain.CandidateProfile `json:"data"`
}

type BulkUploadResponse struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type EnrollmentRequest struct {
	CandidateIDs   []uuid.UUID `json:"candidateIds" binding:"required,min=1"`
	MaxAttempts    int         `json:"maxAttempts" binding:"required,min=1"`
	TokenExpiresAt time.Time   `json:"tokenExpiresAt" binding:"required"`
}

type EnrollmentCreateResponse struct {
	Message   string   `json:"message"`
	RawTokens []string `json:"rawTokens"`
}

type EnrollmentResponse struct {
	Data *domain.ExamEnrollment `json:"data"`
}

type EnrollmentListResponse struct {
	Data []*domain.ExamEnrollment `json:"data"`
}

type EnrollmentTokenResponse struct {
	Message  string `json:"message"`
	RawToken string `json:"rawToken"`
}

type AccessValidateResponse struct {
	Data *domain.ValidateAccessTokenResponse `json:"data"`
}

type SaveAnswerRequest struct {
	SessionQuestionID uuid.UUID       `json:"sessionQuestionId" binding:"required"`
	AnswerData        json.RawMessage `json:"answerData" binding:"required"`
}

// SwaggerAnswerData defines the expected polymorphic payload for answered questions.
// One of the specific fields must be populated based on the Question Type.
type SwaggerAnswerData struct {
	SelectedOptionIDs []string `json:"selectedOptionIds,omitempty" example:"123e4567-e89b-12d3-a456-426614174000"` // For MCQ and True/False Answer
	Text              string   `json:"text,omitempty" example:"This is an essay answer."`                          // For Text or Essay Answer
}

// SaveAnswerRequestSwag is only used for Swagger documentation to provide struct assertion.
type SaveAnswerRequestSwag struct {
	SessionQuestionID uuid.UUID         `json:"sessionQuestionId" binding:"required"`
	AnswerData        SwaggerAnswerData `json:"answerData" binding:"required"`
}

type SubmitRequest struct {
	AutoSubmitted bool `json:"autoSubmitted"`
}

type TerminateSessionRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type SessionResponse struct {
	Data *domain.ExamSession `json:"data"`
}

type SessionListResponse struct {
	Data []*domain.ExamSession `json:"data"`
}

type SessionQuestionListResponse struct {
	Data []domain.SessionQuestion `json:"data"`
}

type SessionAnswerListResponse struct {
	Data []domain.SessionAnswer `json:"data"`
}

type SubmissionResponse struct {
	Data *domain.ExamSubmission `json:"data"`
}

type SubmissionListResponse struct {
	Data []*domain.ExamSubmission `json:"data"`
}

type SubmitResponse struct {
	Message string                 `json:"message"`
	Data    *domain.ExamSubmission `json:"data"`
}

type ValidateAccessTokenResponse struct {
	Data map[string]any `json:"data"`
}
