module github.com/tamirat-dejene/veritas/services/candidate-service

go 1.25.1

require (
	github.com/tamirat-dejene/veritas/shared v0.0.0
	go.uber.org/zap v1.27.0
)

require go.uber.org/multierr v1.10.0 // indirect

replace github.com/tamirat-dejene/veritas/shared => ../../shared
