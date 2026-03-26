package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/shared/pkg/httpclient"
)

type ExamMetadata struct {
	ID                  uuid.UUID  `json:"id"`
	Title               string     `json:"title"`
	DurationMinutes     int        `json:"durationMinutes"`
	PassingScorePercent float64    `json:"passingScorePercent"`
	Status              string     `json:"status"`
	ScheduledStart      *time.Time `json:"scheduledStart"`
	ScheduledEnd        *time.Time `json:"scheduledEnd"`
}

type QuestionSnapshot struct {
	ID             uuid.UUID       `json:"id"`
	Content        json.RawMessage `json:"question"` // The raw question blob from exam-service
	PointsOverride *int            `json:"pointsOverride"`
	OrderIndex     *int            `json:"orderIndex"`
	Points         int             `json:"points"`         // fallback if override is nil
	NegativePoints float64         `json:"negativePoints"` // raw negative points
}

type ExamServiceClient interface {
	GetExamMetadata(ctx context.Context, examID uuid.UUID) (*ExamMetadata, error)
	GetExamQuestions(ctx context.Context, examID uuid.UUID) ([]QuestionSnapshot, error)
}

type examServiceClient struct {
	client httpclient.Client
}

func NewExamServiceClient(baseURL string) ExamServiceClient {
	return &examServiceClient{
		client: httpclient.New(httpclient.Config{
			BaseURL: baseURL,
			Timeout: 10 * time.Second,
		}),
	}
}

func (c *examServiceClient) GetExamMetadata(ctx context.Context, examID uuid.UUID) (*ExamMetadata, error) {
	path := fmt.Sprintf("/api/v1/exams/%s", examID)
	resp, err := c.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	if err := resp.Error(); err != nil {
		return nil, err
	}

	var response struct {
		Data ExamMetadata `json:"data"`
	}
	if err := resp.Decode(&response); err != nil {
		return nil, err
	}

	return &response.Data, nil
}

type rawExamQuestion struct {
	ID             uuid.UUID       `json:"id"`
	QuestionID     uuid.UUID       `json:"questionId"`
	PointsOverride *int            `json:"pointsOverride"`
	OrderIndex     *int            `json:"orderIndex"`
	Question       json.RawMessage `json:"question"`
}

func (c *examServiceClient) GetExamQuestions(ctx context.Context, examID uuid.UUID) ([]QuestionSnapshot, error) {
	path := fmt.Sprintf("/api/v1/exams/%s/questions", examID)
	resp, err := c.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	if err := resp.Error(); err != nil {
		return nil, err
	}

	var response struct {
		Data []rawExamQuestion `json:"data"`
	}
	if err := resp.Decode(&response); err != nil {
		return nil, err
	}

	var snapshots []QuestionSnapshot
	for _, raw := range response.Data {
		var qData struct {
			Points         int     `json:"points"`
			NegativePoints float64 `json:"negativePoints"`
		}
		if err := json.Unmarshal(raw.Question, &qData); err != nil {
			return nil, fmt.Errorf("failed to parse question payload: %w", err)
		}

		snapshots = append(snapshots, QuestionSnapshot{
			ID:             raw.QuestionID,
			Content:        raw.Question,
			PointsOverride: raw.PointsOverride,
			OrderIndex:     raw.OrderIndex,
			Points:         qData.Points,
			NegativePoints: qData.NegativePoints,
		})
	}

	return snapshots, nil
}
