#!/usr/bin/env bash
set -euo pipefail

matches="$(rg -n --glob '*.go' '/wiki/(api/v1|rest/api)' cmd internal main.go || true)"
if [[ -z "$matches" ]]; then
  exit 0
fi

# Confluence attachments upload still requires v1 endpoint.
allowed='internal/client/attachment.go|internal/client/client_test.go|cmd/page_test.go'
violations="$(printf '%s\n' "$matches" | rg -v "$allowed" || true)"
if [[ -n "$violations" ]]; then
  printf '%s\n' "$violations"
  echo "ERROR: unexpected v1 API path detected. Use Confluence REST API v2 (/wiki/api/v2) except attachments." >&2
  exit 1
fi
