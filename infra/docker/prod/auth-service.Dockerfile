# syntax=docker/dockerfile:1.7

FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY shared/go.mod shared/go.sum /app/shared/
COPY services/auth-service/go.mod services/auth-service/go.sum /app/services/auth-service/

WORKDIR /app/services/auth-service

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY services/auth-service /app/services/auth-service
COPY shared /app/shared

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /app/bin/auth-service ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 appuser \
    && mkdir -p /app \
    && chown -R appuser:appuser /app

WORKDIR /app

COPY --from=builder /app/bin/auth-service ./auth-service

USER appuser

EXPOSE 8080

CMD ["./auth-service"]
