module github.com/tamirat-dejene/veritas/services/auth-service

go 1.25.1

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.11.2
	github.com/tamirat-dejene/veritas/shared v0.0.0
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.48.0
)

require go.uber.org/multierr v1.10.0 // indirect

replace github.com/tamirat-dejene/veritas/shared => ../../shared
