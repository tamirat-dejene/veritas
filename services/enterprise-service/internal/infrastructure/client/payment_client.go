package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
)

// httpPaymentClient calls the payment-service over HTTP to fetch subscription data.
type httpPaymentClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewPaymentClient creates a PaymentClient backed by the payment-service HTTP API.
func NewPaymentClient(baseURL string) domain.PaymentClient {
	return &httpPaymentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetActiveSubscription fetches the active subscription for an enterprise
func (c *httpPaymentClient) GetActiveSubscription(ctx context.Context, enterpriseID uuid.UUID) (*domain.SubscriptionSnapshot, error) {
	url := fmt.Sprintf("%s/subscriptions/%s", c.baseURL, enterpriseID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("paymentclient: build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("paymentclient: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("paymentclient: unexpected status %d for enterprise %s", resp.StatusCode, enterpriseID)
	}

	var body *domain.SubscriptionSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("paymentclient: decode response: %w", err)
	}

	return body, nil
}
