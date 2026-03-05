#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

created_file=""
results_dir="$repo_root/test_results"

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

prepare_results_dir() {
  mkdir -p "$results_dir"
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

test_case_page_new() {
  local random_suffix file_name output
  random_suffix="$(date +%s)-$RANDOM"
  file_name="test_results/page-new-${random_suffix}.md"
  created_file="$file_name"

  echo "INFO: test_case_page_new"
  echo "INFO: running: ./cfl page new ${file_name} --parent-id ${PARENT_ID} --space-id ${SPACE_ID}"
  output="$(./cfl page new "$file_name" --parent-id "${PARENT_ID}" --space-id "${SPACE_ID}")"
  echo "$output"
  echo "INFO: created local file: ${file_name}"
}

main() {
  trap cleanup_on_error EXIT
  load_env
  require_env CONFLUENCE_DOMAIN CONFLUENCE_EMAIL CONFLUENCE_API_TOKEN SPACE_ID PARENT_ID
  prepare_results_dir
  build_cfl
  test_case_page_new
}

main "$@"
