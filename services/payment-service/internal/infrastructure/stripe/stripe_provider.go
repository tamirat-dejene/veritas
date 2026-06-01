package stripe

import (
	"context"
	"fmt"

	stripego "github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/webhook"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
)

type stripeProvider struct {
	client        *stripego.Client
	webhookSecret string
	successURL    string
	cancelURL     string
}

// NewStripeProvider creates a Stripe implementation of domain.PaymentProvider.
func NewStripeProvider(secretKey, webhookSecret, successURL, cancelURL string) domain.PaymentProvider {
	return &stripeProvider{
		client:        stripego.NewClient(secretKey),
		webhookSecret: webhookSecret,
		successURL:    successURL,
		cancelURL:     cancelURL,
	}
}

// CreateCheckoutSession creates a Stripe Checkout Session and returns its URL.
func (p *stripeProvider) CreateCheckoutSession(ctx context.Context, req domain.CheckoutRequest) (string, error) {
	params := &stripego.CheckoutSessionCreateParams{
		SuccessURL: stripego.String(p.successURL),
		CancelURL:  stripego.String(p.cancelURL),
		Mode:       stripego.String(string(stripego.CheckoutSessionModeSubscription)),
		LineItems: []*stripego.CheckoutSessionCreateLineItemParams{
			{
				Price:    stripego.String(req.Plan.StripePriceID),
				Quantity: stripego.Int64(1),
			},
		},
		Params: stripego.Params{
			Metadata: map[string]string{
				"enterprise_id": req.EnterpriseID.String(),
				"plan_id":       req.Plan.ID.String(),
			},
		},
	}
	if req.CustomerRef != nil && *req.CustomerRef != "" {
		params.Customer = stripego.String(*req.CustomerRef)
	}

	s, err := p.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return "", fmt.Errorf("stripe: create checkout session: %w", err)
	}

	return s.URL, nil
}

// VerifyWebhookEvent validates the Stripe-Signature header and maps the event
// to the provider-agnostic domain.PaymentEvent.
func (p *stripeProvider) VerifyWebhookEvent(payload []byte, sigHeader string) (*domain.PaymentEvent, error) {
	event, err := webhook.ConstructEvent(payload, sigHeader, p.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("stripe: construct event: %w", err)
	}

	pe := &domain.PaymentEvent{
		EventID:  event.ID,
		EventType: string(event.Type),
		Raw:      event.Data.Object,
	}

	// Extract common fields where available.
	switch event.Type {
	case "checkout.session.completed":
		if meta, ok := event.Data.Object["metadata"].(map[string]any); ok {
			pe.TxRef, _ = meta["enterprise_id"].(string) // used for idempotency key
		}
		pe.CustomerRef, _ = event.Data.Object["customer"].(string)
		pe.SubscriptionRef, _ = event.Data.Object["subscription"].(string)
	case "invoice.paid", "invoice.payment_failed", "invoice.upcoming":
		pe.SubscriptionRef, _ = event.Data.Object["subscription"].(string)
	}

	return pe, nil
}

// CancelSubscription cancels a Stripe subscription.
// When cancelAtPeriodEnd is true, the subscription stays active until billing period ends.
func (p *stripeProvider) CancelSubscription(ctx context.Context, providerSubID string, cancelAtPeriodEnd bool) error {
	if cancelAtPeriodEnd {
		params := &stripego.SubscriptionUpdateParams{
			CancelAtPeriodEnd: stripego.Bool(true),
		}
		_, err := p.client.V1Subscriptions.Update(ctx, providerSubID, params)
		if err != nil {
			return fmt.Errorf("stripe: schedule cancel at period end: %w", err)
		}
		return nil
	}
	_, err := p.client.V1Subscriptions.Cancel(ctx, providerSubID, nil)
	if err != nil {
		return fmt.Errorf("stripe: cancel subscription: %w", err)
	}
	return nil
}

// ReactivateSubscription removes a pending period-end cancellation.
func (p *stripeProvider) ReactivateSubscription(ctx context.Context, providerSubID string) error {
	params := &stripego.SubscriptionUpdateParams{
		CancelAtPeriodEnd: stripego.Bool(false),
	}
	_, err := p.client.V1Subscriptions.Update(ctx, providerSubID, params)
	if err != nil {
		return fmt.Errorf("stripe: reactivate subscription: %w", err)
	}
	return nil
}

// SyncPlan creates a new Stripe Price for the plan and returns its ID.
func (p *stripeProvider) SyncPlan(ctx context.Context, plan *domain.SubscriptionPlan) (string, error) {
	unitAmount := int64(plan.Price * 100)

	params := &stripego.PriceCreateParams{
		UnitAmount: stripego.Int64(unitAmount),
		Currency:   stripego.String(string(plan.Currency)),
		Recurring: &stripego.PriceCreateRecurringParams{
			Interval: stripego.String(string(plan.BillingCycle)),
		},
		ProductData: &stripego.PriceCreateProductDataParams{
			Name: stripego.String(plan.Name),
			Metadata: map[string]string{
				"plan_slug": plan.Slug,
			},
		},
	}

	newPrice, err := p.client.V1Prices.Create(ctx, params)
	if err != nil {
		return "", fmt.Errorf("stripe: create price: %w", err)
	}

	return newPrice.ID, nil
}

// DeactivatePlan marks a Stripe Price as inactive.
func (p *stripeProvider) DeactivatePlan(ctx context.Context, providerPriceID string) error {
	params := &stripego.PriceUpdateParams{
		Active: stripego.Bool(false),
	}
	_, err := p.client.V1Prices.Update(ctx, providerPriceID, params)
	if err != nil {
		return fmt.Errorf("stripe: deactivate price: %w", err)
	}
	return nil
}

// RefundPayment refunds a Stripe payment intent.
// Stripe expects the refund amount in cents.
func (p *stripeProvider) RefundPayment(ctx context.Context, providerPaymentID string, amount float64) error {
	amountCents := int64(amount * 100)

	params := &stripego.RefundCreateParams{
		PaymentIntent: stripego.String(providerPaymentID),
		Amount:        stripego.Int64(amountCents),
	}

	_, err := p.client.V1Refunds.Create(ctx, params)
	if err != nil {
		return fmt.Errorf("stripe: refund payment: %w", err)
	}

	return nil
}
