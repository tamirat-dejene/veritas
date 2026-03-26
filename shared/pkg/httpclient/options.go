package httpclient

import (
	"net/http"
	"time"
)

// RequestOption defines a function type to configure an HTTP request.
type RequestOption func(*httpRequest)

type httpRequest struct {
	headers http.Header
	query   map[string]string
	timeout time.Duration
}

func newHTTPRequest() *httpRequest {
	return &httpRequest{
		headers: make(http.Header),
		query:   make(map[string]string),
	}
}

// WithHeader sets a header on the request.
func WithHeader(key, value string) RequestOption {
	return func(r *httpRequest) {
		r.headers.Set(key, value)
	}
}

// WithQuery sets a query parameter on the request.
func WithQuery(key, value string) RequestOption {
	return func(r *httpRequest) {
		r.query[key] = value
	}
}

// WithTimeout sets a custom timeout for the request.
func WithTimeout(d time.Duration) RequestOption {
	return func(r *httpRequest) {
		r.timeout = d
	}
}

// applyOptions applies all options to the httpRequest.
func (r *httpRequest) applyOptions(opts []RequestOption) {
	for _, opt := range opts {
		opt(r)
	}
}
