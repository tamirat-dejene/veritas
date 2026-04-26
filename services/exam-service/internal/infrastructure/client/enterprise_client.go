package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/httpclient"
)

type enterpriseClient struct {
	client httpclient.Client
}

func NewEnterpriseClient(baseURL string) domain.EnterpriseClient {
	return &enterpriseClient{
		client: httpclient.New(httpclient.Config{BaseURL: baseURL}),
	}
}

func (c *enterpriseClient) GetEnterpriseAdminEmail(ctx context.Context, enterpriseID uuid.UUID) (string, error) {
	var resp struct {
		ID           uuid.UUID `json:"id"`
		ContactEmail string    `json:"contactEmail"`
	}

	path := fmt.Sprintf("/enterprises/%s", enterpriseID)
	hResp, err := c.client.Get(ctx, path)
	if err != nil {
		return "", fmt.Errorf("enterprise_client: get enterprise: %w", err)
	}
	if err := hResp.Error(); err != nil {
		return "", err
	}
	if err := hResp.Decode(&resp); err != nil {
		return "", err
	}

	return resp.ContactEmail, nil
}
