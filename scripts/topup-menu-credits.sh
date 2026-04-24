#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  topup-menu-credits.sh --email EMAIL --amount AMOUNT [options]

Options:
  --email EMAIL           User email to top up. Required.
  --amount AMOUNT         Credits amount to grant. Required.
  --host HOST             SSH host alias. Default: mix
  --container NAME        Docker container name. Default: kong-database
  --db-name NAME          PostgreSQL database name. Default: kong
  --db-user USER          PostgreSQL user. Default: kong
  --reference-id ID       Custom reference id. Optional.
  --dry-run               Query user/org/current balance only, do not insert.
  -h, --help              Show this help message.

Examples:
  ./scripts/topup-menu-credits.sh --email admin@verilocale.com --amount 1000
  ./scripts/topup-menu-credits.sh --host mix --email user@example.com --amount 500 --dry-run
EOF
}

HOST="mix"
CONTAINER="kong-database"
DB_NAME="kong"
DB_USER="kong"
EMAIL=""
AMOUNT=""
REFERENCE_ID=""
DRY_RUN="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --email)
      EMAIL="${2:-}"
      shift 2
      ;;
    --amount)
      AMOUNT="${2:-}"
      shift 2
      ;;
    --host)
      HOST="${2:-}"
      shift 2
      ;;
    --container)
      CONTAINER="${2:-}"
      shift 2
      ;;
    --db-name)
      DB_NAME="${2:-}"
      shift 2
      ;;
    --db-user)
      DB_USER="${2:-}"
      shift 2
      ;;
    --reference-id)
      REFERENCE_ID="${2:-}"
      shift 2
      ;;
    --dry-run)
      DRY_RUN="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ -z "$EMAIL" || -z "$AMOUNT" ]]; then
  echo "--email and --amount are required." >&2
  usage
  exit 1
fi

if ! [[ "$AMOUNT" =~ ^[0-9]+$ ]]; then
  echo "--amount must be a positive integer." >&2
  exit 1
fi

if [[ -z "$REFERENCE_ID" ]]; then
  SAFE_EMAIL="$(printf '%s' "$EMAIL" | tr '@.' '__' | tr -cd '[:alnum:]_')"
  REFERENCE_ID="manual_topup_${SAFE_EMAIL}_${AMOUNT}_$(date +%Y%m%d%H%M%S)"
fi

run_psql() {
  local sql="$1"
  ssh "$HOST" "docker exec $CONTAINER psql -U $DB_USER -d $DB_NAME -At -F \$'\t' -c \"$sql\""
}

echo "[1/4] Query user and org by email: $EMAIL"
USER_ROW="$(run_psql "SELECT id, email, COALESCE(current_org_id::text, org_id::text, last_active_org_id::text) AS org_id FROM users WHERE email = '$EMAIL' LIMIT 1;")"

if [[ -z "$USER_ROW" ]]; then
  echo "User not found: $EMAIL" >&2
  exit 1
fi

USER_ID="$(printf '%s' "$USER_ROW" | awk -F '\t' '{print $1}')"
ORG_ID="$(printf '%s' "$USER_ROW" | awk -F '\t' '{print $3}')"

if [[ -z "$ORG_ID" || "$ORG_ID" == "null" ]]; then
  echo "No org_id found for user: $EMAIL" >&2
  exit 1
fi

echo "user_id=$USER_ID"
echo "org_id=$ORG_ID"

echo "[2/4] Query current available credits"
BALANCE_SQL="SELECT COALESCE(SUM(CASE WHEN direction IN ('grant','refund','consume_revert') THEN amount WHEN direction='consume' THEN -amount ELSE 0 END), 0) AS available_credits FROM credits_ledgers WHERE billing_subject_type = 'organization' AND billing_subject_id = '$ORG_ID';"
BEFORE_BALANCE="$(run_psql "$BALANCE_SQL")"
echo "before_balance=${BEFORE_BALANCE:-0}"

if [[ "$DRY_RUN" == "true" ]]; then
  echo "[dry-run] Skip insert."
  exit 0
fi

echo "[3/4] Insert credits grant ledger"
INSERT_SQL="INSERT INTO credits_ledgers (id, billing_subject_type, billing_subject_id, direction, amount, reason, reference_id, created_at) VALUES ('credit_manual_' || extract(epoch from clock_timestamp())::bigint, 'organization', '$ORG_ID', 'grant', $AMOUNT, 'manual_sql_topup', '$REFERENCE_ID', NOW());"
ssh "$HOST" "docker exec $CONTAINER psql -U $DB_USER -d $DB_NAME -c \"$INSERT_SQL\""

echo "[4/4] Query latest balance and last ledger"
AFTER_BALANCE="$(run_psql "$BALANCE_SQL")"
LAST_LEDGER="$(run_psql "SELECT direction, amount, reason, reference_id, created_at FROM credits_ledgers WHERE billing_subject_type = 'organization' AND billing_subject_id = '$ORG_ID' ORDER BY created_at DESC LIMIT 1;")"

echo "after_balance=${AFTER_BALANCE:-0}"
echo "last_ledger=$LAST_LEDGER"
