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

type httpExamClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewExamClient(baseURL string) domain.ExamClient {
	return &httpExamClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *httpExamClient) GetActiveExamsCount(ctx context.Context, enterpriseID uuid.UUID) (int, error) {
	url := fmt.Sprintf("%s/internal/enterprises/%s/counts", c.baseURL, enterpriseID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var res struct {
		ActiveExamCount int `json:"active_exam_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}

	return res.ActiveExamCount, nil
}
