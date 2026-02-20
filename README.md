# cfl

`cfl` is a CLI for Confluence Cloud REST API v2.

It focuses on profile-based configuration, page operations, and practical Markdown-to-Confluence body conversion for `create` / `update`.

## Features

- Profile management (`init`, `config init/edit/list/show/delete`, `use`)
- Page operations (`list`, `get`, `create`, `update`, `delete`)
- `markdown` and `storage` body input for page create/update
- Frontmatter support for page metadata (`title`, `parent-id`)
- Output modes: `table` (default) and `json`

## Requirements

- Go `1.26.0`
- Confluence Cloud account
- Confluence API token (set via environment variable)

## Build

```bash
go build -o cfl .
```

Or with `mise`:

```bash
mise run build
```

## Authentication

`cfl` uses Basic Auth with:

- user email from the selected profile
- API token from `CONFLUENCE_API_TOKEN`

```bash
export CONFLUENCE_API_TOKEN="your_api_token"
```

## Configuration

Config file location:

- `$XDG_CONFIG_HOME/cflcli/config.toml`
- default: `~/.config/cflcli/config.toml`

Initialize config interactively:

```bash
cfl init
```

Manage profiles:

```bash
cfl config init work
cfl config edit work
cfl config list
cfl config show
cfl use work
```

Example `config.toml`:

```toml
current = "work"

[[profiles]]
name = "work"
domain = "your-domain.atlassian.net"
user = "you@example.com"
space_key = "TEST"
output = "table"
```

## Common Commands

List pages:

```bash
cfl page list --space-key TEST --limit 25
```

Get page body (storage format):

```bash
cfl page get 123456
```

Create a page from Markdown:

```bash
cfl page create --space-key TEST --title "Release Notes" --body-file page.md
```

Update a page:

```bash
cfl page update 123456 --title "Release Notes v2" --body-file page.md
```

Delete a page:

```bash
cfl page delete 123456
```

Use JSON output:

```bash
cfl -o json page list --space-key TEST
```

## Body Input And Frontmatter

`--body-format` supports:

- `markdown` (default)
- `storage`

When using Markdown body files, frontmatter is supported:

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

- If `--title` is not set, `title` from frontmatter is used.
- If `--parent-id` is not set, `parent-id` from frontmatter is used.
- If both flag and frontmatter are set for the same field, command returns an error.
- Frontmatter block is removed before body conversion.

Accepted keys for parent id: `parent-id`, `parent_id`, `parentid`.

## Development

Run all checks:

```bash
mise run all
```

Useful tasks:

```bash
mise run fmt
mise run lint
mise run test
mise run test-it
mise run test-live
```
