package httpclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Response wraps an http.Response and provides helper methods.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       io.ReadCloser
}

// Decode decodes the response body into the given value.
// It also closes the body.
func (r *Response) Decode(v any) error {
	defer r.Body.Close()

	if v == nil {
		return nil
	}

	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to decode response body: %w", err)
	}

	return nil
}

// Error returns an error if the status code is not 2xx.
func (r *Response) Error() error {
	if r.StatusCode >= 200 && r.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	return &HTTPError{
		StatusCode: r.StatusCode,
		Body:       string(body),
	}
}

// HTTPError represents an error returned by the server.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http error: status=%d body=%s", e.StatusCode, e.Body)
}
