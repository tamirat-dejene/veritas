FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Install git (sometimes needed for private deps)
RUN apk add --no-cache git

# Copy only go.mod first (better layer caching)
COPY go.mod go.sum ./

# Cache Go modules
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source
COPY . .

# Cache Go build files
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags="-s -w" \
    -o /app/bin/api-gateway ./cmd/server

# Final stage
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