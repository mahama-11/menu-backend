#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  topup-menu-credits.sh --email EMAIL --amount AMOUNT [options]

Options:
  --email EMAIL           User email to top up. Required.
  --amount AMOUNT         Credits amount to grant. Required.
  --asset-code CODE       Wallet asset code. Default: MENU_CREDIT
  --host HOST             SSH host alias. Default: boe
  --container NAME        Docker container name. Default: database
  --db-name NAME          PostgreSQL database name. Default: kong
  --db-user USER          PostgreSQL user. Default: kong
  --reference-id ID       Custom reference id. Optional.
  --dry-run               Query org and wallet balances only, do not insert.
  -h, --help              Show this help message.

Examples:
  ./scripts/topup-menu-credits.sh --email admin@verilocale.com --amount 1000
  ./scripts/topup-menu-credits.sh --email user@example.com --amount 500 --asset-code MENU_PROMO_CREDIT --dry-run
EOF
}

HOST="boe"
CONTAINER="database"
DB_NAME="kong"
DB_USER="kong"
ASSET_CODE="MENU_CREDIT"
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
    --asset-code)
      ASSET_CODE="${2:-}"
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
  REFERENCE_ID="manual_wallet_topup_${SAFE_EMAIL}_${ASSET_CODE}_${AMOUNT}_$(date +%Y%m%d%H%M%S)"
fi

run_psql() {
  local sql="$1"
  ssh "$HOST" "docker exec $CONTAINER psql -U $DB_USER -d $DB_NAME -At -F \$'\t' -c \"$sql\""
}

echo "[1/5] Query user and org by email: $EMAIL"
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

echo "[2/5] Query wallet asset definition"
ASSET_DEF_ROW="$(run_psql "SELECT asset_code, asset_type, lifecycle_type, status FROM asset_definitions WHERE asset_code = '$ASSET_CODE' LIMIT 1;")"
if [[ -z "$ASSET_DEF_ROW" ]]; then
  echo "Asset definition not found: $ASSET_CODE" >&2
  exit 1
fi

ASSET_TYPE="$(printf '%s' "$ASSET_DEF_ROW" | awk -F '\t' '{print $2}')"
LIFECYCLE_TYPE="$(printf '%s' "$ASSET_DEF_ROW" | awk -F '\t' '{print $3}')"
ASSET_STATUS="$(printf '%s' "$ASSET_DEF_ROW" | awk -F '\t' '{print $4}')"

echo "asset_code=$ASSET_CODE"
echo "asset_type=$ASSET_TYPE"
echo "lifecycle_type=$LIFECYCLE_TYPE"
echo "asset_status=$ASSET_STATUS"

if [[ "$ASSET_STATUS" != "active" ]]; then
  echo "Asset definition is not active: $ASSET_CODE" >&2
  exit 1
fi

echo "[3/5] Query current wallet balance"
ACCOUNT_ROW="$(run_psql "SELECT id, balance FROM wallet_accounts WHERE billing_subject_type = 'organization' AND billing_subject_id = '$ORG_ID' AND asset_code = '$ASSET_CODE' LIMIT 1;")"
ACCOUNT_ID="$(printf '%s' "$ACCOUNT_ROW" | awk -F '\t' '{print $1}')"
BEFORE_BALANCE="$(printf '%s' "$ACCOUNT_ROW" | awk -F '\t' '{print $2}')"

if [[ -z "$BEFORE_BALANCE" ]]; then
  BEFORE_BALANCE="0"
fi

echo "wallet_account_id=${ACCOUNT_ID:-<will_create>}"
echo "before_balance=$BEFORE_BALANCE"

if [[ "$DRY_RUN" == "true" ]]; then
  echo "[dry-run] Skip insert."
  exit 0
fi

echo "[4/5] Insert wallet bucket + ledger and update wallet account"
TOPUP_SQL="
DO \$\$
DECLARE
  v_account_id varchar(64);
  v_bucket_id varchar(64);
  v_ledger_id varchar(64);
BEGIN
  SELECT id INTO v_account_id
  FROM wallet_accounts
  WHERE billing_subject_type = 'organization'
    AND billing_subject_id = '$ORG_ID'
    AND asset_code = '$ASSET_CODE'
  LIMIT 1;

  IF v_account_id IS NULL THEN
    v_account_id := 'wa_' || replace(gen_random_uuid()::text, '-', '');
    INSERT INTO wallet_accounts (
      id, billing_subject_type, billing_subject_id, asset_code, asset_type, balance, status, metadata, created_at, updated_at
    ) VALUES (
      v_account_id, 'organization', '$ORG_ID', '$ASSET_CODE', '$ASSET_TYPE', 0, 'active', '{}', NOW(), NOW()
    );
  END IF;

  v_bucket_id := 'wb_' || replace(gen_random_uuid()::text, '-', '');
  INSERT INTO wallet_buckets (
    id, wallet_account_id, billing_subject_type, billing_subject_id, asset_code, asset_type, lifecycle_type,
    source_type, source_id, cycle_key, balance, status, metadata, created_at, updated_at
  ) VALUES (
    v_bucket_id, v_account_id, 'organization', '$ORG_ID', '$ASSET_CODE', '$ASSET_TYPE', '$LIFECYCLE_TYPE',
    'manual_topup', '$REFERENCE_ID', '', $AMOUNT, 'active', '{}', NOW(), NOW()
  );

  v_ledger_id := 'wl_' || replace(gen_random_uuid()::text, '-', '');
  INSERT INTO wallet_ledgers (
    id, wallet_account_id, wallet_bucket_id, billing_subject_type, billing_subject_id, asset_code,
    direction, amount, reason, reference_type, reference_id, status, metadata, created_at
  ) VALUES (
    v_ledger_id, v_account_id, v_bucket_id, 'organization', '$ORG_ID', '$ASSET_CODE',
    'credit', $AMOUNT, 'manual_sql_topup', 'manual_topup', '$REFERENCE_ID', 'posted', '{}', NOW()
  );

  UPDATE wallet_accounts
  SET balance = balance + $AMOUNT, updated_at = NOW()
  WHERE id = v_account_id;
END
\$\$;
"
ssh "$HOST" "docker exec $CONTAINER psql -U $DB_USER -d $DB_NAME -c \"$TOPUP_SQL\""

echo "[5/5] Query latest wallet balance and ledger"
AFTER_ROW="$(run_psql "SELECT id, balance FROM wallet_accounts WHERE billing_subject_type = 'organization' AND billing_subject_id = '$ORG_ID' AND asset_code = '$ASSET_CODE' LIMIT 1;")"
AFTER_ACCOUNT_ID="$(printf '%s' "$AFTER_ROW" | awk -F '\t' '{print $1}')"
AFTER_BALANCE="$(printf '%s' "$AFTER_ROW" | awk -F '\t' '{print $2}')"
LAST_LEDGER="$(run_psql "SELECT direction, amount, reason, reference_type, reference_id, created_at FROM wallet_ledgers WHERE billing_subject_type = 'organization' AND billing_subject_id = '$ORG_ID' AND asset_code = '$ASSET_CODE' ORDER BY created_at DESC LIMIT 1;")"

echo "wallet_account_id=${AFTER_ACCOUNT_ID:-unknown}"
echo "after_balance=${AFTER_BALANCE:-0}"
echo "last_wallet_ledger=$LAST_LEDGER"
