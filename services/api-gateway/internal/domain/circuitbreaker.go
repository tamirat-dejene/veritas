package domain

// CircuitBreaker defines the interface for circuit breaker functionality.
// It follows the circuit breaker pattern to prevent cascading failures.
type CircuitBreaker interface {
	// Execute runs the given function with circuit breaker protection.
	// Returns an error if the circuit is open or if the function fails.
	Execute(func() error) error

	// State returns the current state of the circuit breaker.
	// Possible values: "closed", "open", "half-open"
	State() string

	// Name returns the name of the circuit breaker (typically the service name).
	Name() string
}
