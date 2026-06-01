package providerregistry

import (
	"context"
	"errors"
	"testing"

	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
)

type mockProvider struct{}

func (m *mockProvider) CreateCheckoutSession(ctx context.Context, req domain.CheckoutRequest) (string, error) {
	return "mock-url", nil
}

func (m *mockProvider) VerifyWebhookEvent(payload []byte, sigHeader string) (*domain.PaymentEvent, error) {
	return nil, nil
}

func (m *mockProvider) CancelSubscription(ctx context.Context, providerSubID string, cancelAtPeriodEnd bool) error {
	return nil
}

func (m *mockProvider) ReactivateSubscription(ctx context.Context, providerSubID string) error {
	return nil
}

func (m *mockProvider) SyncPlan(ctx context.Context, plan *domain.SubscriptionPlan) (string, error) {
	return "mock-price-id", nil
}

func (m *mockProvider) DeactivatePlan(ctx context.Context, providerPriceID string) error {
	return nil
}

func (m *mockProvider) RefundPayment(ctx context.Context, providerPaymentID string, amount float64) error {
	return nil
}

func TestProviderRegistry(t *testing.T) {
	stripeMock := &mockProvider{}
	chapaMock := &mockProvider{}

	registry := NewProviderRegistry(stripeMock, chapaMock)

	// 1. Get stripe provider should succeed
	p, err := registry.Get(domain.PaymentProviderStripe)
	if err != nil {
		t.Fatalf("unexpected error getting stripe: %v", err)
	}
	if p != stripeMock {
		t.Error("returned provider did not match stripe mock")
	}

	// 2. Get chapa provider should succeed
	p, err = registry.Get(domain.PaymentProviderChapa)
	if err != nil {
		t.Fatalf("unexpected error getting chapa: %v", err)
	}
	if p != chapaMock {
		t.Error("returned provider did not match chapa mock")
	}

	// 3. Get invalid provider should return ErrInvalidProvider
	_, err = registry.Get("invalid")
	if err == nil {
		t.Error("expected error for invalid provider, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidProvider) {
		t.Errorf("expected error to wrap ErrInvalidProvider, got: %v", err)
	}
}
