#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

created_files=()
created_page_ids=()
last_created_page_id=""
results_dir="$repo_root/test_results"
fixture_file="$repo_root/scripts/live-fixture-supported.md"

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

cleanup() {
  local status
  status=$?

  local file
  if [[ "$status" -ne 0 ]]; then
    for file in "${created_files[@]}"; do
      if [[ -f "$file" ]]; then
        rm -f "$file"
        echo "INFO: removed local file on failure: ${file}" >&2
      fi
    done
  fi

  local page_id
  for page_id in "${created_page_ids[@]}"; do
    if [[ -n "$page_id" ]]; then
      if ! scripts/delete-page.sh "$page_id" >/dev/null 2>&1; then
        echo "WARN: failed to delete page ${page_id}" >&2
      else
        echo "INFO: deleted page ${page_id}" >&2
      fi
    fi
  done
}

random_suffix() {
  printf '%s-%s' "$(date +%s)" "$RANDOM"
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

new_page_file() {
  local file_name output url page_id
  file_name="$1"
  output="$(./cfl page new "$file_name" --parent-id "${PARENT_ID}" --space-id "${SPACE_ID}")"
  echo "$output" >&2
  url="$(printf '%s' "$output" | tail -n1)"
  page_id="$(extract_page_id_from_url "$url")"
  if [[ -z "$page_id" ]]; then
    echo "ERROR: failed to parse page-id from: $url" >&2
    exit 1
  fi
  created_page_ids+=("$page_id")
  last_created_page_id="$page_id"
}

prepare_sync_target_from_fixture() {
  local page_id target_file tmp
  page_id="$1"
  target_file="$2"

  cp "$fixture_file" "$target_file"
  tmp="${target_file}.tmp"
  sed \
    -e "s/__SPACE_ID__/${SPACE_ID}/g" \
    -e "s/__PAGE_ID__/${page_id}/g" \
    -e "s/__PARENT_ID__/${PARENT_ID}/g" \
    "$target_file" >"$tmp"
  mv "$tmp" "$target_file"
}

test_case_page_new() {
  local suffix file_name
  suffix="$(random_suffix)"
  file_name="test_results/page-new-${suffix}.md"
  created_files+=("$file_name")

  echo "INFO: test_case_page_new"
  echo "INFO: running: ./cfl page new ${file_name} --parent-id ${PARENT_ID} --space-id ${SPACE_ID}"
  new_page_file "$file_name"
  echo "INFO: created page id: ${last_created_page_id}"
  echo "INFO: created local file: ${file_name}"
}

test_case_page_sync() {
  local suffix seed_file sync_file page_id
  suffix="$(random_suffix)"
  seed_file="test_results/sync-seed-${suffix}.md"
  sync_file="test_results/sync-${suffix}.md"
  created_files+=("$seed_file" "$sync_file")

  echo "INFO: test_case_page_sync"
  echo "INFO: creating seed page for sync target"
  new_page_file "$seed_file"
  page_id="$last_created_page_id"
  prepare_sync_target_from_fixture "$page_id" "$sync_file"

  echo "INFO: running: ./cfl page sync ${sync_file}"
  ./cfl page sync "$sync_file"
  echo "INFO: sync succeeded: ${sync_file}"
}

test_case_page_sync_watch() {
  local suffix seed_file watch_file page_id watch_pid
  suffix="$(random_suffix)"
  seed_file="test_results/watch-seed-${suffix}.md"
  watch_file="test_results/watch-${suffix}.md"
  created_files+=("$seed_file" "$watch_file")

  echo "INFO: test_case_page_sync_watch"
  echo "INFO: creating seed page for watch target"
  new_page_file "$seed_file"
  page_id="$last_created_page_id"
  prepare_sync_target_from_fixture "$page_id" "$watch_file"

  echo "INFO: running watch: ./cfl page sync ${watch_file} --watch"
  ./cfl page sync "$watch_file" --watch >/dev/null 2>&1 &
  watch_pid=$!
  sleep 1
  if ! kill -0 "$watch_pid" 2>/dev/null; then
    echo "ERROR: watch process exited unexpectedly" >&2
    wait "$watch_pid"
    exit 1
  fi

  cat >>"$watch_file" <<'EOF'

### Watch Update
- watch trigger
EOF
  sleep 2
  if ! kill -0 "$watch_pid" 2>/dev/null; then
    echo "ERROR: watch process exited after file update" >&2
    wait "$watch_pid"
    exit 1
  fi

  kill -INT "$watch_pid" 2>/dev/null || true
  for _ in $(seq 1 20); do
    if ! kill -0 "$watch_pid" 2>/dev/null; then
      break
    fi
    sleep 0.1
  done
  if kill -0 "$watch_pid" 2>/dev/null; then
    kill -TERM "$watch_pid" 2>/dev/null || true
  fi
  wait "$watch_pid" || true
  echo "INFO: watch sync succeeded: ${watch_file}"
}

main() {
  trap cleanup EXIT
  load_env
  require_env CONFLUENCE_DOMAIN CONFLUENCE_EMAIL CONFLUENCE_API_TOKEN SPACE_ID PARENT_ID
  prepare_results_dir
  build_cfl
  test_case_page_new
  test_case_page_sync
  test_case_page_sync_watch
}

main "$@"
