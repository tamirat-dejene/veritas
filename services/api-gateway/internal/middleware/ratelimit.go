package middleware

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// IPRateLimiter implementation
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}
}

func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter := rate.NewLimiter(i.r, i.b)
	i.ips[ip] = limiter
	return limiter
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	limiter, exists := i.ips[ip]
	i.mu.Unlock()

	if !exists {
		return i.AddIP(ip)
	}

	return limiter
}

func RateLimit(limit rate.Limit, burst int) func(http.Handler) http.Handler {
	limiter := NewIPRateLimiter(limit, burst)

	// Background goroutine to cleanup old entries could be added here to avoid memory leak
	// For this implementations, we'll keep it simple.

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr // Setup valid IP extraction (X-Forwarded-For etc) if behind proxy
			// Simple implementation using RemoteAddr

			if !limiter.GetLimiter(ip).Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
