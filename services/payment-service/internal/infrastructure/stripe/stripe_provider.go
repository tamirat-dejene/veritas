package stripe

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/checkout/session"
	"github.com/stripe/stripe-go/v74/webhook"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
)

type stripeProvider struct {
	secretKey     string
	webhookSecret string
}

func NewStripeProvider(secretKey, webhookSecret string) domain.PaymentProvider {
	stripe.Key = secretKey
	return &stripeProvider{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
	}
}

func (p *stripeProvider) CreateCheckoutSession(ctx context.Context, enterpriseID uuid.UUID, plan *domain.SubscriptionPlan) (string, error) {
	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String("https://veritas.com/payment/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String("https://veritas.com/payment/cancel"),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(plan.StripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Params: stripe.Params{
			Metadata: map[string]string{
				"enterprise_id": enterpriseID.String(),
				"plan_id":       plan.ID.String(),
			},
		},
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
	return event, nil
}
