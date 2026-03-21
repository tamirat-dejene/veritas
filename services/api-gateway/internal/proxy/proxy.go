package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
	"go.uber.org/zap"
)

// Proxy wraps a reverse proxy with circuit breaker protection.
type Proxy struct {
	reverseProxy   *httputil.ReverseProxy
	circuitBreaker domain.CircuitBreaker
	serviceName    string
	targetURL      string
}

// NewProxy creates a new Proxy with circuit breaker protection.
func NewProxy(target string, circuitBreaker domain.CircuitBreaker, serviceName string) (*Proxy, error) {
	targetURL, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL: %w", err)
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Configure the director to set forwarding headers
	originalDirector := reverseProxy.Director
	reverseProxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Set X-Forwarded headers
		clientIP := req.RemoteAddr
		if ip := req.Header.Get("X-Forwarded-For"); ip != "" {
			req.Header.Set("X-Forwarded-For", ip+", "+clientIP)
		} else {
			req.Header.Set("X-Forwarded-For", clientIP)
		}
		req.Header.Set("X-Forwarded-Host", req.Host)
		proto := "http"
		if req.TLS != nil {
			proto = "https"
		}
		req.Header.Set("X-Forwarded-Proto", proto)
		req.Host = targetURL.Host
	}

	proxy := &Proxy{
		reverseProxy:   reverseProxy,
		circuitBreaker: circuitBreaker,
		serviceName:    serviceName,
		targetURL:      target,
	}

	reverseProxy.ErrorHandler = proxy.errorHandler

	return proxy, nil
}

// ServeHTTP implements http.Handler with circuit breaker protection.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	zap.L().Debug("Proxying request", zap.String("service", p.serviceName), zap.String("path", r.URL.Path))

	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	err := p.circuitBreaker.Execute(func() error {
		p.reverseProxy.ServeHTTP(rw, r)

		if rw.statusCode >= 500 {
			return fmt.Errorf("backend returned %d", rw.statusCode)
		}

		return nil
	})

	if err != nil {
		zap.L().Warn(
			"Circuit breaker triggered or backend error",
			zap.String("service", p.serviceName),
			zap.String("state", p.circuitBreaker.State()),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)

		if rw.wroteHeader {
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.Header().Set("Retry-After", "30")
		rw.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(rw, `{"error":"Service temporarily unavailable","service":"%s","state":"%s"}`,
			p.serviceName, p.circuitBreaker.State())
	}
}

// errorHandler handles errors from the reverse proxy (connection failures etc.).
// It receives `rw` (the wrapped responseWriter) as its `w` parameter because
// ServeHTTP passes `rw` into reverseProxy.ServeHTTP, which in turn calls this handler.
// Writing through `w` here correctly sets rw.wroteHeader, preventing double-writes.
func (p *Proxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	zap.L().Error(
		"Proxy error",
		zap.String("service", p.serviceName),
		zap.String("path", r.URL.Path),
		zap.Error(err),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	fmt.Fprintf(w, `{"error":"Bad Gateway","service":"%s"}`, p.serviceName)
}

// responseWriter wraps http.ResponseWriter to capture status codes.
type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.statusCode = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}
