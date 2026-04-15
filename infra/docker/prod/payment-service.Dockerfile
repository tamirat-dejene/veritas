# syntax=docker/dockerfile:1.7

FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY shared/go.mod shared/go.sum /app/shared/
COPY services/payment-service/go.mod services/payment-service/go.sum /app/services/payment-service/

WORKDIR /app/services/payment-service

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY services/payment-service /app/services/payment-service
COPY shared /app/shared

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /app/bin/payment-service ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 appuser \
    && mkdir -p /app \
    && chown -R appuser:appuser /app

WORKDIR /app

COPY --from=builder /app/bin/payment-service ./payment-service

USER appuser

EXPOSE 8080

CMD ["./payment-service"]
