#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Create a Go service folder structure (directories + go.mod)."
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

echo "Created Go service folders under $service_root"
echo "Initialized go.mod at $service_root/go.mod"