package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/topics"
)

type kafkaPublisher struct {
	publisher messaging.Publisher
}

// NewKafkaPublisher creates a new EventPublisher backed by Kafka.
func NewKafkaPublisher(pub messaging.Publisher) domain.EventPublisher {
	return &kafkaPublisher{
		publisher: pub,
	}
}

// EnterpriseCreatedEvent is the payload for the enterprise.created event.
type EnterpriseCreatedEvent struct {
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Name         string    `json:"name"`
	OwnerEmail   string    `json:"owner_email"`
	Timestamp    int64     `json:"timestamp"`
}

// PublishEnterpriseCreated publishes an enterprise.created event.
func (p *kafkaPublisher) PublishEnterpriseCreated(ctx context.Context, enterpriseID uuid.UUID, legalName string, ownerEmail string) error {
	event := EnterpriseCreatedEvent{
		EnterpriseID: enterpriseID,
		Name:         legalName,
		OwnerEmail:   ownerEmail,
		Timestamp:    time.Now().Unix(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka_publisher: marshal event: %w", err)
	}

	msg := messaging.Message{
		Topic: topics.EnterpriseCreated,
		Key:   enterpriseID[:],
		Value: payload,
	}

	if err := p.publisher.Publish(ctx, msg); err != nil {
		return fmt.Errorf("kafka_publisher: publish: %w", err)
	}

	return nil
}

// EnterpriseStaffCreatedEvent is the payload for the enterprise.staff.created event.
type EnterpriseStaffCreatedEvent struct {
	StaffID        uuid.UUID `json:"staff_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	TempPassword   string    `json:"temp_password"`
	EnterpriseName string    `json:"enterprise_name"`
	Timestamp      int64     `json:"timestamp"`
}

// PublishEnterpriseStaffCreated publishes an enterprise.staff.created event.
func (p *kafkaPublisher) PublishEnterpriseStaffCreated(ctx context.Context, staffID uuid.UUID, email, name, tempPassword, enterpriseName string) error {
	event := EnterpriseStaffCreatedEvent{
		StaffID:        staffID,
		Email:          email,
		Name:           name,
		TempPassword:   tempPassword,
		EnterpriseName: enterpriseName,
		Timestamp:      time.Now().Unix(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka_publisher: marshal event: %w", err)
	}

	msg := messaging.Message{
		Topic: topics.EnterpriseStaffCreated,
		Key:   staffID[:],
		Value: payload,
	}

	if err := p.publisher.Publish(ctx, msg); err != nil {
		return fmt.Errorf("kafka_publisher: publish: %w", err)
	}

	return nil
}

// PasswordResetRequestedEvent is the payload for the enterprise.password.reset.requested event.
type PasswordResetRequestedEvent struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	ResetLink string    `json:"reset_link"`
	Timestamp int64     `json:"timestamp"`
}

// PublishPasswordResetRequested publishes a password reset request event.
func (p *kafkaPublisher) PublishPasswordResetRequested(ctx context.Context, userID uuid.UUID, email, name, resetLink string) error {
	event := PasswordResetRequestedEvent{
		UserID:    userID,
		Email:     email,
		Name:      name,
		ResetLink: resetLink,
		Timestamp: time.Now().Unix(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka_publisher: marshal event: %w", err)
	}

	msg := messaging.Message{
		Topic: topics.EnterprisePasswordResetRequested,
		Key:   userID[:],
		Value: payload,
	}

	if err := p.publisher.Publish(ctx, msg); err != nil {
		return fmt.Errorf("kafka_publisher: publish: %w", err)
	}

	return nil
}

// EnterpriseLifecycleEvent is the payload for general enterprise lifecycle events.
type EnterpriseLifecycleEvent struct {
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Name         string    `json:"name"`
	ContactEmail string    `json:"contact_email"`
	Reason       string    `json:"reason,omitempty"`
	Timestamp    int64     `json:"timestamp"`
}

func (p *kafkaPublisher) PublishEnterpriseApproved(ctx context.Context, enterpriseID uuid.UUID, legalName, contactEmail string) error {
	event := EnterpriseLifecycleEvent{
		EnterpriseID: enterpriseID,
		Name:         legalName,
		ContactEmail: contactEmail,
		Timestamp:    time.Now().Unix(),
	}
	return p.publishLifecycleEvent(ctx, topics.EnterpriseApproved, enterpriseID, event)
}

func (p *kafkaPublisher) PublishEnterpriseSuspended(ctx context.Context, enterpriseID uuid.UUID, legalName, contactEmail, reason string) error {
	event := EnterpriseLifecycleEvent{
		EnterpriseID: enterpriseID,
		Name:         legalName,
		ContactEmail: contactEmail,
		Reason:       reason,
		Timestamp:    time.Now().Unix(),
	}
	return p.publishLifecycleEvent(ctx, topics.EnterpriseSuspended, enterpriseID, event)
}

func (p *kafkaPublisher) PublishEnterpriseDeleted(ctx context.Context, enterpriseID uuid.UUID, legalName, contactEmail string) error {
	event := EnterpriseLifecycleEvent{
		EnterpriseID: enterpriseID,
		Name:         legalName,
		ContactEmail: contactEmail,
		Timestamp:    time.Now().Unix(),
	}
	return p.publishLifecycleEvent(ctx, topics.EnterpriseDeleted, enterpriseID, event)
}

func (p *kafkaPublisher) PublishEnterpriseHardDeleted(ctx context.Context, enterpriseID uuid.UUID, legalName, contactEmail string) error {
	event := EnterpriseLifecycleEvent{
		EnterpriseID: enterpriseID,
		Name:         legalName,
		ContactEmail: contactEmail,
		Timestamp:    time.Now().Unix(),
	}
	return p.publishLifecycleEvent(ctx, topics.EnterpriseHardDeleted, enterpriseID, event)
}

func (p *kafkaPublisher) PublishEnterpriseReactivated(ctx context.Context, enterpriseID uuid.UUID, legalName, contactEmail string) error {
	event := EnterpriseLifecycleEvent{
		EnterpriseID: enterpriseID,
		Name:         legalName,
		ContactEmail: contactEmail,
		Timestamp:    time.Now().Unix(),
	}
	return p.publishLifecycleEvent(ctx, topics.EnterpriseReactivated, enterpriseID, event)
}

func (p *kafkaPublisher) PublishEnterpriseRestored(ctx context.Context, enterpriseID uuid.UUID, legalName, contactEmail string) error {
	event := EnterpriseLifecycleEvent{
		EnterpriseID: enterpriseID,
		Name:         legalName,
		ContactEmail: contactEmail,
		Timestamp:    time.Now().Unix(),
	}
	return p.publishLifecycleEvent(ctx, topics.EnterpriseRestored, enterpriseID, event)
}

func (p *kafkaPublisher) publishLifecycleEvent(ctx context.Context, topic string, key uuid.UUID, event EnterpriseLifecycleEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka_publisher: marshal lifecycle event: %w", err)
	}
	msg := messaging.Message{
		Topic: topic,
		Key:   key[:],
		Value: payload,
	}
	if err := p.publisher.Publish(ctx, msg); err != nil {
		return fmt.Errorf("kafka_publisher: publish %s: %w", topic, err)
	}
	return nil
}

// UserLifecycleEvent is the payload for user lifecycle events.
type UserLifecycleEvent struct {
	UserID         uuid.UUID `json:"user_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	EnterpriseName string    `json:"enterprise_name"`
	TempPassword   string    `json:"temp_password,omitempty"`
	Timestamp      int64     `json:"timestamp"`
}

func (p *kafkaPublisher) PublishUserDeactivated(ctx context.Context, userID uuid.UUID, email, name, enterpriseName string) error {
	event := UserLifecycleEvent{
		UserID:         userID,
		Email:          email,
		Name:           name,
		EnterpriseName: enterpriseName,
		Timestamp:      time.Now().Unix(),
	}
	return p.publishUserLifecycleEvent(ctx, topics.UserDeactivated, userID, event)
}

func (p *kafkaPublisher) PublishUserActivated(ctx context.Context, userID uuid.UUID, email, name, enterpriseName string) error {
	event := UserLifecycleEvent{
		UserID:         userID,
		Email:          email,
		Name:           name,
		EnterpriseName: enterpriseName,
		Timestamp:      time.Now().Unix(),
	}
	return p.publishUserLifecycleEvent(ctx, topics.UserActivated, userID, event)
}

func (p *kafkaPublisher) PublishUserDeleted(ctx context.Context, userID uuid.UUID, email, name, enterpriseName string) error {
	event := UserLifecycleEvent{
		UserID:         userID,
		Email:          email,
		Name:           name,
		EnterpriseName: enterpriseName,
		Timestamp:      time.Now().Unix(),
	}
	return p.publishUserLifecycleEvent(ctx, topics.UserDeleted, userID, event)
}

func (p *kafkaPublisher) PublishUserPasswordChanged(ctx context.Context, userID uuid.UUID, email, name, enterpriseName string) error {
	event := UserLifecycleEvent{
		UserID:         userID,
		Email:          email,
		Name:           name,
		EnterpriseName: enterpriseName,
		Timestamp:      time.Now().Unix(),
	}
	return p.publishUserLifecycleEvent(ctx, topics.UserPasswordChanged, userID, event)
}

func (p *kafkaPublisher) PublishUserPasswordResetAdmin(ctx context.Context, userID uuid.UUID, email, name, tempPassword, enterpriseName string) error {
	event := UserLifecycleEvent{
		UserID:         userID,
		Email:          email,
		Name:           name,
		EnterpriseName: enterpriseName,
		TempPassword:   tempPassword,
		Timestamp:      time.Now().Unix(),
	}
	return p.publishUserLifecycleEvent(ctx, topics.UserPasswordResetAdmin, userID, event)
}

func (p *kafkaPublisher) publishUserLifecycleEvent(ctx context.Context, topic string, key uuid.UUID, event UserLifecycleEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka_publisher: marshal user lifecycle event: %w", err)
	}
	msg := messaging.Message{
		Topic: topic,
		Key:   key[:],
		Value: payload,
	}
	if err := p.publisher.Publish(ctx, msg); err != nil {
		return fmt.Errorf("kafka_publisher: publish %s: %w", topic, err)
	}
	return nil
}
