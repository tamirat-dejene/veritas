package domain

import (
	"context"
	"encoding/json"
	"io"
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

// BulkAnswerItem is one entry in a bulk save-answers request.
type BulkAnswerItem struct {
	SessionQuestionID uuid.UUID       `json:"sessionQuestionId"`
	AnswerData        json.RawMessage `json:"answerData"`
}

// BulkAnswerResult is the per-item outcome returned in a 207 bulk-save response.
type BulkAnswerResult struct {
	SessionQuestionID uuid.UUID `json:"sessionQuestionId"`
	Status            string    `json:"status"` // "saved" | "failed"
	Error             *string   `json:"error,omitempty"`
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
	FaceRegisteredEmbedding []float64 `db:"face_registered_embedding" json:"faceRegisteredEmbedding,omitempty"`
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
	BulkUpsertAnswer(ctx context.Context, answers []*SessionAnswer) ([]uuid.UUID, error)
	GetSessionAnswers(ctx context.Context, sessionID uuid.UUID) ([]SessionAnswer, error)

	CreateSubmission(ctx context.Context, submission *ExamSubmission) error
	GetSubmissionBySession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID) (*ExamSubmission, error)
	GetSubmissionsByExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params) ([]*ExamSubmission, int64, error)
	GetSubmissionByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*ExamSubmission, error)
	GetSessionByEnrollment(ctx context.Context, enrollmentID uuid.UUID) (*ExamSession, error)
	CountSessionsByEnterpriseAndStatus(ctx context.Context, enterpriseID uuid.UUID, status SessionStatus) (int, error)
	GetExpiredActiveSessions(ctx context.Context, limit int) ([]*ExamSession, error)
	TerminateActiveSessionsByEnterprise(ctx context.Context, enterpriseID uuid.UUID, reason string) error
	TerminateActiveSessionsByExam(ctx context.Context, examID uuid.UUID, reason string) error
	WithTx(tx pgx.Tx) SessionRepository
}

// SessionUseCase covers Candidate Access Flow and internal service-to-service data access.
type SessionUseCase interface {
	ValidateAccessToken(ctx context.Context, enrollmentID, enterpriseID uuid.UUID) (*ValidateAccessTokenResponse, error)
	StartSession(ctx context.Context, enrollmentID, enterpriseID uuid.UUID, clientIP, userAgent string, faceImage io.Reader) (*ExamSession, error)
	ResumeActiveSession(ctx context.Context, candidateID uuid.UUID) (*ExamSession, error)
	GetSessionDetails(ctx context.Context, sessionID uuid.UUID, requestingUserID uuid.UUID, role string) (*ExamSession, error)
	GetSessionQuestionsSnapshot(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID) ([]SessionQuestion, error)
	SaveAnswers(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID, sessionQuestionID uuid.UUID, answerData json.RawMessage) error
	BulkSaveAnswers(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID, answers []BulkAnswerItem) ([]BulkAnswerResult, error)
	GetMyAnswers(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID) ([]SessionAnswer, error)
	SubmitExam(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID, autoSubmitted bool) (*ExamSubmission, error)
	TerminateSession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID, reason string) error
	ForceExpireSession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID) error
	InternalUseCase
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

type GradingOption struct {
	ID      uuid.UUID `json:"id"`
	Content string    `json:"content"`
}

type CandidateAnswerData struct {
	SelectedOptionIDs []uuid.UUID `json:"selectedOptionIds,omitempty"`
	Text              *string     `json:"text,omitempty"`
}

// GradingItem unifies the true evaluation criteria and the candidate's actual answer
type GradingItem struct {
	QuestionID        uuid.UUID `json:"question_id"`
	SessionQuestionID uuid.UUID `json:"session_question_id"`
	QuestionType      string    `json:"question_type"`
	Content           string    `json:"content"`
	Title             string    `json:"title"`
	Topic             string    `json:"topic"`
	MediaURL          *string   `json:"media_url,omitempty"`
	// Scoring
	Points         int     `json:"points"`
	NegativePoints float64 `json:"negative_points"`

	// True Evaluation Criteria (from Exam Service)
	ExpectedAnswer     *string        `json:"expected_answer,omitempty"`
	EvaluationCriteria map[string]any `json:"evaluation_criteria,omitempty"`
	CorrectOptionIDs   []uuid.UUID    `json:"correct_option_ids,omitempty"`
	Options            []GradingOption `json:"options,omitempty"`

	// Candidate's Actual Answer (from Candidate Service)
	HasAnswer       bool                 `json:"has_answer"`
	CandidateAnswer *CandidateAnswerData `json:"candidate_answer,omitempty"`
}

// ExamReadyForGradingEvent is the slim Kafka trigger published on topic exam.session.ready_for_grading (v3.0).
// It carries only identifiers and session metadata; the grading-service fetches the full
// grading payload via the candidate-service internal HTTP endpoint.
type ExamReadyForGradingEvent struct {
	EventID   uuid.UUID `json:"event_id"`
	EventType string    `json:"event_type"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	TraceID   string    `json:"trace_id,omitempty"`

	// Context
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	ExamID       uuid.UUID `json:"exam_id"`
	SessionID    uuid.UUID `json:"session_id"`
	CandidateID  uuid.UUID `json:"candidate_id"`
	EnrollmentID uuid.UUID `json:"enrollment_id"`

	// Metadata
	Status            string     `json:"status"`
	StartedAt         time.Time  `json:"started_at"`
	SubmittedAt       *time.Time `json:"submitted_at,omitempty"`
	TerminatedAt      *time.Time `json:"terminated_at,omitempty"`
	AutoSubmitted     bool       `json:"auto_submitted"`
	TerminationReason *string    `json:"termination_reason,omitempty"`
}

// GradingPayload is the response body returned by the internal grading-payload endpoint.
// It aggregates session questions, candidate answers, and master question evaluation
// criteria so the grading-service can grade the exam in a single HTTP call.
type GradingPayload struct {
	SessionID         uuid.UUID  `json:"session_id"`
	EnterpriseID      uuid.UUID  `json:"enterprise_id"`
	ExamID            uuid.UUID  `json:"exam_id"`
	CandidateID       uuid.UUID  `json:"candidate_id"`
	EnrollmentID      uuid.UUID  `json:"enrollment_id"`
	CandidateName     string     `json:"candidate_name"`
	CandidateEmail    string     `json:"candidate_email"`
	ExamTitle         string     `json:"exam_title"`
	Status            string     `json:"status"`
	StartedAt         time.Time  `json:"started_at"`
	SubmittedAt       *time.Time `json:"submitted_at,omitempty"`
	TerminatedAt      *time.Time `json:"terminated_at,omitempty"`
	AutoSubmitted     bool       `json:"auto_submitted"`
	TerminationReason *string    `json:"termination_reason,omitempty"`
	Items             []GradingItem `json:"items"`
}

// InternalUseCase covers service-to-service data access.
type InternalUseCase interface {
	BuildGradingPayload(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID) (*GradingPayload, error)
}
