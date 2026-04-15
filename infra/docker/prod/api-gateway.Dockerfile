# syntax=docker/dockerfile:1.7

FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY shared/go.mod shared/go.sum /app/shared/
COPY services/api-gateway/go.mod services/api-gateway/go.sum /app/services/api-gateway/

WORKDIR /app/services/api-gateway

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY services/api-gateway /app/services/api-gateway
COPY shared /app/shared

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /app/bin/api-gateway ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 appuser \
    && mkdir -p /app \
    && chown -R appuser:appuser /app

WORKDIR /app

COPY --from=builder /app/bin/api-gateway ./api-gateway

USER appuser

EXPOSE 8080

CMD ["./api-gateway"]
