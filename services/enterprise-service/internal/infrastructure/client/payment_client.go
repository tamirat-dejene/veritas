package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/httpclient"
)

// httpPaymentClient calls the payment-service over HTTP to fetch subscription data.
type httpPaymentClient struct {
	client httpclient.Client
}

// NewPaymentClient creates a PaymentClient backed by the payment-service HTTP API.
func NewPaymentClient(baseURL string) domain.PaymentClient {
	return &httpPaymentClient{
		client: httpclient.New(httpclient.Config{
			BaseURL: baseURL,
			Timeout: 10 * time.Second,
		}),
	}
}

// GetActiveSubscription fetches the active subscription for an enterprise
func (c *httpPaymentClient) GetActiveSubscription(ctx context.Context, enterpriseID uuid.UUID) (*domain.SubscriptionSnapshot, error) {
	path := fmt.Sprintf("/subscriptions/%s", enterpriseID)

	resp, err := c.client.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("paymentclient: request failed: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if err := resp.Error(); err != nil {
		return nil, fmt.Errorf("paymentclient: %w", err)
	}

	var body *domain.SubscriptionSnapshot
	if err := resp.Decode(&body); err != nil {
		return nil, fmt.Errorf("paymentclient: decode response: %w", err)
	}

	return body, nil
}
