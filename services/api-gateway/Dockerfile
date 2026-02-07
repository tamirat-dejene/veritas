FROM golang:1.25.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/bin/api-gateway ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
	&& adduser -D -u 10001 appuser \
	&& mkdir -p /app \
	&& chown -R appuser:appuser /app

WORKDIR /app

COPY --from=builder /app/bin/api-gateway ./api-gateway

USER appuser

EXPOSE 8080

CMD ["./api-gateway"]