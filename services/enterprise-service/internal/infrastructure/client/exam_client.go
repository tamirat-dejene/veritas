package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/httpclient"
)

type httpExamClient struct {
	client httpclient.Client
}

func NewExamClient(baseURL string) domain.ExamClient {
	return &httpExamClient{
		client: httpclient.New(httpclient.Config{
			BaseURL: baseURL,
			Timeout: 5 * time.Second,
		}),
	}
}

func (c *httpExamClient) GetActiveExamsCount(ctx context.Context, enterpriseID uuid.UUID) (int, error) {
	path := fmt.Sprintf("/internal/enterprises/%s/counts", enterpriseID)

	resp, err := c.client.Get(ctx, path)
	if err != nil {
		return 0, err
	}

	if err := resp.Error(); err != nil {
		return 0, err
	}

	var res struct {
		ActiveExamCount int `json:"active_exam_count"`
	}
	if err := resp.Decode(&res); err != nil {
		return 0, err
	}

	return res.ActiveExamCount, nil
}
