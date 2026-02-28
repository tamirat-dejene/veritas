package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
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
	GetExamMetadata(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID) (*ExamMetadata, error)
	GetExamQuestions(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID) ([]QuestionSnapshot, error)
}

type examServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewExamServiceClient(baseURL string) ExamServiceClient {
	return &examServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *examServiceClient) GetExamMetadata(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID) (*ExamMetadata, error) {
	url := fmt.Sprintf("%s/api/v1/exams/%s", c.baseURL, examID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Internal headers for service-to-service auth/tenancy mapping
	req.Header.Set("X-Enterprise-Id", enterpriseID.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exam-service returned status: %d", resp.StatusCode)
	}

	var response struct {
		Data ExamMetadata `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
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

func (c *examServiceClient) GetExamQuestions(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID) ([]QuestionSnapshot, error) {
	url := fmt.Sprintf("%s/api/v1/exams/%s/questions", c.baseURL, examID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Enterprise-Id", enterpriseID.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exam-service returned status: %d", resp.StatusCode)
	}

	var response struct {
		Data []rawExamQuestion `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
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
			ID:             raw.QuestionID, // We use the actual QuestionID here to map candidate answers
			Content:        raw.Question,
			PointsOverride: raw.PointsOverride,
			OrderIndex:     raw.OrderIndex,
			Points:         qData.Points,
			NegativePoints: qData.NegativePoints,
		})
	}

	return snapshots, nil
}
