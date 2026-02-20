#!/usr/bin/env bash
set -euo pipefail

if rg -n --glob '*.go' '/wiki/(api/v1|rest/api)' cmd internal main.go; then
  echo "ERROR: v1 API path detected. Use Confluence REST API v2 (/wiki/api/v2)." >&2
  exit 1
fi
