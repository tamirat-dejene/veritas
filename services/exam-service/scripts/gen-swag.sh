#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR"

SWAG_BIN="swag"
if ! command -v swag >/dev/null 2>&1; then
  if [ -f "$(go env GOPATH)/bin/swag" ]; then
    SWAG_BIN="$(go env GOPATH)/bin/swag"
  else
    SWAG_BIN="go run github.com/swaggo/swag/cmd/swag@v1.16.6"
  fi
fi

$SWAG_BIN init -g cmd/server/main.go -o docs/swagger --parseDependency --parseInternal

# Swaggo emits package-qualified schema names. Normalize to clean model names.
PREFIX_DOMAIN="github_com_tamirat-dejene_veritas_services_exam-service_internal_domain."
PREFIX_HTTP="internal_handler."

for f in docs/swagger/docs.go docs/swagger/swagger.json docs/swagger/swagger.yaml; do
  sed -i "s/${PREFIX_DOMAIN}//g" "$f"
  sed -i "s/${PREFIX_HTTP}//g" "$f"
done

echo "Generated Swagger docs in docs/swagger"
