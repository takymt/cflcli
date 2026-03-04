#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/live-test.sh --bin <path> [--profile <name>] --space-id <space-id> [options]

Required arguments:
  --bin <path>              Path to the built cfl binary
  --space-id <space-id>     Target Confluence space id

Optional arguments:
  --profile <name>          cfl profile name
  --parent-id <parent-id>   Parent page id. If omitted, the space homepage id is used
  --title <title>           Page title. Default: cfl-live-test-<timestamp>
  --create-body <body>      Markdown body used for create
  --update-body <body>      Markdown body used for update
  -h, --help                Show this help
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

die() {
  echo "error: $*" >&2
  exit 1
}

run_cfl() {
  local output_file
  output_file="$(mktemp)"

  if ! "${CFL_BIN}" "${GLOBAL_ARGS[@]}" "$@" >"${output_file}" 2>&1; then
    {
      echo "cfl command failed"
      echo "  binary: ${CFL_BIN}"
      echo "  args: ${GLOBAL_ARGS[*]} $*"
      echo "  output:"
      sed 's/^/    /' "${output_file}"
    } >&2
    rm -f "${output_file}"
    exit 1
  fi

  cat "${output_file}"
  rm -f "${output_file}"
}

require_cmd jq

CFL_BIN=""
PROFILE=""
SPACE_ID=""
PARENT_ID=""
TITLE=""
CREATE_BODY=""
UPDATE_BODY=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bin)
      CFL_BIN="${2:-}"
      shift 2
      ;;
    --profile)
      PROFILE="${2:-}"
      shift 2
      ;;
    --space-id)
      SPACE_ID="${2:-}"
      shift 2
      ;;
    --parent-id)
      PARENT_ID="${2:-}"
      shift 2
      ;;
    --title)
      TITLE="${2:-}"
      shift 2
      ;;
    --create-body)
      CREATE_BODY="${2:-}"
      shift 2
      ;;
    --update-body)
      UPDATE_BODY="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage >&2
      die "unknown argument: $1"
      ;;
  esac
done

[[ -n "${CFL_BIN}" ]] || { usage >&2; die "--bin is required"; }
[[ -n "${SPACE_ID}" ]] || { usage >&2; die "--space-id is required"; }
[[ -x "${CFL_BIN}" ]] || die "--bin must point to an executable file: ${CFL_BIN}"

timestamp="$(date -u +%Y%m%d%H%M%S)"
title="${TITLE:-cfl-live-test-${timestamp}}"
create_body="${CREATE_BODY:-created by live-test.sh at ${timestamp}}"
update_body="${UPDATE_BODY:-updated by live-test.sh at ${timestamp}}"

GLOBAL_ARGS=(-o json)
if [[ -n "${PROFILE}" ]]; then
  GLOBAL_ARGS+=(-p "${PROFILE}")
fi

create_body_file="$(mktemp)"
update_body_file="$(mktemp)"

cleanup() {
  rm -f "${create_body_file}" "${update_body_file}"
}
trap cleanup EXIT

printf '%s\n' "${create_body}" >"${create_body_file}"
printf '%s\n' "${update_body}" >"${update_body_file}"

create_args=(page create --space-id "${SPACE_ID}" --title "${title}" --body-file "${create_body_file}" --body-format markdown)
if [[ -n "${PARENT_ID}" ]]; then
  create_args+=(--parent-id "${PARENT_ID}")
fi

echo "Using binary ${CFL_BIN} space ${SPACE_ID} title ${title}"
if [[ -n "${PARENT_ID}" ]]; then
  echo "Using parent ${PARENT_ID}"
fi

create_response="$(run_cfl "${create_args[@]}")"

page_id="$(printf '%s' "${create_response}" | jq -er '.id // .page.id // .result.id // .data.id')" \
  || die "failed to parse page id from create output"

page_url="$(printf '%s' "${create_response}" | jq -er '.url // .page.url // .result.url // .data.url // empty' || true)"
if [[ -z "${page_url}" ]]; then
  page_url="(url not present in cfl output)"
fi

echo "Created page ${page_id} ${page_url}"

update_args=(page update "${page_id}" --title "${title}" --body-file "${update_body_file}" --body-format markdown)
if [[ -n "${PARENT_ID}" ]]; then
  update_args+=(--parent-id "${PARENT_ID}")
fi

update_response="$(run_cfl "${update_args[@]}")"

updated_page_id="$(printf '%s' "${update_response}" | jq -er '.id // .page.id // .result.id // .data.id')" \
  || die "failed to parse page id from update output"

echo "Updated page ${updated_page_id}"
echo "Live test passed"
