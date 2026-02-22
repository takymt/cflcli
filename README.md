> [!WARNING]
> This README is a temporary AI-generated draft. It is optimized for getting users unstuck and productive quickly. A human-edited version will follow, so please expect rough edges for now.

# cfl

`cfl` is a CLI for Confluence Cloud. It focuses on practical page operations on top of the Confluence REST API v2, with a Markdown-friendly workflow for creating and updating pages.

## What Can It Do?

Current core capabilities:

- Profile management (multiple environments/sites)
- Page list / get / create / update / delete
- Markdown to Confluence storage format conversion (`create` / `update`)
- Frontmatter support (`title`, `parent-id`)
- Local image upload from Markdown to Confluence attachments
- Mermaid fenced code block rendering to images (enabled by default)
- Export Confluence pages as Markdown + attachments (`migrate export`)
- Output formats: `table` / `json`

## Quick Start

### 1. Build

```bash
go build -o cfl .
```

With `mise`:

```bash
mise run build
```

### 2. Set your API token

`cfl` uses Basic Auth:

- Username: the email configured in your profile
- Password: `CFL_API_TOKEN`

```bash
export CFL_API_TOKEN="your_confluence_api_token"
```

### 3. Initial setup (create your first profile)

```bash
./cfl init
```

`cfl init` creates a `default` profile interactively when the config is not initialized yet.

### 4. (Optional) Add and switch profiles

```bash
./cfl config init work
./cfl config init personal
./cfl use work
```

You can also use `cfl config use work`.

### 5. List pages first

If your profile already has `space_key`, you can omit `--space-key`.

```bash
./cfl page list --limit 25
```

Or specify it explicitly:

```bash
./cfl page list --space-key TEST --limit 25
```

## Common User Flows

### 1. Find pages (list, filter, paginate)

```bash
./cfl page list --space-key TEST --status current,archived --sort -modified-date --limit 50
```

- `--status`: `current`, `archived`, `deleted`, `trashed` (comma-separated)
- `--sort`: `id`, `-id`, `created-date`, `-created-date`, `modified-date`, `-modified-date`, `title`, `-title`
- `--cursor`: continue from a previous response's `next`

To get `next` in JSON output:

```bash
./cfl -o json page list --space-key TEST
```

### 2. Get page body (storage format)

```bash
./cfl page get 123456 > page.storage.xhtml
```

`page get` writes the Confluence storage format body directly to stdout (`-o json/table` does not apply).

### 3. Create a page from Markdown

```bash
./cfl page create \
  --space-key TEST \
  --title "Release Notes" \
  --body-file ./docs/release-notes.md
```

- Default `--body-format` is `markdown`
- Use `--body-format storage` to send storage format as-is
- Local images in Markdown are uploaded as attachments and rewritten to `ri:attachment` references
- Mermaid fenced code blocks are rendered to images by default (`--no-render-mermaid` disables this)

### 4. Update an existing page

```bash
./cfl page update 123456 \
  --title "Release Notes v2" \
  --body-file ./docs/release-notes.md
```

- Page version is resolved automatically
- On version conflicts, fetch the latest page state and retry

### 5. Delete a page

```bash
./cfl page delete 123456
```

### 6. Export Confluence pages to Markdown (migration flow)

Export an entire space:

```bash
./cfl migrate export --space-key TEST --out ./export
```

Export only a subtree under a root page:

```bash
./cfl migrate export \
  --space-key TEST \
  --root-page-id 123456 \
  --out ./export
```

- Output is Markdown + frontmatter (`page-id`, `title`, `parent-id`, `space-key`)
- Attachments are saved under `attachments/_migrate` by default
- Override with `--attachments-dir`

## Markdown Workflow Notes

### Frontmatter (`create` / `update`)

If your Markdown file starts with frontmatter, `cfl` can use it for `--title` and `--parent-id`.

```markdown
---
title: Weekly Update
parent-id: "123456"
---

# Summary

- Item 1
- Item 2
```

Behavior:

- If `--title` is not set, frontmatter `title` is used
- If `--parent-id` is not set, frontmatter `parent-id` is used
- Using both flag and frontmatter for the same field is an error
- The frontmatter block is removed before body conversion
- Accepted parent keys: `parent-id`, `parent_id`, `parentid`

### Local images and `--assets-root`

Image path resolution rules for `--body-format markdown`:

- `http://` / `https://`: treated as external URLs
- `./` / `../` / bare path: resolved relative to the `--body-file` directory
- `/`-prefixed paths: resolved relative to `--assets-root` (not OS root)

If `assets_root` is configured in the profile, it is used when `--assets-root` is omitted.

## Configuration (Profiles)

Config file location:

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

[[profiles]]
name = "personal"
domain = "my-site.atlassian.net"
user = "me@example.com"
space_key = "DEV"
output = "json"
```

Main profile commands:

```bash
./cfl config init [name]    # create interactively
./cfl config edit <name>    # edit interactively
./cfl config list           # list profiles
./cfl config show           # show current profile
./cfl config delete <name>  # delete profile
./cfl use [name]            # switch profile (interactive if omitted)
```

## Output Formats

Global flags:

- `-o, --output`: `table` (default) / `json`
- `-p, --profile`: temporary profile override
- `-v, --verbose`: verbose output

Example:

```bash
./cfl -o json page list --space-key TEST
```

Note:

- `page get` always prints the storage body directly and is not affected by `--output`

## Current Command Tree

```text
cfl
├── init
├── config
│   ├── init / edit / use / list / show / delete
├── use
├── page
│   ├── list / get / create / update / delete
└── migrate
    └── export
```

For planned features (labels, folder, attachment, comment, sync, etc.), see `TODO.md`.

## Development

### Tooling

- Go `1.26.0` (matches `go.mod` / `mise.toml`)
- `mise` (optional)

### Common Commands

```bash
mise run all        # build + fmt + lint + test
mise run fmt
mise run lint
mise run test
mise run test-it    # integration tests (build tag)
mise run test-live  # live Confluence tests (CFL_LIVE=1)
mise run cover
mise run build
```

See `TESTING.md` for the testing strategy.

## API / Implementation Notes

- Base API: Confluence Cloud REST API v2
- Some attachment-related behavior uses v1 endpoints (for local image uploads)
- Pagination is cursor-based (`limit` + `cursor`, and response `next`)
