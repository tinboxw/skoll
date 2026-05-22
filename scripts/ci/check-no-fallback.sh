#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

TARGET_DIRS=("internal/handler/http/v1/plugin" "plugins/developer-portal/static")
FILE_PATTERNS=("*.go" "*.js" "*.ts" "*.vue" "*.html" "*.yml" "*.yaml")
FORBIDDEN_TOKENS=("fallback" "degrade" "兜底" "兼容实现")
LEGACY_API_PATTERNS=("/v1/plugins/dev/release([\"'[:space:]]|$)")
LEGACY_STATUS_FIELD_PATTERNS=("\breleaseStatus\b" "\brelease_status\b" "\bpublishStatus\b" "\bpublish_status\b")

pattern_args=()
for pattern in "${FILE_PATTERNS[@]}"; do
  pattern_args+=( -name "$pattern" -o )
done
unset 'pattern_args[${#pattern_args[@]}-1]'

mapfile -d '' target_files < <(find "${TARGET_DIRS[@]}" \
  -type f \
  ! -name '*_test.go' \
  ! -path '*/node_modules/*' \
  ! -path '*/dist/*' \
  \( "${pattern_args[@]}" \) \
  -print0)

if [ "${#target_files[@]}" -eq 0 ]; then
  echo "No target source files found; skip no-fallback check."
  exit 0
fi

echo "Running no-fallback baseline checks..."

for token in "${FORBIDDEN_TOKENS[@]}"; do
  if printf '%s\0' "${target_files[@]}" | xargs -0 grep -InE "\b${token}\b" >/tmp/no_fallback_hits.txt; then
    echo "Forbidden token detected: $token"
    cat /tmp/no_fallback_hits.txt
    exit 1
  fi
done

for pattern in "${LEGACY_API_PATTERNS[@]}"; do
  if printf '%s\0' "${target_files[@]}" | xargs -0 grep -InE "$pattern" >/tmp/no_fallback_legacy_hits.txt; then
    echo "Legacy API pattern detected: $pattern"
    cat /tmp/no_fallback_legacy_hits.txt
    exit 1
  fi
done

for pattern in "${LEGACY_STATUS_FIELD_PATTERNS[@]}"; do
  if printf '%s\0' "${target_files[@]}" | xargs -0 grep -InE "$pattern" >/tmp/no_fallback_legacy_status_hits.txt; then
    echo "Legacy release status field detected: $pattern"
    cat /tmp/no_fallback_legacy_status_hits.txt
    exit 1
  fi
done

echo "No-fallback checks passed."
