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

echo "Generated Swagger docs in docs/swagger"
