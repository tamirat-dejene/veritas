#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_CMD=(docker-compose)
else
  echo "Error: neither 'docker compose' nor 'docker-compose' was found." >&2
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
  "${COMPOSE_CMD[@]}" up -d postgres >/dev/null
}

psql_exec() {
  "${COMPOSE_CMD[@]}" exec -T postgres psql -U "$PG_VERITAS_USER" "$@"
}

ensure_database() {
  local exists
  exists="$(psql_exec -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='${PG_VERITAS_CORE_DB}'")"
  if [[ "$exists" != "1" ]]; then
    echo "Creating database '${PG_VERITAS_CORE_DB}'..."
    psql_exec -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE \"${PG_VERITAS_CORE_DB}\";"
  fi
}

ensure_extensions() {
  psql_exec -d "$PG_VERITAS_CORE_DB" -v ON_ERROR_STOP=1 -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"
}

apply_sql_file() {
  local file="$1"
  echo "Applying ${file}"
  psql_exec -d "$PG_VERITAS_CORE_DB" -v ON_ERROR_STOP=1 -f - < "$file"
}

run_up() {
  local dir file
  for dir in "${UP_DIRS[@]}"; do
    if [[ -d "$dir" ]]; then
      while IFS= read -r file; do
        apply_sql_file "$file"
      done < <(find "$dir" -maxdepth 1 -type f -name '*.up.sql' | sort)
    fi
  done
}

run_down() {
  local dir file
  for dir in "${DOWN_DIRS[@]}"; do
    if [[ -d "$dir" ]]; then
      while IFS= read -r file; do
        apply_sql_file "$file"
      done < <(find "$dir" -maxdepth 1 -type f -name '*.down.sql' | sort -r)
    fi
  done
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
      echo "Unknown action: ${action}" >&2
      usage
      exit 1
      ;;
  esac
}

main "$@"
