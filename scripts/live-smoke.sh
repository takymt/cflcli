#!/usr/bin/env bash

set -euo pipefail

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

require_var() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "missing required env: $name" >&2
    exit 1
  fi
}

require_cmd curl
require_cmd jq

require_var DOMAIN
require_var EMAIL
require_var TOKEN
require_var SPACE_ID

base_url="https://${DOMAIN}"
api_base="${base_url}/wiki/api/v2"
timestamp="$(date -u +%Y%m%d%H%M%S)"
title="${TITLE:-cfl-live-smoke-${timestamp}}"
create_body="${CREATE_BODY:-<p>created by live-smoke.sh at ${timestamp}</p>}"
update_body="${UPDATE_BODY:-<p>updated by live-smoke.sh at ${timestamp}</p>}"

auth_args=(-u "${EMAIL}:${TOKEN}" -H "Accept: application/json")

resolve_parent_id() {
  if [[ -n "${PARENT_ID:-}" ]]; then
    printf '%s\n' "${PARENT_ID}"
    return 0
  fi

  curl -fsS "${auth_args[@]}" \
    "${api_base}/spaces/${SPACE_ID}" \
  | jq -er '.homepageId'
}

parent_id="$(resolve_parent_id)"

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

create_response="$(
  curl -fsS "${auth_args[@]}" \
    -H "Content-Type: application/json" \
    -X POST \
    --data "${create_payload}" \
    "${api_base}/pages"
)"

page_id="$(printf '%s' "${create_response}" | jq -er '.id')"
version_number="$(printf '%s' "${create_response}" | jq -er '.version.number')"
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

update_response="$(
  curl -fsS "${auth_args[@]}" \
    -H "Content-Type: application/json" \
    -X PUT \
    --data "${update_payload}" \
    "${api_base}/pages/${page_id}"
)"

updated_version="$(printf '%s' "${update_response}" | jq -er '.version.number')"

echo "Updated page ${page_id} to version ${updated_version}"
echo "Smoke passed: ${page_url}"
