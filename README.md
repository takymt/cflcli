> [!WARNING]
> This README is a temporary AI-generated draft focused on usability first. A human-edited version will follow.

# cfl

`cfl` is a CLI for Confluence Cloud (REST API v2) with a Markdown-friendly workflow for page operations.

## Usage

Output of `cfl --help`:

```text
CLI tool for Confluence Cloud REST API v2

Usage:
  cfl [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  config      Manage configuration profiles
  help        Help about any command
  init        Initialize cfl configuration
  migrate     Migration commands
  page        Manage Confluence pages
  use         Switch to a profile

Flags:
  -h, --help             help for cfl
  -o, --output string    output format (json | table) (default "table")
  -p, --profile string   profile name (temporary override)
  -v, --verbose          verbose output
  -V, --version          version for cfl

Use "cfl [command] --help" for more information about a command.
```

## Quick Start

1. Install:

```bash
go install github.com/takymt/cflcli/cmd/cfl@latest
cfl --help
```

2. Set your Confluence API token:

```bash
export CFL_API_TOKEN="your_confluence_api_token"
```

3. Initialize config (creates a `default` profile interactively on first run):

```bash
cfl init
```

4. List pages (uses `space_key` from the selected profile if configured):

```bash
cfl page list --limit 25
```

If you are working from a local checkout of this repo, you can run `go run ./cmd/cfl` instead of `cfl`.

## Typical Flows

Create a page from Markdown:

```bash
cfl page create --space-key TEST --title "Release Notes" --body-file ./docs/release-notes.md
```

Update a page from Markdown:

```bash
cfl page update 123456 --title "Release Notes v2" --body-file ./docs/release-notes.md
```

Export pages to Markdown (migration flow):

```bash
cfl migrate export --space-key TEST --out ./export
```

## Markdown Notes

- Default `--body-format` is `markdown`.
- Use `--body-format storage` to send Confluence storage format as-is.
- Frontmatter keys supported for create/update: `title`, `parent-id` (also `parent_id`, `parentid`).
- Local images in Markdown are uploaded as Confluence attachments and rewritten to `ri:attachment` references.
- Mermaid fenced code blocks are rendered to images by default; disable with `--no-render-mermaid`.

Frontmatter example:

```markdown
---
title: Weekly Update
parent-id: "123456"
---

# Summary
```

## Configuration

Config file:

- `$XDG_CONFIG_HOME/cflcli/config.toml`
- Default: `~/.config/cflcli/config.toml`

Minimal example:

```toml
current = "work"

[[profiles]]
name = "work"
domain = "your-domain.atlassian.net"
user = "you@example.com"
space_key = "TEST"
output = "table"
```

See `TODO.md` for planned features.
