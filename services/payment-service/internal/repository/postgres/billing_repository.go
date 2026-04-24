package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
)

const (
	invoiceFields = "id, enterprise_id, subscription_id, number, status, amount_due, amount_paid, amount_remaining, currency, due_date, paid_at, hosted_invoice_url, invoice_pdf_url, created_at, updated_at"
	paymentFields = "id, enterprise_id, invoice_id, amount, currency, status, payment_method_type, provider, provider_payment_id, provider_error_code, provider_error_message, created_at"
)

type billingRepository struct {
	db DBTX
}

func NewBillingRepository(db DBTX) domain.BillingRepository {
	return &billingRepository{db: db}
}

func (r *billingRepository) CreateInvoice(ctx context.Context, i *domain.Invoice) error {
	query := `
		INSERT INTO veritas_invoices (
			id, enterprise_id, subscription_id, number, status, amount_due, amount_paid, 
			amount_remaining, currency, due_date, paid_at, hosted_invoice_url, 
			invoice_pdf_url, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	now := time.Now()
	i.CreatedAt = now
	i.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		i.ID, i.EnterpriseID, i.SubscriptionID, i.Number, i.Status, i.AmountDue, i.AmountPaid,
		i.AmountRemaining, i.Currency, i.DueDate, i.PaidAt, i.HostedInvoiceURL,
		i.InvoicePDFURL, i.CreatedAt, i.UpdatedAt,
	)
	return err
}

func (r *billingRepository) GetInvoiceByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_invoices WHERE id = $1", invoiceFields)
	var i domain.Invoice
	err := r.db.QueryRow(ctx, query, id).Scan(
		&i.ID, &i.EnterpriseID, &i.SubscriptionID, &i.Number, &i.Status, &i.AmountDue, &i.AmountPaid, &i.AmountRemaining, &i.Currency, &i.DueDate, &i.PaidAt, &i.HostedInvoiceURL, &i.InvoicePDFURL, &i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, err
	}
	return &i, nil
}

func (r *billingRepository) GetInvoiceByNumber(ctx context.Context, number string) (*domain.Invoice, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_invoices WHERE number = $1", invoiceFields)
	var i domain.Invoice
	err := r.db.QueryRow(ctx, query, number).Scan(
		&i.ID, &i.EnterpriseID, &i.SubscriptionID, &i.Number, &i.Status, &i.AmountDue, &i.AmountPaid, &i.AmountRemaining, &i.Currency, &i.DueDate, &i.PaidAt, &i.HostedInvoiceURL, &i.InvoicePDFURL, &i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, err
	}
	return &i, nil
}

func (r *billingRepository) ListInvoicesByEnterprise(ctx context.Context, enterpriseID uuid.UUID) ([]*domain.Invoice, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_invoices WHERE enterprise_id = $1 ORDER BY created_at DESC", invoiceFields)
	rows, err := r.db.Query(ctx, query, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*domain.Invoice
	for rows.Next() {
		var i domain.Invoice
		err := rows.Scan(
			&i.ID, &i.EnterpriseID, &i.SubscriptionID, &i.Number, &i.Status, &i.AmountDue, &i.AmountPaid, &i.AmountRemaining, &i.Currency, &i.DueDate, &i.PaidAt, &i.HostedInvoiceURL, &i.InvoicePDFURL, &i.CreatedAt, &i.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, &i)
	}
	return invoices, nil
}

func (r *billingRepository) UpdateInvoice(ctx context.Context, i *domain.Invoice) error {
	query := `
		UPDATE veritas_invoices SET
			status = $1, amount_paid = $2, amount_remaining = $3, paid_at = $4, updated_at = $5
		WHERE id = $6
	`
	i.UpdatedAt = time.Now()
	_, err := r.db.Exec(ctx, query,
		i.Status, i.AmountPaid, i.AmountRemaining, i.PaidAt, i.UpdatedAt, i.ID,
	)
	return err
}

func (r *billingRepository) WithTx(tx pgx.Tx) domain.BillingRepository {
	return &billingRepository{db: tx}
}

func (r *billingRepository) CreatePayment(ctx context.Context, p *domain.Payment) error {
	query := `
		INSERT INTO veritas_payments (
			id, enterprise_id, invoice_id, amount, currency, status, payment_method_type,
			provider, provider_payment_id, provider_error_code, provider_error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}

	_, err := r.db.Exec(ctx, query,
		p.ID, p.EnterpriseID, p.InvoiceID, p.Amount, p.Currency, p.Status, p.PaymentMethodType,
		p.Provider, p.ProviderPaymentID, p.ProviderErrorCode, p.ProviderErrorMessage, p.CreatedAt,
	)
	return err
}

func (r *billingRepository) ListPaymentsByEnterprise(ctx context.Context, enterpriseID uuid.UUID) ([]*domain.Payment, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_payments WHERE enterprise_id = $1 ORDER BY created_at DESC", paymentFields)
	rows, err := r.db.Query(ctx, query, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		err := rows.Scan(
			&p.ID, &p.EnterpriseID, &p.InvoiceID, &p.Amount, &p.Currency, &p.Status, &p.PaymentMethodType, &p.Provider, &p.ProviderPaymentID, &p.ProviderErrorCode, &p.ProviderErrorMessage, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		payments = append(payments, &p)
	}
	return payments, nil
}

func (r *billingRepository) RecordEventProcessed(ctx context.Context, eventID string, eventType string) error {
	query := `
		INSERT INTO veritas_processed_webhook_events (event_id, event_type)
		VALUES ($1, $2)
	`
	_, err := r.db.Exec(ctx, query, eventID, eventType)
	return err
}

func (r *billingRepository) HasEventBeenProcessed(ctx context.Context, eventID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM veritas_processed_webhook_events WHERE event_id = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, eventID).Scan(&exists)
	return exists, err
}
