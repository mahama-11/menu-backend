#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "[guardrails] checking for forbidden gorm.Open usage"
OPEN_MATCHES="$(rg -n "gorm\.Open\(" "$ROOT_DIR" --glob '*.go' --glob '!internal/storage/*' --glob '!**/*_test.go' --glob '!vendor/**' || true)"
if [[ -n "$OPEN_MATCHES" ]]; then
  echo "Forbidden gorm.Open usage found outside internal/storage:"
  echo "$OPEN_MATCHES"
  exit 1
fi

echo "[guardrails] checking for forbidden TableName overrides"
TABLENAME_MATCHES="$(rg -n "func\s+\([^)]*\)\s+TableName\(" "$ROOT_DIR/internal/models" --glob '*.go' || true)"
if [[ -n "$TABLENAME_MATCHES" ]]; then
  echo "Forbidden TableName override found in internal/models:"
  echo "$TABLENAME_MATCHES"
  exit 1
fi

echo "[guardrails] all checks passed"
