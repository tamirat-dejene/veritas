#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Create a Go service folder structure (directories + go.mod + Dockerfile)."
  echo
  echo "Usage: $(basename "$0") <service-name>"
  echo "Example: $(basename "$0") payment-service"
}

service_name=""
if [[ $# -eq 0 ]]; then
  usage
  echo
  read -r -p "Service name: " service_name
elif [[ $# -eq 1 ]]; then
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    *)
      service_name="$1"
      ;;
  esac
else
  usage >&2
  exit 1
fi

if [[ -z "$service_name" ]]; then
  echo "Error: service name is required." >&2
  exit 1
fi
service_root="services/${service_name}"

if [[ -e "$service_root" ]]; then
  echo "Error: '$service_root' already exists." >&2
  exit 1
fi

mkdir -p \
  "$service_root/cmd/server" \
  "$service_root/internal/config" \
  "$service_root/internal/domain" \
  "$service_root/internal/handler" \
  "$service_root/internal/infrastructure" \
  "$service_root/internal/middleware" \
  "$service_root/internal/repository" \
  "$service_root/internal/router" \
  "$service_root/internal/usecase" \
  "$service_root/migrations"

cat > "$service_root/go.mod" <<EOF
module github.com/tamirat-dejene/veritas/services/${service_name}

go 1.25.1

require github.com/tamirat-dejene/veritas/shared v0.0.0

replace github.com/tamirat-dejene/veritas/shared => ../../shared
EOF

touch "$service_root/go.sum"

cat > "$service_root/Dockerfile" <<EOF
FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Install git (sometimes needed for private deps)
RUN apk add --no-cache git

# Copy only go.mod/go.sum first (better layer caching)
COPY shared/go.mod shared/go.sum /app/shared/
COPY services/${service_name}/go.mod services/${service_name}/go.sum /app/services/${service_name}/

WORKDIR /app/services/${service_name}

# Cache Go modules
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

# Copy source
COPY services/${service_name} /app/services/${service_name}
COPY shared /app/shared

# Cache Go build files
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux go build \
  -trimpath -ldflags="-s -w" \
  -o /app/bin/${service_name} ./cmd/server

# Final stage
FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
  && adduser -D -u 10001 appuser \
  && mkdir -p /app \
  && chown -R appuser:appuser /app

WORKDIR /app
COPY --from=builder /app/bin/${service_name} ./${service_name}
USER appuser
EXPOSE 8080
CMD ["./${service_name}"]
EOF

echo "Created Go service folders under $service_root"
echo "Initialized go.mod at $service_root/go.mod"
echo "Initialized Dockerfile at $service_root/Dockerfile"