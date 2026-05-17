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
	ExternalID string  `json:"externalId" binding:"required"`
	FirstName  string  `json:"firstName" binding:"required"`
	LastName   string  `json:"lastName" binding:"required"`
	Email      *string `json:"email"`
}

type CandidateUpdateRequest struct {
	FirstName string  `json:"firstName" binding:"required"`
	LastName  string  `json:"lastName" binding:"required"`
	Email     *string `json:"email"`
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

// EnrollmentResultItem is per-candidate data returned on enrollment creation.
// InvitationURL is an opaque-code URL safe for the admin to copy/distribute.
// The raw JWT and raw opaque code are NEVER returned.
type EnrollmentResultItem struct {
	EnrollmentID  uuid.UUID `json:"enrollmentId"`
	CandidateID   uuid.UUID `json:"candidateId"`
	InvitationURL string    `json:"invitationUrl"`
	Status        string    `json:"status"`
}

type EnrollmentCreateResponse struct {
	Message string                 `json:"message"`
	Results []EnrollmentResultItem `json:"results"`
}

type EnrollmentResponse struct {
	Data *domain.ExamEnrollment `json:"data"`
}

type EnrollmentListResponse struct {
	Data []*domain.ExamEnrollment `json:"data"`
}

// NotifyResultItem describes the outcome of a single notification attempt.
type NotifyResultItem struct {
	EnrollmentID uuid.UUID `json:"enrollmentId"`
	CandidateID  uuid.UUID `json:"candidateId"`
	// NotifyStatus: "sent" | "skipped_no_email"
	NotifyStatus string `json:"notifyStatus"`
}

type NotifyResponse struct {
	Message string             `json:"message"`
	Results []NotifyResultItem `json:"results"`
}

// NotifySingleRequest optionally lets the caller specify enrollment IDs for bulk notify.
type NotifyBulkRequest struct {
	// EnrollmentIDs is optional; if empty, all Pending/Invited enrollments for the exam are notified.
	EnrollmentIDs []uuid.UUID `json:"enrollmentIds"`
}

// RedeemRequest is the payload the candidate frontend sends to exchange an opaque code for a JWT.
type RedeemRequest struct {
	Code string `json:"code" binding:"required"`
}

// RedeemResponse carries the raw JWT in the response body (never in a URL).
type RedeemResponse struct {
	Token string `json:"token"`
}

// InvitationLinkResponse is returned when an admin requests a fresh link for a no-email candidate.
type InvitationLinkResponse struct {
	EnrollmentID   uuid.UUID `json:"enrollmentId"`
	InvitationURL  string    `json:"invitationUrl"`
	Status         string    `json:"status"`
	TokenExpiresAt time.Time `json:"tokenExpiresAt"`
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

// BulkAnswerItemRequest is one entry in a bulk save-answers request body.
type BulkAnswerItemRequest struct {
	SessionQuestionID uuid.UUID       `json:"sessionQuestionId" binding:"required"`
	AnswerData        json.RawMessage `json:"answerData"        binding:"required"`
}

// BulkSaveAnswersRequest is the body for PUT /sessions/{sessionId}/answers.
// At most 100 items may be submitted in a single request.
type BulkSaveAnswersRequest struct {
	Answers []BulkAnswerItemRequest `json:"answers" binding:"required,min=1,max=100"`
}

// BulkAnswerItemRequestSwag and BulkSaveAnswersRequestSwag are Swagger-only types.
type BulkAnswerItemRequestSwag struct {
	SessionQuestionID uuid.UUID         `json:"sessionQuestionId"`
	AnswerData        SwaggerAnswerData `json:"answerData"`
}

type BulkSaveAnswersRequestSwag struct {
	Answers []BulkAnswerItemRequestSwag `json:"answers"`
}

// BulkAnswerResultItem is the per-item outcome within a 207 bulk-save response.
type BulkAnswerResultItem struct {
	SessionQuestionID uuid.UUID `json:"sessionQuestionId"`
	Status            string    `json:"status"` // "saved" | "failed"
	Error             *string   `json:"error,omitempty"`
}

// BulkSaveAnswersResponse is the 207 response for PUT /sessions/{sessionId}/answers.
type BulkSaveAnswersResponse struct {
	SavedCount  int                    `json:"savedCount"`
	FailedCount int                    `json:"failedCount"`
	Results     []BulkAnswerResultItem `json:"results"`
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

type CandidateEmailsResponse struct {
	Emails []string `json:"emails"`
}
