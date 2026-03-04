#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/live-test.sh --domain <domain> --email <email> --token <token> --space-id <space-id> [options]

Required arguments:
  --domain <domain>         Confluence domain, for example example.atlassian.net
  --email <email>           Atlassian account email
  --token <token>           Atlassian API token
  --space-id <space-id>     Target Confluence space id

Optional arguments:
  --parent-id <parent-id>   Parent page id. If omitted, the space homepage id is used
  --title <title>           Page title. Default: cfl-live-test-<timestamp>
  --create-body <html>      Storage-format body used for create
  --update-body <html>      Storage-format body used for update
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

curl_json() {
  local method="$1"
  local url="$2"
  local data="${3-}"
  local body_file
  body_file="$(mktemp)"

  local http_code
  if [[ -n "${data}" ]]; then
    http_code="$(
      curl -sS \
        -o "${body_file}" \
        -w '%{http_code}' \
        -u "${EMAIL}:${TOKEN}" \
        -H 'Accept: application/json' \
        -H 'Content-Type: application/json' \
        -X "${method}" \
        --data "${data}" \
        "${url}"
    )"
  else
    http_code="$(
      curl -sS \
        -o "${body_file}" \
        -w '%{http_code}' \
        -u "${EMAIL}:${TOKEN}" \
        -H 'Accept: application/json' \
        -X "${method}" \
        "${url}"
    )"
  fi

  if [[ ! "${http_code}" =~ ^2 ]]; then
    {
      echo "request failed"
      echo "  method: ${method}"
      echo "  url: ${url}"
      echo "  status: ${http_code}"
      echo "  response:"
      sed 's/^/    /' "${body_file}"
    } >&2
    rm -f "${body_file}"
    exit 1
  fi

  cat "${body_file}"
  rm -f "${body_file}"
}

require_cmd curl
require_cmd jq

DOMAIN=""
EMAIL=""
TOKEN=""
SPACE_ID=""
PARENT_ID=""
TITLE=""
CREATE_BODY=""
UPDATE_BODY=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --domain)
      DOMAIN="${2:-}"
      shift 2
      ;;
    --email)
      EMAIL="${2:-}"
      shift 2
      ;;
    --token)
      TOKEN="${2:-}"
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

[[ -n "${DOMAIN}" ]] || { usage >&2; die "--domain is required"; }
[[ -n "${EMAIL}" ]] || { usage >&2; die "--email is required"; }
[[ -n "${TOKEN}" ]] || { usage >&2; die "--token is required"; }
[[ -n "${SPACE_ID}" ]] || { usage >&2; die "--space-id is required"; }

base_url="https://${DOMAIN}"
api_base="${base_url}/wiki/api/v2"
timestamp="$(date -u +%Y%m%d%H%M%S)"
title="${TITLE:-cfl-live-test-${timestamp}}"
create_body="${CREATE_BODY:-<p>created by live-test.sh at ${timestamp}</p>}"
update_body="${UPDATE_BODY:-<p>updated by live-test.sh at ${timestamp}</p>}"

resolve_parent_id() {
  if [[ -n "${PARENT_ID}" ]]; then
    printf '%s\n' "${PARENT_ID}"
    return 0
  fi

  curl_json GET "${api_base}/spaces/${SPACE_ID}" | jq -er '.homepageId'
}

parent_id="$(resolve_parent_id)" || die "failed to resolve parent page id"

echo "Using space ${SPACE_ID} parent ${parent_id} title ${title}"

create_payload="$(jq -n \
  --arg spaceId "${SPACE_ID}" \
  --arg status "current" \
  --arg title "${title}" \
  --arg parentId "${parent_id}" \
  --arg value "${create_body}" \
  '{
    spaceId: $spaceId,
    status: $status,
    title: $title,
    parentId: $parentId,
    body: {
      representation: "storage",
      value: $value
    }
  }'
)"

create_response="$(curl_json POST "${api_base}/pages" "${create_payload}")" || die "failed to create page"

page_id="$(printf '%s' "${create_response}" | jq -er '.id')" || die "failed to parse created page id"
version_number="$(printf '%s' "${create_response}" | jq -er '.version.number')" || die "failed to parse created page version"
page_url="${base_url}/wiki/pages/viewpage.action?pageId=${page_id}"

echo "Created page ${page_id} ${page_url}"

update_payload="$(jq -n \
  --arg id "${page_id}" \
  --arg spaceId "${SPACE_ID}" \
  --arg parentId "${parent_id}" \
  --arg status "current" \
  --arg title "${title}" \
  --arg value "${update_body}" \
  --argjson version "$((version_number + 1))" \
  '{
    id: $id,
    spaceId: $spaceId,
    parentId: $parentId,
    status: $status,
    title: $title,
    body: {
      representation: "storage",
      value: $value
    },
    version: {
      number: $version
    }
  }'
)"

update_response="$(curl_json PUT "${api_base}/pages/${page_id}" "${update_payload}")" || die "failed to update page"

updated_version="$(printf '%s' "${update_response}" | jq -er '.version.number')" || die "failed to parse updated page version"

echo "Updated page ${page_id} to version ${updated_version}"
echo "Live test passed: ${page_url}"
