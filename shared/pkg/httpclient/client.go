package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
)

// Client interface for HTTP communication.
type Client interface {
	Get(ctx context.Context, path string, options ...RequestOption) (*Response, error)
	Post(ctx context.Context, path string, body any, options ...RequestOption) (*Response, error)
	Put(ctx context.Context, path string, body any, options ...RequestOption) (*Response, error)
	Delete(ctx context.Context, path string, options ...RequestOption) (*Response, error)
	Patch(ctx context.Context, path string, body any, options ...RequestOption) (*Response, error)
}

type client struct {
	baseURL    string
	httpClient *http.Client
}

// Config for the HTTP client.
type Config struct {
	BaseURL string
	Timeout time.Duration
}

// New creates a new HTTP client.
func New(cfg Config) Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *client) Get(ctx context.Context, path string, options ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodGet, path, nil, options...)
}

func (c *client) Post(ctx context.Context, path string, body any, options ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodPost, path, body, options...)
}

func (c *client) Put(ctx context.Context, path string, body any, options ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodPut, path, body, options...)
}

func (c *client) Delete(ctx context.Context, path string, options ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodDelete, path, nil, options...)
}

func (c *client) Patch(ctx context.Context, path string, body any, options ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodPatch, path, body, options...)
}

func (c *client) do(ctx context.Context, method, path string, body any, options ...RequestOption) (*Response, error) {
	reqOpts := newHTTPRequest()
	reqOpts.applyOptions(options)

	fullURL := c.baseURL + path
	if len(reqOpts.query) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL: %w", err)
		}
		q := u.Query()
		for k, v := range reqOpts.query {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		fullURL = u.String()
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default content type
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Apply custom headers from options
	for k, v := range reqOpts.headers {
		for _, val := range v {
			req.Header.Add(k, val)
		}
	}

	// Propagate standard headers from context
	c.propagateHeaders(ctx, req)

	// Custom timeout per request if provided
	httpClient := c.httpClient
	if reqOpts.timeout > 0 {
		clone := *c.httpClient
		clone.Timeout = reqOpts.timeout
		httpClient = &clone
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       resp.Body,
	}, nil
}

func (c *client) propagateHeaders(ctx context.Context, req *http.Request) {
	if rid, ok := ctx.Value(logger.RequestIDKey).(string); ok && rid != "" {
		req.Header.Set("X-Request-ID", rid)
	}
	if uid, ok := ctx.Value(logger.UserIDKey).(string); ok && uid != "" {
		req.Header.Set("X-User-ID", uid)
	}
	if role, ok := ctx.Value(logger.RoleKey).(string); ok && role != "" {
		req.Header.Set("X-User-Role", role)
	}
	if eid, ok := ctx.Value(logger.EnterpriseIDKey).(string); ok && eid != "" {
		req.Header.Set("X-Enterprise-ID", eid)
	}
}
