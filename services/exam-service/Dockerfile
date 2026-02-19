FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Install git (sometimes needed for private deps)
RUN apk add --no-cache git

# Copy only go.mod/go.sum first (better layer caching)
COPY shared/go.mod shared/go.sum /app/shared/
COPY services/exam-service/go.mod services/exam-service/go.sum /app/services/exam-service/

WORKDIR /app/services/exam-service

# Cache Go modules
RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download

# Copy source
COPY services/exam-service /app/services/exam-service
COPY shared /app/shared

# Cache Go build files
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 GOOS=linux go build \
	-trimpath -ldflags="-s -w" \
	-o /app/bin/exam-service ./cmd/server

# Final stage
FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
	&& adduser -D -u 10001 appuser \
	&& mkdir -p /app \
	&& chown -R appuser:appuser /app

WORKDIR /app
COPY --from=builder /app/bin/exam-service ./exam-service
USER appuser
EXPOSE 8080
CMD ["./exam-service"]
