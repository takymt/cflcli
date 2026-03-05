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
  build_cfl
  test_case_page_new
}

main "$@"
