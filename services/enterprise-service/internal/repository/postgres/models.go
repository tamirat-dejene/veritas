package postgres

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
)

// userModel represents the database schema for veritas_users in enterprise-service.
type userModel struct {
	ID                  uuid.UUID
	Email               string
	PasswordHash        string
	Honorific           pgtype.Text
	FirstName           pgtype.Text
	LastName            pgtype.Text
	Phone               pgtype.Text
	Role                string
	EnterpriseID        pgtype.UUID
	IsActive            bool
	IsDeleted           bool
	EmailVerified       bool
	EmailVerifiedAt     pgtype.Timestamptz
	FailedLoginAttempts int32
	LockedUntil         pgtype.Timestamptz
	PasswordChangedAt   time.Time
	MustChangePassword  bool
	LastLoginAt         pgtype.Timestamptz
	LastLoginIP         pgtype.Text
	LastUserAgent       pgtype.Text
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (m *userModel) toDomain() *domain.User {
	u := &domain.User{
		ID:                  m.ID,
		Email:               m.Email,
		PasswordHash:        m.PasswordHash,
		Role:                domain.Role(m.Role),
		IsActive:            m.IsActive,
		IsDeleted:           m.IsDeleted,
		EmailVerified:       m.EmailVerified,
		FailedLoginAttempts: int(m.FailedLoginAttempts),
		PasswordChangedAt:   m.PasswordChangedAt,
		MustChangePassword:  m.MustChangePassword,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}

	if m.Honorific.Valid {
		u.Honorific = &m.Honorific.String
	}
	if m.FirstName.Valid {
		u.FirstName = &m.FirstName.String
	}
	if m.LastName.Valid {
		u.LastName = &m.LastName.String
	}
	if m.Phone.Valid {
		u.Phone = &m.Phone.String
	}
	if m.EnterpriseID.Valid {
		id := uuid.UUID(m.EnterpriseID.Bytes)
		u.EnterpriseID = &id
	}
	if m.EmailVerifiedAt.Valid {
		u.EmailVerifiedAt = &m.EmailVerifiedAt.Time
	}
	if m.LockedUntil.Valid {
		u.LockedUntil = &m.LockedUntil.Time
	}
	if m.LastLoginAt.Valid {
		u.LastLoginAt = &m.LastLoginAt.Time
	}
	if m.LastLoginIP.Valid {
		u.LastLoginIP = &m.LastLoginIP.String
	}
	if m.LastUserAgent.Valid {
		u.LastUserAgent = &m.LastUserAgent.String
	}

	return u
}

// enterpriseModel represents the database schema for enterprises.
type enterpriseModel struct {
	ID                 uuid.UUID
	Slug               string
	DisplayName        string
	LegalName          string
	ContactEmail       string
	OwnerAccountID     uuid.UUID
	Status             string
	ApprovedAt         pgtype.Timestamptz
	SuspendedAt        pgtype.Timestamptz
	DeletedAt          pgtype.Timestamptz
	SubscriptionPlanID pgtype.UUID
	SubscriptionStatus pgtype.Text
	CurrentPeriodStart pgtype.Timestamptz
	CurrentPeriodEnd   pgtype.Timestamptz
	LogoURL            pgtype.Text
	PrimaryColor       pgtype.Text
	SecondaryColor     pgtype.Text
	CustomDomain       pgtype.Text
	ContactPhone       pgtype.Text
	AddressLine1       pgtype.Text
	AddressLine2       pgtype.Text
	City               pgtype.Text
	Country            pgtype.Text
	Settings           json.RawMessage
	RetentionUntil     pgtype.Timestamptz
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CreatedBy          uuid.UUID
	UpdatedBy          uuid.UUID
}

func (m *enterpriseModel) toDomain() *domain.Enterprise {
	e := &domain.Enterprise{
		ID:             m.ID,
		Slug:           m.Slug,
		DisplayName:    m.DisplayName,
		LegalName:      m.LegalName,
		ContactEmail:   m.ContactEmail,
		OwnerAccountID: m.OwnerAccountID,
		Status:         domain.EnterpriseStatus(m.Status),
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		CreatedBy:      m.CreatedBy,
		UpdatedBy:      m.UpdatedBy,
	}

	if m.ApprovedAt.Valid {
		e.ApprovedAt = &m.ApprovedAt.Time
	}
	if m.SuspendedAt.Valid {
		e.SuspendedAt = &m.SuspendedAt.Time
	}
	if m.DeletedAt.Valid {
		e.DeletedAt = &m.DeletedAt.Time
	}
	if m.SubscriptionPlanID.Valid {
		id := uuid.UUID(m.SubscriptionPlanID.Bytes)
		e.SubscriptionPlanID = &id
	}
	if m.SubscriptionStatus.Valid {
		s := domain.SubscriptionStatus(m.SubscriptionStatus.String)
		e.SubscriptionStatus = &s
	}
	if m.CurrentPeriodStart.Valid {
		e.CurrentPeriodStart = &m.CurrentPeriodStart.Time
	}
	if m.CurrentPeriodEnd.Valid {
		e.CurrentPeriodEnd = &m.CurrentPeriodEnd.Time
	}
	if m.LogoURL.Valid {
		e.LogoURL = &m.LogoURL.String
	}
	if m.PrimaryColor.Valid {
		e.PrimaryColor = &m.PrimaryColor.String
	}
	if m.SecondaryColor.Valid {
		e.SecondaryColor = &m.SecondaryColor.String
	}
	if m.CustomDomain.Valid {
		e.CustomDomain = &m.CustomDomain.String
	}
	if m.ContactPhone.Valid {
		e.ContactPhone = &m.ContactPhone.String
	}
	if m.AddressLine1.Valid {
		e.AddressLine1 = &m.AddressLine1.String
	}
	if m.AddressLine2.Valid {
		e.AddressLine2 = &m.AddressLine2.String
	}
	if m.City.Valid {
		e.City = &m.City.String
	}
	if m.Country.Valid {
		e.Country = &m.Country.String
	}
	if m.RetentionUntil.Valid {
		e.RetentionUntil = &m.RetentionUntil.Time
	}

	if len(m.Settings) > 0 {
		var settings map[string]any
		if err := json.Unmarshal(m.Settings, &settings); err == nil {
			e.Settings = settings
		}
	} else {
		e.Settings = make(map[string]any)
	}

	return e
}
