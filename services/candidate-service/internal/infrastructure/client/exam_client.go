package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/httpclient"
)


type ExamServiceClient interface {
	GetExamMetadata(ctx context.Context, examID uuid.UUID) (*sdomain.Exam, error)
	GetExamQuestions(ctx context.Context, examID uuid.UUID) ([]sdomain.ExamQuestion, error)
}

type examServiceClient struct {
	client httpclient.Client
}

func NewExamServiceClient(baseURL string, timeout time.Duration) ExamServiceClient {
	return &examServiceClient{
		client: httpclient.New(httpclient.Config{
			BaseURL: baseURL,
			Timeout: timeout,
		}),
	}
}

func (c *examServiceClient) GetExamMetadata(ctx context.Context, examID uuid.UUID) (*sdomain.Exam, error) {
	path := fmt.Sprintf("/api/v1/exams/%s", examID)
	resp, err := c.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	if err := resp.Error(); err != nil {
		return nil, err
	}

	var response struct {
		Data sdomain.Exam `json:"data"`
	}
	if err := resp.Decode(&response); err != nil {
		return nil, err
	}

	return &response.Data, nil
}

func (c *examServiceClient) GetExamQuestions(ctx context.Context, examID uuid.UUID) ([]sdomain.ExamQuestion, error) {
	path := fmt.Sprintf("/api/v1/exams/%s/questions", examID)
	resp, err := c.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	if err := resp.Error(); err != nil {
		return nil, err
	}

	var response struct {
		Data []sdomain.ExamQuestion `json:"data"`
	}
	if err := resp.Decode(&response); err != nil {
		return nil, err
	}

	return response.Data, nil
}
