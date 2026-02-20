#!/usr/bin/env bash
set -euo pipefail

ALLOW_NONTEST=false
DOMAIN="${CFL_LIVE_DOMAIN:-}"
USER="${CFL_LIVE_USER:-}"
SPACE_ID="${CFL_LIVE_SPACE_ID:-}"
LIMIT="${CFL_LIVE_LIMIT:-5}"

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"

usage() {
  cat <<'USAGE'
Usage: scripts/live-test.sh [options]

Options:
  --domain <domain>     Confluence domain (e.g. example.atlassian.net)
  --user <email>        Confluence user email
  --space-id <id>       Optional space-id for page list
  --limit <n>           Page list limit (default: 5)
  --allow-nontest       Allow running against non-test accounts
  -h, --help            Show this help

Environment:
  CONFLUENCE_API_TOKEN  Required API token
  CFL_LIVE_DOMAIN       Same as --domain
  CFL_LIVE_USER         Same as --user
  CFL_LIVE_SPACE_ID     Same as --space-id
  CFL_LIVE_LIMIT        Same as --limit
  CFL_LIVE_ALLOW_NONTEST=1 to bypass test account guardrail
USAGE
}

while [ $# -gt 0 ]; do
  case "$1" in
    --domain)
      DOMAIN="$2"
      shift
      ;;
    --user)
      USER="$2"
      shift
      ;;
    --space-id)
      SPACE_ID="$2"
      shift
      ;;
    --limit)
      LIMIT="$2"
      shift
      ;;
    --allow-nontest)
      ALLOW_NONTEST=true
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown arg: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

is_test_account() {
  local a
  a=$(echo "$1" | tr 'A-Z' 'a-z')
  case "$a" in
    *test*|*bot*|*sandbox*|*qa*|*staging*|*dev*|*@example.com)
      return 0
      ;;
  esac
  case "$a" in
    *+*)
      return 0
      ;;
  esac
  return 1
}

if [ -z "$DOMAIN" ] || [ -z "$USER" ] || [ -z "${CONFLUENCE_API_TOKEN:-}" ]; then
  echo "CFL_LIVE_DOMAIN, CFL_LIVE_USER, and CONFLUENCE_API_TOKEN are required." >&2
  exit 1
fi

if [ "${ALLOW_NONTEST:-false}" = false ] && [ -z "${CFL_LIVE_ALLOW_NONTEST:-}" ]; then
  if ! is_test_account "$USER"; then
    echo "Refusing to run live tests against non-test account: $USER" >&2
    echo "Pass --allow-nontest or set CFL_LIVE_ALLOW_NONTEST=1 to override." >&2
    exit 2
  fi
fi

BIN="${CFL_BIN:-$ROOT_DIR/cfl}"
if [ ! -x "$BIN" ]; then
  go build -o "$BIN" "$ROOT_DIR"
fi

LIVE_TMP="$(mktemp -d "${TMPDIR:-/tmp}/cfl-live-XXXXXX")"
trap 'rm -rf "$LIVE_TMP"' EXIT

export XDG_CONFIG_HOME="$LIVE_TMP/xdg"
mkdir -p "$XDG_CONFIG_HOME/cflcli"

cat >"$XDG_CONFIG_HOME/cflcli/config.toml" <<EOF
current = "live"

[[profiles]]
name = "live"
domain = "$DOMAIN"
user = "$USER"
space_key = "${SPACE_ID:-LIVE}"
output = "table"
EOF

echo "Using account: $USER"

echo "==> config list"
"$BIN" config list >/dev/null

echo "==> config show"
"$BIN" config show >/dev/null

LIST_JSON=""
if [ -n "$SPACE_ID" ]; then
  echo "==> page list (json, with space-id)"
  LIST_JSON="$("$BIN" --output json page list --space-id "$SPACE_ID" --limit "$LIMIT")"
  echo "==> page list (table, with space-id)"
  "$BIN" --output table page list --space-id "$SPACE_ID" --limit "$LIMIT" >/dev/null
else
  echo "==> page list (json)"
  LIST_JSON="$("$BIN" --output json page list --limit "$LIMIT")"
  echo "==> page list (table)"
  "$BIN" --output table page list --limit "$LIMIT" >/dev/null
fi

PAGE_ID="$(printf '%s\n' "$LIST_JSON" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
if [ -n "$PAGE_ID" ]; then
  echo "==> page get (id: $PAGE_ID)"
  "$BIN" page get "$PAGE_ID" >/dev/null
else
  echo "==> page get skipped (no page found from list)"
fi

echo "Live tests complete."
