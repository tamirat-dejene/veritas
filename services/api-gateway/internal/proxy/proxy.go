package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
	"go.uber.org/zap"
)

// Proxy wraps a reverse proxy with circuit breaker protection.
type Proxy struct {
	reverseProxy   *httputil.ReverseProxy
	circuitBreaker domain.CircuitBreaker
	serviceName    string
	targetURL      string
	mu             sync.RWMutex
	lastError      error
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

	// Set custom error handler
	reverseProxy.ErrorHandler = proxy.errorHandler

	return proxy, nil
}

// ServeHTTP implements http.Handler with circuit breaker protection.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Execute request through circuit breaker
	err := p.circuitBreaker.Execute(func() error {
		// Use a custom response writer to capture errors
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		p.reverseProxy.ServeHTTP(rw, r)

		// Consider 5xx responses as failures for circuit breaker
		if rw.statusCode >= 500 {
			return fmt.Errorf("backend returned %d", rw.statusCode)
		}

		return nil
	})

	if err != nil {
		// Circuit breaker is open or request failed
		p.mu.Lock()
		p.lastError = err
		p.mu.Unlock()

		zap.L().Warn(
			"Circuit breaker triggered",
			zap.String("service", p.serviceName),
			zap.String("state", p.circuitBreaker.State()),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)

		// Return 503 Service Unavailable
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"error":"Service temporarily unavailable","service":"%s","state":"%s"}`,
			p.serviceName, p.circuitBreaker.State())
	}
}

// errorHandler handles errors from the reverse proxy.
func (p *Proxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	p.mu.Lock()
	p.lastError = err
	p.mu.Unlock()

	zap.L().Error(
		"Proxy error",
		zap.String("service", p.serviceName),
		zap.String("path", r.URL.Path),
		zap.Error(err),
	)

	// This error will be caught by the circuit breaker
	// Return 502 Bad Gateway
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	fmt.Fprintf(w, `{"error":"Bad Gateway","service":"%s"}`, p.serviceName)
}

// responseWriter wraps http.ResponseWriter to capture status codes.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}
