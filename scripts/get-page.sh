#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

usage() {
  cat <<'EOF'
Usage:
  scripts/get-page.sh <page-id> [body-format]

Examples:
  scripts/get-page.sh 3604481
  scripts/get-page.sh 3604481 storage
EOF
}

load_env() {
  if [[ ! -f ".env" ]]; then
    echo "ERROR: .env not found at repository root" >&2
    exit 1
  fi

  set -a
  # shellcheck disable=SC1091
  source ".env"
  set +a
}

require_env() {
  local name
  for name in "$@"; do
    if [[ -z "${!name:-}" ]]; then
      echo "ERROR: $name is required in .env" >&2
      exit 1
    fi
  done
}

main() {
  if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
  fi

  local page_id body_format auth response
  page_id="${1:-}"
  body_format="${2:-storage}"

  if [[ -z "$page_id" ]]; then
    usage >&2
    exit 1
  fi
  if [[ ! "$page_id" =~ ^[0-9]+$ ]]; then
    echo "ERROR: page-id must be numeric" >&2
    exit 1
  fi

  load_env
  require_env CONFLUENCE_DOMAIN CONFLUENCE_EMAIL CONFLUENCE_API_TOKEN

  auth="$(printf '%s:%s' "${CONFLUENCE_EMAIL}" "${CONFLUENCE_API_TOKEN}" | base64)"
  response="$(
    curl -sS \
      -H "Authorization: Basic ${auth}" \
      -H "Accept: application/json" \
      "https://${CONFLUENCE_DOMAIN}/wiki/api/v2/pages/${page_id}?body-format=${body_format}"
  )"

  if command -v jq >/dev/null 2>&1; then
    printf '%s\n' "$response" | jq .
  else
    printf '%s\n' "$response"
  fi
}

main "$@"
