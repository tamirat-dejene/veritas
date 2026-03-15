package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

type ExamSubmission struct {
	ID            uuid.UUID `db:"id" json:"id"`
	SessionID     uuid.UUID `db:"session_id" json:"sessionId"`
	SubmittedAt   time.Time `db:"submitted_at" json:"submittedAt"`
	AutoSubmitted bool      `db:"auto_submitted" json:"autoSubmitted"`
	TotalScore    *float64  `db:"total_score" json:"totalScore,omitempty"`
	GradingStatus string    `db:"grading_status" json:"gradingStatus"`
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
	CheatingScore     *float64      `db:"cheating_score" json:"cheatingScore,omitempty"`
	CreatedAt         time.Time     `db:"created_at" json:"createdAt"`

	// Relations
	Questions  []SessionQuestion `json:"questions,omitempty"`
	Answers    []SessionAnswer   `json:"answers,omitempty"`
	Submission *ExamSubmission   `json:"submission,omitempty"`
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session *ExamSession) error
	GetSessionByID(ctx context.Context, id uuid.UUID) (*ExamSession, error)
	ListSessionsByExam(ctx context.Context, examID uuid.UUID, status *SessionStatus) ([]*ExamSession, error)
	UpdateSessionStatus(ctx context.Context, id uuid.UUID, status SessionStatus, reason *string) error

	SaveQuestionsSnapshot(ctx context.Context, sessionID uuid.UUID, questions []SessionQuestion) error
	GetSessionQuestions(ctx context.Context, sessionID uuid.UUID) ([]SessionQuestion, error)

	UpsertAnswer(ctx context.Context, answer *SessionAnswer) error
	GetSessionAnswers(ctx context.Context, sessionID uuid.UUID) ([]SessionAnswer, error)

	CreateSubmission(ctx context.Context, submission *ExamSubmission) error
	GetSubmissionBySession(ctx context.Context, sessionID uuid.UUID) (*ExamSubmission, error)
	GetSubmissionsByExam(ctx context.Context, examID uuid.UUID) ([]*ExamSubmission, error)
	WithTx(tx pgx.Tx) SessionRepository
}

// SessionUseCase covers Candidate Access Flow
type SessionUseCase interface {
	ValidateAccessToken(ctx context.Context, token string) (map[string]interface{}, error) // Returns exam basic metadata + validity
	StartSession(ctx context.Context, token string, clientIP, userAgent string) (*ExamSession, error)
	ResumeActiveSession(ctx context.Context, candidateID uuid.UUID) (*ExamSession, error)
	GetSessionDetails(ctx context.Context, sessionID uuid.UUID, requestingUserID uuid.UUID, role string) (*ExamSession, error)
	GetSessionQuestionsSnapshot(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID) ([]SessionQuestion, error)
	SaveAnswers(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID, questionID uuid.UUID, answerData json.RawMessage) error
	GetMyAnswers(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID) ([]SessionAnswer, error)
	SubmitExam(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID, autoSubmitted bool) (*ExamSubmission, error)
	TerminateSession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID, reason string) error
	ForceExpireSession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID) error
}

type MonitoringUseCase interface {
	ListSessionsForExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, status *SessionStatus, candidateID *uuid.UUID) ([]*ExamSession, error)
	GetSessionSummary(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID) (*ExamSession, error)
	GetSubmissions(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID) ([]*ExamSubmission, error)
	GetSubmissionDetail(ctx context.Context, submissionID uuid.UUID, enterpriseID uuid.UUID) (*ExamSubmission, error)
	CandidateGetResult(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID) (*ExamSubmission, error) // Returns only if grading rules allow
}
