package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/httpclient"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
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
	path := fmt.Sprintf("/exams/%s", examID)
	resp, err := c.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	if err := resp.Error(); err != nil {
		return nil, err
	}

	var exam sdomain.Exam
	if err := resp.Decode(&exam); err != nil {
		return nil, err
	}

	return &exam, nil
}

func (c *examServiceClient) GetExamQuestions(ctx context.Context, examID uuid.UUID) ([]sdomain.ExamQuestion, error) {
	var allQuestions []sdomain.ExamQuestion
	page := 1
	limit := 100

	for {
		path := fmt.Sprintf("/exams/%s/questions?with_correct_answer=false&page=%d&limit=%d", examID, page, limit)

		resp, err := c.client.Get(ctx, path)
		if err != nil {
			return nil, err
		}

		if err := resp.Error(); err != nil {
			return nil, err
		}

		var response pagination.PaginatedResponse[sdomain.ExamQuestion]
		if err := resp.Decode(&response); err != nil {
			return nil, err
		}

		allQuestions = append(allQuestions, response.Data...)

		if !response.Metadata.HasNext {
			break
		}
		page++
	}

	return allQuestions, nil
}
