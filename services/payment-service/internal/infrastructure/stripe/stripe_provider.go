package stripe

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	stripego "github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/checkout/session"
	"github.com/stripe/stripe-go/v74/price"
	"github.com/stripe/stripe-go/v74/refund"
	stripesubscription "github.com/stripe/stripe-go/v74/subscription"
	"github.com/stripe/stripe-go/v74/webhook"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
)

type stripeProvider struct {
	secretKey     string
	webhookSecret string
	successURL    string
	cancelURL     string
}

func NewStripeProvider(secretKey, webhookSecret, successURL, cancelURL string) domain.PaymentProvider {
	stripego.Key = secretKey
	return &stripeProvider{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		successURL:    successURL,
		cancelURL:     cancelURL,
	}
}

func (p *stripeProvider) CreateCheckoutSession(ctx context.Context, enterpriseID uuid.UUID, plan *domain.SubscriptionPlan, stripeCustomerID *string) (string, error) {
	params := &stripego.CheckoutSessionParams{
		SuccessURL: stripego.String(p.successURL),
		CancelURL:  stripego.String(p.cancelURL),
		Mode:       stripego.String(string(stripego.CheckoutSessionModeSubscription)),
		LineItems: []*stripego.CheckoutSessionLineItemParams{
			{
				Price:    stripego.String(plan.StripePriceID),
				Quantity: stripego.Int64(1),
			},
		},
		Params: stripego.Params{
			Metadata: map[string]string{
				"enterprise_id": enterpriseID.String(),
				"plan_id":       plan.ID.String(),
			},
		},
	}
	if stripeCustomerID != nil && *stripeCustomerID != "" {
		params.Customer = stripego.String(*stripeCustomerID)
	}

	s, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create stripe session: %w", err)
	}

	return s.URL, nil
}

func (p *stripeProvider) ConstructEvent(payload []byte, sigHeader string) (any, error) {
	event, err := webhook.ConstructEvent(payload, sigHeader, p.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to construct stripe event: %w", err)
	}
	return &event, nil
}

// CancelStripeSubscription cancels a Stripe subscription.
// When cancelAtPeriodEnd is true, the subscription stays active until billing period ends.
func (p *stripeProvider) CancelStripeSubscription(_ context.Context, stripeSubscriptionID string, cancelAtPeriodEnd bool) error {
	if cancelAtPeriodEnd {
		params := &stripego.SubscriptionParams{
			CancelAtPeriodEnd: stripego.Bool(true),
		}
		_, err := stripesubscription.Update(stripeSubscriptionID, params)
		if err != nil {
			return fmt.Errorf("stripe: schedule cancel at period end: %w", err)
		}
		return nil
	}
	_, err := stripesubscription.Cancel(stripeSubscriptionID, nil)
	if err != nil {
		return fmt.Errorf("stripe: cancel subscription: %w", err)
	}
	return nil
}

// ReactivateStripeSubscription removes a pending period-end cancellation.
func (p *stripeProvider) ReactivateStripeSubscription(_ context.Context, stripeSubscriptionID string) error {
	params := &stripego.SubscriptionParams{
		CancelAtPeriodEnd: stripego.Bool(false),
	}
	_, err := stripesubscription.Update(stripeSubscriptionID, params)
	if err != nil {
		return fmt.Errorf("stripe: reactivate subscription: %w", err)
	}
	return nil
}

// RefundStripePayment refunds a Stripe payment (Charge or PaymentIntent).
func (p *stripeProvider) RefundStripePayment(_ context.Context, stripePaymentID string, amount float64) error {
	// Stripe expects the refund amount in cents
	amountCents := int64(amount * 100)

	params := &stripego.RefundParams{
		PaymentIntent: stripego.String(stripePaymentID),
		Amount:        stripego.Int64(amountCents),
	}

	_, err := refund.New(params)
	if err != nil {
		return fmt.Errorf("stripe: refund payment: %w", err)
	}

	return nil
}

func (p *stripeProvider) SyncPlanToStripe(_ context.Context, plan *domain.SubscriptionPlan) (string, error) {
	unitAmount := int64(plan.Price * 100)

	params := &stripego.PriceParams{
		UnitAmount: stripego.Int64(unitAmount),
		Currency:   stripego.String(string(plan.Currency)),
		Recurring: &stripego.PriceRecurringParams{
			Interval: stripego.String(string(plan.BillingCycle)),
		},
		ProductData: &stripego.PriceProductDataParams{
			Name: stripego.String(plan.Name),
			Metadata: map[string]string{
				"plan_slug": plan.Slug,
			},
		},
	}

	newPrice, err := price.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create stripe price: %w", err)
	}

	return newPrice.ID, nil
}

func (p *stripeProvider) DeactivateStripePrice(_ context.Context, stripePriceID string) error {
	params := &stripego.PriceParams{
		Active: stripego.Bool(false),
	}
	_, err := price.Update(stripePriceID, params)
	if err != nil {
		return fmt.Errorf("failed to deactivate stripe price: %w", err)
	}
	return nil
}
