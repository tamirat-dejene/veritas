#!/usr/bin/env bash
set -euo pipefail

# ANSI color codes for pretty printing
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() {
  echo -e "${BLUE}ℹ️  $1${NC}"
}

success() {
  echo -e "${GREEN}✅ $1${NC}"
}

warn() {
  echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
  echo -e "${RED}❌ $1${NC}" >&2
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_CMD=(docker-compose)
else
  error "neither 'docker compose' nor 'docker-compose' was found."
  exit 1
fi

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
elif [[ -f .env.example ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env.example
  set +a
fi

PG_VERITAS_USER="${PG_VERITAS_USER:-postgres}"
PG_VERITAS_CORE_DB="${PG_VERITAS_CORE_DB:-veritas_core}"

UP_DIRS=(
  "services/enterprise-service/migrations"
  "services/auth-service/migrations"
  "services/exam-service/migrations"
  "services/candidate-service/migrations"
  "services/payment-service/migrations"
)

DOWN_DIRS=(
  "services/payment-service/migrations"
  "services/candidate-service/migrations"
  "services/exam-service/migrations"
  "services/auth-service/migrations"
  "services/enterprise-service/migrations"
)

usage() {
  cat <<'EOF'
Run database migrations for all Veritas Go services.

Usage:
  scripts/migrate.sh up       # apply all *.up.sql files
  scripts/migrate.sh down     # apply all *.down.sql files (reverse order)
  scripts/migrate.sh reset    # down then up

Notes:
  - Starts postgres service automatically if needed.
  - Ensures target database exists.
  - Ensures pgcrypto extension exists (required for gen_random_uuid()).
EOF
}

ensure_postgres_ready() {
  info "Starting postgres container (if not already running)..."
  "${COMPOSE_CMD[@]}" up -d postgres >/dev/null

  info "Waiting for PostgreSQL to be ready..."
  local retries=30
  local count=0
  local wait=2

  while ! "${COMPOSE_CMD[@]}" exec -T postgres pg_isready -U "$PG_VERITAS_USER" >/dev/null 2>&1; do
    count=$((count + 1))
    if [ $count -ge $retries ]; then
      error "PostgreSQL did not become ready in time."
      exit 1
    fi
    echo -n "."
    sleep "$wait"
  done
  echo ""
  success "PostgreSQL is ready!"
}

psql_exec() {
  "${COMPOSE_CMD[@]}" exec -T postgres psql -U "$PG_VERITAS_USER" "$@"
}

ensure_database() {
  local exists
  exists="$(psql_exec -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='${PG_VERITAS_CORE_DB}'")"
  if [[ "$exists" != "1" ]]; then
    info "Creating database '${PG_VERITAS_CORE_DB}'..."
    psql_exec -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE \"${PG_VERITAS_CORE_DB}\";"
    success "Database '${PG_VERITAS_CORE_DB}' created."
  else
    success "Database '${PG_VERITAS_CORE_DB}' exists."
  fi
}

ensure_extensions() {
  info "Ensuring pgcrypto extension exists..."
  psql_exec -d "$PG_VERITAS_CORE_DB" -v ON_ERROR_STOP=1 -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"
  success "pgcrypto extension is ready."
}

apply_sql_file() {
  local file="$1"
  info "Applying ${file}"
  if psql_exec -d "$PG_VERITAS_CORE_DB" -v ON_ERROR_STOP=1 -f - < "$file"; then
    success "Successfully applied $(basename "$file")"
  else
    error "Failed to apply $(basename "$file")"
    exit 1
  fi
}

run_up() {
  info "Running UP migrations..."
  local dir file
  for dir in "${UP_DIRS[@]}"; do
    if [[ -d "$dir" ]]; then
      info "Directory: $dir"
      while IFS= read -r file; do
        if [[ -n "$file" ]]; then
          apply_sql_file "$file"
        fi
      done < <(find "$dir" -maxdepth 1 -type f -name '*.up.sql' | sort)
    else
      warn "Migration directory not found: $dir"
    fi
  done
  success "UP migrations completed."
}

run_down() {
  warn "Running DOWN migrations..."
  local dir file
  for dir in "${DOWN_DIRS[@]}"; do
    if [[ -d "$dir" ]]; then
      info "Directory: $dir"
      while IFS= read -r file; do
        if [[ -n "$file" ]]; then
          apply_sql_file "$file"
        fi
      done < <(find "$dir" -maxdepth 1 -type f -name '*.down.sql' | sort -r)
    else
      warn "Migration directory not found: $dir"
    fi
  done
  success "DOWN migrations completed."
}

main() {
  local action="${1:-}"
  if [[ -z "$action" ]]; then
    usage
    exit 1
  fi

  case "$action" in
    up)
      ensure_postgres_ready
      ensure_database
      ensure_extensions
      run_up
      ;;
    down)
      ensure_postgres_ready
      ensure_database
      run_down
      ;;
    reset)
      ensure_postgres_ready
      ensure_database
      run_down
      ensure_extensions
      run_up
      ;;
    -h|--help|help)
      usage
      ;;
    *)
      error "Unknown action: ${action}"
      usage
      exit 1
      ;;
  esac
}

main "$@"
