# AGENTS.md - cfl (Confluence CLI)

## Overview

A CLI tool for Confluence Cloud using REST API v2.

- Binary name: `cfl`
- Authentication: Basic Auth (email + API token)

## API Reference

- Base URL: `https://{domain}.atlassian.net/wiki/api/v2/`
- Docs: https://developer.atlassian.com/cloud/confluence/rest/v2/intro/
- Pagination: cursor-based (`limit` + `cursor` parameters, `_links.next` in response)

## Configuration

- Path: `$XDG_CONFIG_HOME/cflcli/config.toml` (default: `~/.config/cflcli/config.toml`)
- API token: environment variable `CFL_API_TOKEN` (preferred), fallback to config file with file permission warning
- Multiple named profiles supported; `current` field tracks active profile

```toml
current = "work"

[[profiles]]
name = "work"
domain = "tarte-dev.atlassian.net"
user = "user@example.com"
space_key = "TEST"

[[profiles]]
name = "personal"
domain = "my-site.atlassian.net"
user = "me@example.com"
space_key = "DEV"
```

## Command Structure

Commands follow REST API v2 endpoint structure. Resource names are singular.

### Global Flags

```
--profile, -p   : profile name (temporary override)
--output, -o    : output format (json | table), default: table
--verbose, -v   : verbose output
```

### Commands

See [TODO.md](TODO.md) for full command list by phase.

### Body Handling (Phase 1)

- `--body-file` accepts a file in Confluence storage format (XHTML)
- No Markdown conversion (planned for Phase 2)
- `get` outputs storage format as-is

## Project Structure

```
cflcli/
├── main.go
├── go.mod / go.sum
├── mise.toml
├── lefthook.yml
├── .golangci.yml
├── README.md
├── TODO.md
├── AGENTS.md
├── cmd/                     # Command definitions (cobra)
│   ├── root.go              # cfl
│   ├── config.go            # cfl config {init|list|show|delete}
│   ├── use.go               # cfl use [profile-name]
│   └── page.go              # cfl page {list|get|create|update|delete}
├── internal/
│   ├── client/              # HTTP client
│   │   ├── client.go
│   │   ├── page.go
│   │   └── pagination.go
│   ├── config/              # Config management (XDG, TOML, profiles)
│   │   └── config.go
│   ├── model/               # API response structs
│   │   └── page.go
│   └── output/              # Output formatters
│       ├── output.go
│       └── table.go
└── test/
    └── page_cli_test.go
```

## Libraries

Dependency policy: avoid large dependencies; lightweight, focused libraries are preferred.

- **CLI framework**: `github.com/spf13/cobra` — subcommand routing, completion, help generation
- **TOML parser**: `github.com/BurntSushi/toml` — lightweight, stable, TOML 1.0 compliant
- **Color output**: `github.com/fatih/color` — terminal color with automatic TTY detection
- **HTTP client**: standard `net/http` — sufficient for Basic Auth + JSON
- **Table output**: standard `text/tabwriter` — kubectl-style aligned output

## Common Commands

```bash
mise run all          # Run fmt, lint, test, build (in order)
mise run fmt          # Format code
mise run lint         # Run golangci-lint
mise run test         # Run all tests
mise run build        # Build binary (./cfl)
mise run bump         # Bump patch version via gobump
mise run clean        # Remove build artifacts
```

## Testing

- Integration tests run against live Confluence Cloud: `https://tarte-dev.atlassian.net/wiki/spaces/TEST`
- Unit tests use `net/http/httptest` for mocking API responses

## Workflow

Each TODO.md task is one workflow unit. Steps:

1. Review TODO.md — identify the target task
2. Review AGENTS.md — confirm conventions and constraints
3. Write integration tests first
4. Implement code and write unit tests
5. Run `mise run all` — confirm fmt, lint, test, build all pass
6. Update TODO.md — mark task complete
7. Commit with conventional commit message

Make decisions independently except for significant architectural choices.
Summarize all decisions at workflow end for user review.
When user provides corrections, update AGENTS.md minimally to prevent recurrence.

**Important**: All 7 steps must be executed, including the final commit. Do not stop before committing.

### Post-commit Review

After committing, the user may review and request corrections.
When corrections are requested:

1. Apply the requested changes
2. Commit with an appropriate conventional commit message

### Dependency Management

- After `go get`, always run `go mod tidy` to ensure direct/indirect markers are correct.
- Never commit a go.mod where directly imported packages are marked `// indirect`.
