# cflcli

`cflcli` is a CLI for managing Confluence content with local Markdown files.

## Key Concept
- Treat Markdown as the source of truth
- Manage related files as attachments

## Features
- `cfl page new` creates a Confluence page and generates a Markdown template
- `cfl page sync` converts Markdown to Confluence Storage Format and updates the page
- `cfl page sync --watch` syncs automatically on each save
- `cfl page new --watch` creates and then continuously syncs in a single command
- Converts Mermaid blocks (` ```mermaid `) to SVG and embeds them as attachments
- Resolves and embeds relative files
- Uses diff/cache-based acceleration for faster sync

## Requirements
- Go 1.26+
- [mmdc](https://github.com/mermaid-js/mermaid-cli) (required only when using Mermaid)
- Confluence Cloud access

## Installation

```bash
go install github.com/takymt/cflcli/cmd/cfl@latest
```

```bash
cfl --help
```

## Environment Variables

- `CONFLUENCE_DOMAIN` (e.g. `example.atlassian.net`)
- `CONFLUENCE_EMAIL`
- `CONFLUENCE_API_TOKEN`

## Quick Start

```bash
cfl page new docs/cfl.md --space-key TEST --parent-id 123456 --watch
```

## Markdown Support

See [markdown-syntax.md](/docs/markdown-syntax.md).

## Performance / Cache

Caching is used to reduce rendering and network overhead.

Cache files are created per markdown file path hash:

- `${XDG_CACHE_HOME:-~/.cache}/cflcli/mermaid/<sha256(abs-markdown-path)>.json`
- `${XDG_CACHE_HOME:-~/.cache}/cflcli/attachments/<sha256(abs-markdown-path)>.json`

On macOS, this is typically:

- `~/Library/Caches/cflcli/mermaid/<sha256>.json`
- `~/Library/Caches/cflcli/attachments/<sha256>.json`

## Author

@takymt (a.k.a tarte)

## License
See [LICENSE](/LICENSE).
