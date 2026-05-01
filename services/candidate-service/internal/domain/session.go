package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type SessionStatus string

const (
	SessionActive     SessionStatus = "Active"
	SessionSubmitted  SessionStatus = "Submitted"
	SessionTerminated SessionStatus = "Terminated"
	SessionExpired    SessionStatus = "Expired"
)

type SessionQuestion struct {
	ID               uuid.UUID       `db:"id" json:"id"`
	SessionID        uuid.UUID       `db:"session_id" json:"sessionId"`
	QuestionID       uuid.UUID       `db:"question_id" json:"questionId"`
	QuestionSnapshot json.RawMessage `db:"question_snapshot" json:"questionSnapshot"`
	OrderIndex       int             `db:"order_index" json:"orderIndex"`
	Points           int             `db:"points" json:"points"`
	NegativePoints   float64         `db:"negative_points" json:"negativePoints"`
}

type SessionAnswer struct {
	ID                uuid.UUID       `db:"id" json:"id"`
	SessionID         uuid.UUID       `db:"session_id" json:"sessionId"`
	SessionQuestionID uuid.UUID       `db:"session_question_id" json:"sessionQuestionId"`
	AnswerData        json.RawMessage `db:"answer_data" json:"answerData"`
	IsFinal           bool            `db:"is_final" json:"isFinal"`
	SavedAt           time.Time       `db:"saved_at" json:"savedAt"`
}

type MCQAnswer struct {
	SelectedOptionIDs []uuid.UUID `json:"selectedOptionIds"`
}

type TextAnswer struct {
	Text string `json:"text"`
}

type ExamSubmission struct {
	ID            uuid.UUID `db:"id" json:"id"`
	SessionID     uuid.UUID `db:"session_id" json:"sessionId"`
	SubmittedAt   time.Time `db:"submitted_at" json:"submittedAt"`
	AutoSubmitted bool      `db:"auto_submitted" json:"autoSubmitted"`
	CreatedAt     time.Time `db:"created_at" json:"createdAt"`
}

type ExamSession struct {
	ID                uuid.UUID     `db:"id" json:"id"`
	EnterpriseID      uuid.UUID     `db:"enterprise_id" json:"enterpriseId"`
	ExamID            uuid.UUID     `db:"exam_id" json:"examId"`
	CandidateID       uuid.UUID     `db:"candidate_id" json:"candidateId"`
	EnrollmentID      uuid.UUID     `db:"enrollment_id" json:"enrollmentId"`
	Status            SessionStatus `db:"status" json:"status"`
	StartedAt         time.Time     `db:"started_at" json:"startedAt"`
	ExpiresAt         time.Time     `db:"expires_at" json:"expiresAt"`
	SubmittedAt       *time.Time    `db:"submitted_at" json:"submittedAt,omitempty"`
	TerminatedAt      *time.Time    `db:"terminated_at" json:"terminatedAt,omitempty"`
	TerminationReason *string       `db:"termination_reason" json:"terminationReason,omitempty"`
	ClientIP          *string       `db:"client_ip" json:"clientIp,omitempty"`
	UserAgent         *string       `db:"user_agent" json:"userAgent,omitempty"`
	FaceRegisteredURL *string       `db:"face_registered_url" json:"faceRegisteredUrl,omitempty"`
	CreatedAt         time.Time     `db:"created_at" json:"createdAt"`

	// Relations
	Questions  []SessionQuestion `json:"questions,omitempty"`
	Answers    []SessionAnswer   `json:"answers,omitempty"`
	Submission *ExamSubmission   `json:"submission,omitempty"`
}

type ValidateAccessTokenResponse struct {
	EnrollmentID uuid.UUID `json:"enrollmentId"`
	CandidateID  uuid.UUID `json:"candidateId"`
	ExamID       uuid.UUID `json:"examId"`
	EnterpriseID uuid.UUID `json:"enterpriseId"`
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session *ExamSession) error
	GetSessionByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*ExamSession, error)
	ListSessionsByExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, status *SessionStatus, params pagination.Params) ([]*ExamSession, int64, error)
	UpdateSessionStatus(ctx context.Context, id uuid.UUID, status SessionStatus, reason *string) error

	SaveQuestionsSnapshot(ctx context.Context, sessionID uuid.UUID, questions []SessionQuestion) error
	GetSessionQuestions(ctx context.Context, sessionID uuid.UUID) ([]SessionQuestion, error)
	GetSessionQuestion(ctx context.Context, sessionID uuid.UUID, sessionQuestionID uuid.UUID) (*SessionQuestion, error)

	UpsertAnswer(ctx context.Context, answer *SessionAnswer) error
	GetSessionAnswers(ctx context.Context, sessionID uuid.UUID) ([]SessionAnswer, error)

	CreateSubmission(ctx context.Context, submission *ExamSubmission) error
	GetSubmissionBySession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID) (*ExamSubmission, error)
	GetSubmissionsByExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params) ([]*ExamSubmission, int64, error)
	GetSubmissionByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*ExamSubmission, error)
	GetSessionByEnrollment(ctx context.Context, enrollmentID uuid.UUID) (*ExamSession, error)
	CountSessionsByEnterpriseAndStatus(ctx context.Context, enterpriseID uuid.UUID, status SessionStatus) (int, error)
	WithTx(tx pgx.Tx) SessionRepository
}

// SessionUseCase covers Candidate Access Flow
type SessionUseCase interface {
	ValidateAccessToken(ctx context.Context, enrollmentID, enterpriseID uuid.UUID) (*ValidateAccessTokenResponse, error)
	StartSession(ctx context.Context, enrollmentID, enterpriseID uuid.UUID, clientIP, userAgent string) (*ExamSession, error)
	ResumeActiveSession(ctx context.Context, candidateID uuid.UUID) (*ExamSession, error)
	GetSessionDetails(ctx context.Context, sessionID uuid.UUID, requestingUserID uuid.UUID, role string) (*ExamSession, error)
	GetSessionQuestionsSnapshot(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID) ([]SessionQuestion, error)
	SaveAnswers(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID, sessionQuestionID uuid.UUID, answerData json.RawMessage) error
	GetMyAnswers(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID) ([]SessionAnswer, error)
	SubmitExam(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID, autoSubmitted bool) (*ExamSubmission, error)
	TerminateSession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID, reason string) error
	ForceExpireSession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID) error
}

type MonitoringUseCase interface {
	ListSessionsForExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, status *SessionStatus, candidateID *uuid.UUID, params pagination.Params) ([]*ExamSession, int64, error)
	GetSessionSummary(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID) (*ExamSession, error)
	GetSubmissions(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params) ([]*ExamSubmission, int64, error)
	GetSubmissionDetail(ctx context.Context, submissionID uuid.UUID, enterpriseID uuid.UUID) (*ExamSubmission, error)
	GetActiveSessionsCount(ctx context.Context, enterpriseID uuid.UUID) (int, error)
}

// CandidateExamSubmittedEvent is the Kafka payload published on topic candidate.exam.submitted
type CandidateExamSubmittedEvent struct {
	SessionID      uuid.UUID `json:"session_id"`
	CandidateID    uuid.UUID `json:"candidate_id"`
	ExamID         uuid.UUID `json:"exam_id"`
	EnterpriseID   uuid.UUID `json:"enterprise_id"`
	CandidateName  string    `json:"candidate_name"`
	CandidateEmail string    `json:"candidate_email"`
	ExamTitle      string    `json:"exam_title"`
	SubmittedAt    time.Time `json:"submitted_at"`
	AutoSubmitted  bool      `json:"auto_submitted"`
	Timestamp      int64     `json:"timestamp"`
}
