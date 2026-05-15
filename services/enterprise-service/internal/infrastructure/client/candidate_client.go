package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/httpclient"
)

type httpCandidateClient struct {
	client httpclient.Client
}

func NewCandidateClient(baseURL string) domain.CandidateClient {
	return &httpCandidateClient{
		client: httpclient.New(httpclient.Config{
			BaseURL: baseURL,
			Timeout: 5 * time.Second,
		}),
	}
}

func (c *httpCandidateClient) GetActiveSessionsCount(ctx context.Context, enterpriseID uuid.UUID) (int, error) {
	path := fmt.Sprintf("/internal/enterprises/%s/counts", enterpriseID)

	resp, err := c.client.Get(ctx, path)
	if err != nil {
		return 0, err
	}

	if err := resp.Error(); err != nil {
		return 0, err
	}

	var res struct {
		ActiveSessionCount int `json:"active_session_count"`
	}
	if err := resp.Decode(&res); err != nil {
		return 0, err
	}

	return res.ActiveSessionCount, nil
}
