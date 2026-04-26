package domain

import (
	"context"

	"github.com/google/uuid"
)

// EventPublisher defines the interface for publishing exam-related events.
type EventPublisher interface {
	PublishExamCreated(ctx context.Context, event ExamCreatedEvent) error
	PublishExamScheduled(ctx context.Context, event ExamLifecycleEvent) error
	PublishExamPublished(ctx context.Context, event ExamLifecycleEvent) error
	PublishExamClosed(ctx context.Context, event ExamLifecycleEvent) error
}

// EnterpriseClient defines the interface for fetching enterprise details.
type EnterpriseClient interface {
	GetEnterpriseAdminEmail(ctx context.Context, enterpriseID uuid.UUID) (string, error)
}

// CandidateClient defines the interface for fetching candidate details.
type CandidateClient interface {
	GetCandidateEmailsForExam(ctx context.Context, enterpriseID, examID uuid.UUID) ([]string, error)
}
