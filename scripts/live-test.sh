#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

created_file=""
created_page_id=""

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

build_basic_auth() {
  CONFLUENCE_BASIC_AUTH="$(printf '%s:%s' "${CONFLUENCE_EMAIL}" "${CONFLUENCE_API_TOKEN}" | base64)"
  export CONFLUENCE_BASIC_AUTH
}

cleanup_on_error() {
  local status
  status=$?
  if [[ "$status" -eq 0 ]]; then
    return
  fi

  if [[ -n "$created_file" && -f "$created_file" ]]; then
    rm -f "$created_file"
    echo "INFO: removed local file on failure: ${created_file}" >&2
  fi
}

api_get_page_storage_json() {
  local page_id
  page_id="$1"
  curl -sS \
    -H "Authorization: Basic ${CONFLUENCE_BASIC_AUTH}" \
    -H "Accept: application/json" \
    "https://${CONFLUENCE_DOMAIN}/wiki/api/v2/pages/${page_id}?body-format=storage"
}

extract_page_id_from_url() {
  local url
  url="$1"
  if [[ "$url" =~ pageId=([0-9]+)$ ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
    return 0
  fi
  return 1
}

assert_contains() {
  local text needle message
  text="$1"
  needle="$2"
  message="$3"

  if [[ "$text" != *"$needle"* ]]; then
    echo "ERROR: ${message}" >&2
    exit 1
  fi
}

verify_created_page() {
  local page_id expected_title json
  page_id="$1"
  expected_title="$2"
  json="$(api_get_page_storage_json "$page_id")"

  assert_contains "$json" "\"id\":\"${page_id}\"" "created page id mismatch"
  assert_contains "$json" "\"title\":\"${expected_title}\"" "created page title mismatch"
  assert_contains "$json" "\"parentId\":\"${PARENT_ID}\"" "created page parent mismatch"
  assert_contains "$json" "\"spaceId\":\"${SPACE_ID}\"" "created page space mismatch"
  assert_contains "$json" "\"value\":\"\"" "created page body is not empty"
}

test_case_page_new() {
  local random_suffix file_name page_title output url
  random_suffix="$(date +%s)-$RANDOM"
  file_name="page-new-${random_suffix}.md"
  page_title="${file_name%.md}"
  created_file="$file_name"

  echo "INFO: test_case_page_new"
  echo "INFO: running: ./cfl page new ${file_name} --parent-id ${PARENT_ID} --space-id ${SPACE_ID}"
  output="$(./cfl page new "$file_name" --parent-id "${PARENT_ID}" --space-id "${SPACE_ID}")"
  echo "$output"

  url="$(printf '%s' "$output" | tail -n1)"
  created_page_id="$(extract_page_id_from_url "$url")"
  if [[ -z "$created_page_id" ]]; then
    echo "ERROR: failed to parse page-id from output URL: ${url}" >&2
    exit 1
  fi

  verify_created_page "$created_page_id" "$page_title"
  echo "INFO: verified created page via Confluence API (page-id=${created_page_id})"
  echo "INFO: created local file: ${file_name}"
}

main() {
  trap cleanup_on_error EXIT
  load_env
  require_env CONFLUENCE_DOMAIN CONFLUENCE_EMAIL CONFLUENCE_API_TOKEN SPACE_ID PARENT_ID
  build_basic_auth
  build_cfl
  test_case_page_new
}

main "$@"
