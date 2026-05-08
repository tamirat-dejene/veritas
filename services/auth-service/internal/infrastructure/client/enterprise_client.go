package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/httpclient"
)

type userClient struct {
	client httpclient.Client
}

// NewEnterpriseServiceClient creates a new HTTP client for the enterprise user service.
// timeout configures the per-client deadline; pass 0 to use the shared default (30 s).
func NewEnterpriseServiceClient(baseURL string, timeout time.Duration) domain.EnterpriseServiceClient {
	return &userClient{
		client: httpclient.New(httpclient.Config{
			BaseURL: baseURL,
			Timeout: timeout,
		}),
	}
}

func (c *userClient) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	path := fmt.Sprintf("/internal/users?email=%s", email)

	resp, err := c.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	if err := resp.Error(); err != nil {
		return nil, mapError(err)
	}

	var user domain.User
	if err := resp.Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *userClient) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	path := fmt.Sprintf("/internal/users/%s", id.String())

	resp, err := c.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	if err := resp.Error(); err != nil {
		return nil, mapError(err)
	}

	var user domain.User
	if err := resp.Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *userClient) ListUsersByEnterprise(ctx context.Context, enterpriseID uuid.UUID) ([]uuid.UUID, error) {
	path := fmt.Sprintf("/internal/enterprises/%s/users", enterpriseID.String())

	resp, err := c.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	if err := resp.Error(); err != nil {
		return nil, mapError(err)
	}

	var userIDs []uuid.UUID
	if err := resp.Decode(&userIDs); err != nil {
		return nil, err
	}

	return userIDs, nil
}

func (c *userClient) UpdateLoginSuccess(ctx context.Context, userID uuid.UUID, ip, userAgent string) error {
	path := fmt.Sprintf("/internal/users/%s/login-success", userID.String())
	body := map[string]string{
		"ip":         ip,
		"user_agent": userAgent,
	}

	resp, err := c.client.Post(ctx, path, body)
	if err != nil {
		return err
	}

	return resp.Error()
}

func (c *userClient) UpdateLoginFailure(ctx context.Context, userID uuid.UUID, lockUntil *time.Time, failedLoginAttempts int) error {
	path := fmt.Sprintf("/internal/users/%s/login-failure", userID.String())
	body := map[string]any{
		"locked_until":          lockUntil,
		"failed_login_attempts": failedLoginAttempts,
	}

	resp, err := c.client.Post(ctx, path, body)
	if err != nil {
		return err
	}

	return resp.Error()
}

// mapError translates HTTP-layer errors into domain-recognisable sentinel values.
func mapError(err error) error {
	var httpErr *httpclient.HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
		return domain.ErrUserNotFound
	}
	return err
}
