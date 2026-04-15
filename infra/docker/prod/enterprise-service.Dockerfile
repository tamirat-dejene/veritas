# syntax=docker/dockerfile:1.7

FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY shared/go.mod shared/go.sum /app/shared/
COPY services/enterprise-service/go.mod services/enterprise-service/go.sum /app/services/enterprise-service/

WORKDIR /app/services/enterprise-service

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY services/enterprise-service /app/services/enterprise-service
COPY shared /app/shared

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /app/bin/enterprise-service ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 appuser \
    && mkdir -p /app \
    && chown -R appuser:appuser /app

WORKDIR /app

COPY --from=builder /app/bin/enterprise-service ./enterprise-service

USER appuser

EXPOSE 8080

CMD ["./enterprise-service"]
