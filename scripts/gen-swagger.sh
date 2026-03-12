#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

SERVICES=(
  "auth-service"
  "enterprise-service"
  "exam-service"
  "candidate-service"
  "payment-service"
)

usage() {
  cat <<'EOF'
Generate Swagger docs for Veritas Go services.

Usage:
  scripts/gen-swagger.sh                 # generate for all Go services
  scripts/gen-swagger.sh all             # same as above
  scripts/gen-swagger.sh <service-name>  # generate for one service

Available services:
  - auth-service
  - enterprise-service
  - exam-service
  - candidate-service
  - payment-service
EOF
}

run_service() {
  local service="$1"
  local script_path="services/${service}/scripts/gen-swag.sh"

  if [[ ! -f "$script_path" ]]; then
    echo "Skipping ${service}: ${script_path} not found"
    return 0
  fi

  echo "==> Generating Swagger for ${service}"
  bash "$script_path"
}

run_all() {
  local service
  for service in "${SERVICES[@]}"; do
    run_service "$service"
  done
}

main() {
  local target="${1:-all}"

  case "$target" in
    all)
      run_all
      ;;
    auth-service|enterprise-service|exam-service|candidate-service|payment-service)
      run_service "$target"
      ;;
    -h|--help|help)
      usage
      ;;
    *)
      echo "Unknown service or option: ${target}" >&2
      usage
      exit 1
      ;;
  esac
}

main "$@"