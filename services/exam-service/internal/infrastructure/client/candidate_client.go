package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/httpclient"
)

type candidateClient struct {
	client httpclient.Client
}

func NewCandidateClient(baseURL string) domain.CandidateClient {
	return &candidateClient{
		client: httpclient.New(httpclient.Config{BaseURL: baseURL}),
	}
}

func (c *candidateClient) GetCandidateEmailsForExam(ctx context.Context, enterpriseID, examID uuid.UUID) ([]string, error) {
	var resp struct {
		Emails []string `json:"emails"`
	}

	q := url.Values{}
	q.Set("exam_id", examID.String())
	q.Set("enterprise_id", enterpriseID.String())

	path := "/internal/candidates/emails?" + q.Encode()
	hResp, err := c.client.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("candidate_client: get emails: %w", err)
	}
	if err := hResp.Error(); err != nil {
		return nil, err
	}
	if err := hResp.Decode(&resp); err != nil {
		return nil, err
	}

	return resp.Emails, nil
}
