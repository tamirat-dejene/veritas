FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Install git (sometimes needed for private deps)
RUN apk add --no-cache git

# Copy only go.mod/go.sum first (better layer caching)
COPY shared/go.mod shared/go.sum /app/shared/
COPY services/candidate-service/go.mod services/candidate-service/go.sum /app/services/candidate-service/

WORKDIR /app/services/candidate-service

# Cache Go modules
RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download

# Copy source
COPY services/candidate-service /app/services/candidate-service
COPY shared /app/shared

# Cache Go build files
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 GOOS=linux go build \
	-trimpath -ldflags="-s -w" \
	-o /app/bin/candidate-service ./cmd/server

# Final stage
FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
	&& adduser -D -u 10001 appuser \
	&& mkdir -p /app \
	&& chown -R appuser:appuser /app

WORKDIR /app
COPY --from=builder /app/bin/candidate-service ./candidate-service
USER appuser
EXPOSE 8080
CMD ["./candidate-service"]
