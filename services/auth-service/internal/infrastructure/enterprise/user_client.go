package enterprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
)

type userClient struct {
	baseURL string
	client  *http.Client
}

// NewUserClient creates a new HTTP client for enterprise user service.
func NewUserClient(baseURL string) domain.UserRepository {
	return &userClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *userClient) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	url := fmt.Sprintf("%s/internal/users?email=%s", c.baseURL, email)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrUserNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var user domain.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *userClient) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	url := fmt.Sprintf("%s/internal/users/%s", c.baseURL, id.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrUserNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var user domain.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *userClient) UpdateLoginSuccess(ctx context.Context, userID uuid.UUID, ip, userAgent string) error {
	url := fmt.Sprintf("%s/internal/users/%s/login-success", c.baseURL, userID.String())
	body := map[string]string{
		"ip":         ip,
		"user_agent": userAgent,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func (c *userClient) UpdateLoginFailure(ctx context.Context, userID uuid.UUID, lockUntil *time.Time, failedLoginAttempts int) error {
	url := fmt.Sprintf("%s/internal/users/%s/login-failure", c.baseURL, userID.String())
	body := map[string]any{
		"locked_until": lockUntil,
		"failed_login_attempts": failedLoginAttempts,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func (c *userClient) WithTx(tx pgx.Tx) domain.UserRepository {
	// HTTP is stateless, we cannot propagate transaction to another service via standard HTTP.
	// This is an accepted trade-off for consolidation.
	return c
}
