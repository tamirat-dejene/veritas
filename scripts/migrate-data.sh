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

# Parse flags
PROD_MODE=false
TARGET_DB=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --prod)
      PROD_MODE=true
      shift
      ;;
    -h|--help)
      # usage will be defined below, but we declare a minimal helper here or handle it in main
      PROD_MODE=false # placeholder to bypass
      TARGET_DB="help"
      shift
      ;;
    *)
      if [[ -z "$TARGET_DB" ]]; then
        TARGET_DB="$1"
      else
        error "Unknown argument: $1"
        exit 1
      fi
      shift
      ;;
  esac
done

if [ "$PROD_MODE" = true ]; then
  ENV_FILE="${ROOT_DIR}/.env.production"
  COMPOSE_FILE="${ROOT_DIR}/docker-compose.prod.yml"
  if [[ ! -f "$COMPOSE_FILE" ]]; then
    error "production compose file not found: $COMPOSE_FILE"
    exit 1
  fi
  if [[ -f "$ENV_FILE" ]]; then
    set -a
    # shellcheck disable=SC1090
    source "$ENV_FILE"
    set +a
  else
    error "production env file not found: $ENV_FILE"
    exit 1
  fi
  COMPOSE_OPTS=(-f "$COMPOSE_FILE" --env-file "$ENV_FILE")
else
  ENV_FILE="${ROOT_DIR}/.env.dev"
  if [[ -f "$ENV_FILE" ]]; then
    set -a
    # shellcheck disable=SC1090
    source "$ENV_FILE"
    set +a
  elif [[ -f .env.example ]]; then
    set -a
    # shellcheck disable=SC1091
    source .env.example
    set +a
  fi
  COMPOSE_OPTS=()
fi

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD=(docker compose "${COMPOSE_OPTS[@]}")
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_CMD=(docker-compose "${COMPOSE_OPTS[@]}")
else
  error "neither 'docker compose' nor 'docker-compose' was found."
  exit 1
fi

SRC_DB="${POSTGRES_DB:-veritas_core}"
PG_VERITAS_USER="${PG_VERITAS_USER:-postgres}"

# Define the mapping of service databases to their target tables in the monolithic DB
declare -A DB_TABLES
DB_TABLES["$POSTGRES_ENTERPRISE_DB"]="veritas_users veritas_enterprise veritas_enterprise_audit_logs password_reset_tokens"
DB_TABLES["$POSTGRES_AUTH_DB"]="refresh_tokens"
DB_TABLES["$POSTGRES_EXAM_DB"]="veritas_questions veritas_question_options veritas_exams veritas_exam_questions"
DB_TABLES["$POSTGRES_CANDIDATE_DB"]="candidate_profiles exam_enrollments exam_sessions session_questions session_answers exam_submissions"
DB_TABLES["$POSTGRES_PAYMENT_DB"]="veritas_subscription_plans veritas_enterprise_subscriptions veritas_invoices veritas_payments veritas_processed_webhook_events"
DB_TABLES["$POSTGRES_PROCTORING_DB"]="proctoring_events proctoring_session_scores"
DB_TABLES["$POSTGRES_GRADING_DB"]="" # Add any specific grading tables if applicable

usage() {
  cat <<EOF
Migrate data from the monolithic database ($SRC_DB) to the service-specific databases.

Usage:
  scripts/migrate-data.sh [--prod] [all | <database_name>]

Options:
  --prod    Use production compose configuration (docker-compose.prod.yml and .env.production)

Example:
  scripts/migrate-data.sh all
  scripts/migrate-data.sh --prod all
  scripts/migrate-data.sh $POSTGRES_AUTH_DB
EOF
}

align_schemas() {
  local dst_db="$1"
  local table="$2"

  # Get columns from target database
  local dst_cols
  dst_cols=$("${COMPOSE_CMD[@]}" exec -T postgres psql -U "$PG_VERITAS_USER" -d "$dst_db" -tAc "
    SELECT column_name FROM information_schema.columns WHERE table_name = '$table' AND table_schema = 'public';
  " 2>/dev/null || true)

  if [[ -z "$dst_cols" ]]; then
    return
  fi

  # Get columns from source database
  local src_cols
  src_cols=$("${COMPOSE_CMD[@]}" exec -T postgres psql -U "$PG_VERITAS_USER" -d "$SRC_DB" -tAc "
    SELECT column_name FROM information_schema.columns WHERE table_name = '$table' AND table_schema = 'public';
  " 2>/dev/null || true)

  if [[ -z "$src_cols" ]]; then
    return
  fi

  # For each column in source, if not in target, drop it from source
  local col
  for col in $src_cols; do
    if [[ ! " $dst_cols " =~ "$col" ]]; then
      info "Dropping deprecated column '$col' from source table '$table' in '$SRC_DB'..."
      "${COMPOSE_CMD[@]}" exec -T postgres psql -U "$PG_VERITAS_USER" -d "$SRC_DB" -c "
        ALTER TABLE public.$table DROP COLUMN IF EXISTS \"$col\" CASCADE;
      " >/dev/null 2>&1 || true
    fi
  done
}

migrate_db() {
  local dst_db="$1"
  local tables="${DB_TABLES[$dst_db]:-}"

  if [[ -z "$tables" ]]; then
    info "No tables defined for migration to $dst_db. Skipping."
    return
  fi

  info "Migrating data to $dst_db..."

  # Align source schema with target schema dynamically
  for t in $tables; do
    align_schemas "$dst_db" "$t"
  done

  # Build table flags for pg_dump
  local table_flags=()
  for t in $tables; do
    table_flags+=("-t" "$t")
  done

  # Verify if source tables exist in source database before dumping
  local check_sql=""
  for t in $tables; do
    check_sql+="SELECT to_regclass('public.$t'); "
  done

  local exists
  exists=$("${COMPOSE_CMD[@]}" exec -T postgres psql -U "$PG_VERITAS_USER" -d "$SRC_DB" -tAc "$check_sql" 2>/dev/null || true)
  
  if [[ -z "$exists" || "$exists" == *""* && ! "$exists" =~ [a-zA-Z0-9] ]]; then
    warn "No source tables found in '$SRC_DB' for database '$dst_db'. Skipping."
    return
  fi

  # Run pg_dump and psql together using docker compose exec.
  # Use session_replication_role = 'replica' to bypass foreign key constraint ordering checks during restoration.
  # Truncate target tables first to avoid unique key / duplicate conflicts with migration seeds.
  local truncate_sql=""
  for t in $tables; do
    truncate_sql+="TRUNCATE TABLE public.$t CASCADE; "
  done

  if ("${COMPOSE_CMD[@]}" exec -T postgres bash -c "
    (
      echo \"SET session_replication_role = 'replica';\"
      echo \"$truncate_sql\"
      pg_dump -U \"$PG_VERITAS_USER\" -d \"$SRC_DB\" --data-only --inserts --column-inserts $(echo "${table_flags[@]}")
      echo \"SET session_replication_role = 'origin';\"
    ) | psql -U \"$PG_VERITAS_USER\" -d \"$dst_db\" -v ON_ERROR_STOP=1
  "); then
    success "Data successfully migrated to $dst_db."
  else
    error "Failed to migrate data to $dst_db."
    exit 1
  fi
}

main() {
  local target="$TARGET_DB"
  if [[ -z "$target" || "$target" == "help" ]]; then
    usage
    exit 0
  fi

  if [[ "$target" == "all" ]]; then
    for db in "${!DB_TABLES[@]}"; do
      migrate_db "$db"
    done
  else
    if [[ -n "${DB_TABLES[$target]+exists}" ]]; then
      migrate_db "$target"
    else
      error "Unknown database: $target"
      usage
      exit 1
    fi
  fi
}

main "$@"
