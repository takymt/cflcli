#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

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

build_cfl() {
  echo "INFO: building cfl"
  mise run build
}

api_get_status() {
  local path status
  path="$1"
  status="$(
    curl -sS -o /tmp/cfl_live_response.json -w "%{http_code}" \
      -H "Authorization: Basic ${CONFLUENCE_BASIC_AUTH}" \
      -H "Accept: application/json" \
      "https://${CONFLUENCE_DOMAIN}${path}"
  )"
  printf '%s' "$status"
}

assert_numeric_id() {
  local label value
  label="$1"
  value="$2"
  if [[ ! "$value" =~ ^[0-9]+$ ]]; then
    echo "ERROR: ${label} must be numeric for Confluence REST API v2, got: ${value}" >&2
    exit 1
  fi
}

preflight_case_validate_env_values() {
  assert_numeric_id "SPACE_ID" "${SPACE_ID}"
  assert_numeric_id "PARENT_ID" "${PARENT_ID}"
}

preflight_case_check_space_exists() {
  local status
  status="$(api_get_status "/wiki/api/v2/spaces/${SPACE_ID}")"
  if [[ "$status" != "200" ]]; then
    echo "ERROR: space lookup failed (HTTP ${status}) for SPACE_ID=${SPACE_ID}" >&2
    head -c 300 /tmp/cfl_live_response.json >&2 || true
    echo >&2
    exit 1
  fi
}

preflight_case_check_parent_exists() {
  local status
  status="$(api_get_status "/wiki/api/v2/pages/${PARENT_ID}")"
  if [[ "$status" != "200" ]]; then
    echo "ERROR: parent page lookup failed (HTTP ${status}) for PARENT_ID=${PARENT_ID}" >&2
    head -c 300 /tmp/cfl_live_response.json >&2 || true
    echo >&2
    exit 1
  fi
}

test_case_page_new() {
  local random_suffix file_name
  random_suffix="$(date +%s)-$RANDOM"
  file_name="page-new-${random_suffix}.md"

  echo "INFO: test_case_page_new"
  echo "INFO: running: ./cfl page new ${file_name} --parent-id ${PARENT_ID} --space-id ${SPACE_ID}"
  ./cfl page new "$file_name" --parent-id "${PARENT_ID}" --space-id "${SPACE_ID}"
  echo "INFO: created local file: ${file_name}"
}

main() {
  load_env
  require_env CONFLUENCE_DOMAIN CONFLUENCE_EMAIL CONFLUENCE_API_TOKEN SPACE_ID PARENT_ID
  CONFLUENCE_BASIC_AUTH="$(printf '%s:%s' "${CONFLUENCE_EMAIL}" "${CONFLUENCE_API_TOKEN}" | base64)"
  export CONFLUENCE_BASIC_AUTH
  preflight_case_validate_env_values
  preflight_case_check_space_exists
  preflight_case_check_parent_exists
  build_cfl
  test_case_page_new
}

main "$@"
