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

**Note**: Mermaid diagram conversion requires [mmdc](https://github.com/mermaid-js/mermaid-cli)

## Installation

### Homebrew

```bash
brew install takymt/tap/cflcli
```

### go install

```sh
go install github.com/takymt/cflcli/cmd/cfl@latest
```

### Download Binary

Download from [GitHub Releases](https://github.com/takymt/cflcli/releases)

```bash
cfl --help
```

## Authentication

`cfl` resolves credentials per key in this order:

- environment variables
- `${XDG_CONFIG_HOME:-~/.config}/cflcli/config.yml`

Supported environment variables:

- `CONFLUENCE_DOMAIN` (e.g. `example.atlassian.net`)
- `CONFLUENCE_EMAIL`
- `CONFLUENCE_API_TOKEN`

Save credentials locally:

```bash
cfl auth

# Or pass values explicitly
cfl auth login --domain example.atlassian.net --email user@example.com --api-token <token>

# Or skip validation
cfl auth login --no-validate
```

Clear the saved config:

```bash
cfl auth logout
```

## Quick Start

```bash
cfl page new --title "CFL CLI" --space-key TEST --parent-id 123456 --path docs/cfl.md --watch
```

## Usage

```console
Usage:
  cfl [command]

Available Commands:
  attachment  Manage page attachments
  auth        Manage Confluence authentication
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  page        Manage Confluence pages

Flags:
  -h, --help   help for cfl

Use "cfl [command] --help" for more information about a command.
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
