package infrastructure

import (
	"time"

	"github.com/sony/gobreaker"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
	"go.uber.org/zap"
)

// CircuitBreakerSettings holds configuration for circuit breaker behavior.
type CircuitBreakerSettings struct {
	MaxRequests      uint32        // Max requests allowed in half-open state
	Interval         time.Duration // Time window for counting failures
	Timeout          time.Duration // Time to wait before transitioning to half-open
	FailureThreshold uint32        // Number of consecutive failures to open circuit
}

// DefaultCircuitBreakerSettings returns sensible defaults for circuit breaker.
func DefaultCircuitBreakerSettings() CircuitBreakerSettings {
	return CircuitBreakerSettings{
		MaxRequests:      3,
		Interval:         60 * time.Second,
		Timeout:          30 * time.Second,
		FailureThreshold: 5,
	}
}

// GoBreakerWrapper wraps the gobreaker library to implement our domain interface.
type GoBreakerWrapper struct {
	cb   *gobreaker.CircuitBreaker
	name string
}

// NewCircuitBreaker creates a new circuit breaker with the given name and settings.
func NewCircuitBreaker(name string, settings CircuitBreakerSettings) domain.CircuitBreaker {
	gbSettings := gobreaker.Settings{
		Name:        name,
		MaxRequests: settings.MaxRequests,
		Interval:    settings.Interval,
		Timeout:     settings.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Open circuit if consecutive failures exceed threshold
			return counts.ConsecutiveFailures >= settings.FailureThreshold
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			zap.L().Warn("Circuit breaker state changed",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	}

	return &GoBreakerWrapper{
		cb:   gobreaker.NewCircuitBreaker(gbSettings),
		name: name,
	}
}

// Execute runs the given function with circuit breaker protection.
func (w *GoBreakerWrapper) Execute(fn func() error) error {
	_, err := w.cb.Execute(func() (any, error) {
		return nil, fn()
	})
	return err
}

// State returns the current state of the circuit breaker.
func (w *GoBreakerWrapper) State() string {
	state := w.cb.State()
	switch state {
	case gobreaker.StateClosed:
		return "closed"
	case gobreaker.StateOpen:
		return "open"
	case gobreaker.StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Name returns the name of the circuit breaker.
func (w *GoBreakerWrapper) Name() string {
	return w.name
}
