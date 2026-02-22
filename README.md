> [!WARNING]
> This README is a temporary AI-generated draft focused on usability first. A human-edited version will follow.

# cfl

`cfl` is a CLI for Confluence Cloud (REST API v2) with a Markdown-friendly workflow for page operations.

## What You Can Do

- Manage profiles for multiple Confluence sites/environments
- List / get / create / update / delete pages
- Create/update pages from Markdown or storage format
- Use frontmatter (`title`, `parent-id`) in Markdown files
- Upload local images in Markdown as Confluence attachments
- Render Mermaid fenced blocks to images (default on)
- Export Confluence pages to Markdown + attachments (`migrate export`)

## Quick Start

### 1. Run without building

From this repo:

```bash
go run . --help
```

### 2. Set your API token

`cfl` uses Basic Auth:

- Username: profile email
- Password: `CFL_API_TOKEN`

```bash
export CFL_API_TOKEN="your_confluence_api_token"
```

### 3. Initialize config and create a profile

```bash
cfl init
```

This creates a `default` profile interactively on first run.

### 4. List pages

If your profile has `space_key`, you can omit `--space-key`.

```bash
cfl page list --limit 25
```

Examples below use `cfl`. If you have not installed/built a binary, replace `cfl` with `go run .`.

## Common Commands

List pages (with filters):

```bash
cfl page list --space-key TEST --status current,archived --sort -modified-date --limit 50
```

Get page body (Confluence storage format):

```bash
cfl page get 123456 > page.storage.xhtml
```

Create page from Markdown:

```bash
cfl page create --space-key TEST --title "Release Notes" --body-file ./docs/release-notes.md
```

Update page from Markdown:

```bash
cfl page update 123456 --title "Release Notes v2" --body-file ./docs/release-notes.md
```

Delete page:

```bash
cfl page delete 123456
```

Export pages to Markdown (migration):

```bash
cfl migrate export --space-key TEST --out ./export
```

Export a subtree only:

```bash
cfl migrate export --space-key TEST --root-page-id 123456 --out ./export
```

## Markdown Notes

- Default `--body-format` is `markdown`; use `--body-format storage` to send storage format as-is.
- Frontmatter keys supported for create/update: `title`, `parent-id` (also `parent_id`, `parentid`).
- If both a flag and frontmatter specify the same field, `cfl` returns an error.
- Local images in Markdown are uploaded as attachments and rewritten to `ri:attachment` references.
- Mermaid fenced code blocks are rendered to images by default; disable with `--no-render-mermaid`.

Frontmatter example:

```markdown
---
title: Weekly Update
parent-id: "123456"
---

# Summary
```

### Local image path resolution (`--body-format markdown`)

- `http://` / `https://`: external URL
- `./` / `../` / bare path: relative to the `--body-file` directory
- `/...`: relative to `--assets-root` (not OS root)

If `assets_root` is set in the profile, it is used when `--assets-root` is omitted.

## Configuration

Config file:

- `$XDG_CONFIG_HOME/cflcli/config.toml`
- Default: `~/.config/cflcli/config.toml`

Example:

```toml
current = "work"

[[profiles]]
name = "work"
domain = "your-domain.atlassian.net"
user = "you@example.com"
space_key = "TEST"
assets_root = "/Users/you/docs"
output = "table"
```

Useful profile commands:

```bash
cfl config init [name]
cfl config edit <name>
cfl config list
cfl config show
cfl config delete <name>
cfl use [name]
```

## Output and Global Flags

Global flags:

- `-o, --output`: `table` (default) / `json`
- `-p, --profile`: temporary profile override
- `-v, --verbose`: verbose output

Example JSON output:

```bash
cfl -o json page list --space-key TEST
```

Note: `page get` always prints the storage body directly (it does not use `--output`).

## Current Commands

```text
cfl
├── init
├── config (init / edit / use / list / show / delete)
├── use
├── page (list / get / create / update / delete)
└── migrate (export)
```

Planned features (labels, folder, attachment, comment, sync, etc.) are tracked in `TODO.md`.

## Development

Requirements:

- Go `1.26.0`
- `mise` (optional)

Common tasks:

```bash
mise run all
mise run fmt
mise run lint
mise run test
mise run test-it
mise run test-live
mise run cover
mise run build
```

See `TESTING.md` for test strategy.
