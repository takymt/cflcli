# Page MVP Implementation Plan

## Goal

Implement the MVP defined in `docs/product-specs/page-mvp.md` with the shortest path to a usable `cfl page new` and `cfl page sync`.

## Execution Strategy

Build the CLI in thin vertical slices so that each step leaves the repository in a runnable state.

The implementation order should optimize for:

- early validation of Confluence API assumptions
- minimal rework across command wiring and shared page logic
- a working non-watch sync flow before adding file watching

## Phase 1: Project Skeleton

### Scope

- add the Go entrypoint
- add the root CLI command
- add the `page` command group
- add placeholder `new` and `sync` subcommands
- define package layout for CLI, Confluence client, frontmatter, and Markdown conversion

### Deliverable

The binary can run:

```bash
cfl page new --help
cfl page sync --help
```

### Exit Criteria

- command tree is wired
- flag parsing works
- help text reflects the agreed MVP commands

## Phase 2: Config and Confluence Client

### Scope

- load Confluence configuration from environment variables
- validate required configuration at startup
- implement a minimal Confluence Cloud client for:
  - creating a page
  - updating a page
  - resolving the root page id for a space
  - detecting duplicate page titles under the same parent
  - building the canonical page URL for output

### Deliverable

There is a reusable client package that supports the MVP page flows.

### Exit Criteria

- env validation is explicit and actionable
- API errors are wrapped into user-facing messages
- root page resolution and duplicate title checks are implemented

## Phase 3: Frontmatter and File Model

### Scope

- parse YAML frontmatter from a single Markdown file
- validate the required keys: `space-id`, `page-id`, `parent-id`
- generate a new file for `cfl page new`
- derive page title from the file basename
- preserve file content rules for UTF-8 Markdown

### Deliverable

The repository has a single file model used by both `new` and `sync`.

### Exit Criteria

- invalid or missing frontmatter returns early errors
- `new` can generate a Markdown file with the required frontmatter
- file title derivation matches the product spec

## Phase 4: `cfl page new`

### Scope

- implement the full `new` command flow
- fail if the target file already exists
- resolve `parent-id` from the space root when not provided
- fail if a duplicate page title already exists under the target parent
- create the remote Confluence page
- create the local Markdown file with the resolved ids
- print a single-line success message with the page URL

### Deliverable

`cfl page new <title>.md --space-id <space-id> [--parent-id <parent-id>]` works end to end.

### Exit Criteria

- local and remote resources are both created
- resolved ids are written into frontmatter
- duplicate title and file existence checks are enforced

## Phase 5: Markdown Conversion

### Scope

- implement the minimum Markdown-to-Confluence conversion required by the MVP
- support:
  - paragraph
  - heading
  - list
  - code block

### Deliverable

The sync flow can transform supported Markdown into the Confluence body format accepted by the API.

### Exit Criteria

- empty body is supported
- unsupported constructs fail predictably or degrade in a controlled way
- conversion logic is isolated from command wiring

## Phase 6: `cfl page sync`

### Scope

- implement the full `sync` command flow
- require valid YAML frontmatter
- load `space-id`, `page-id`, and `parent-id` from the file
- update the remote page title from the file basename
- overwrite the full remote page body with the local Markdown content
- print a single-line success message with the page URL

### Deliverable

`cfl page sync <file>.md` works end to end.

### Exit Criteria

- sync succeeds with an empty body
- title and body both update on every run
- frontmatter validation errors are clear and early

## Phase 7: `--watch`

### Scope

- add single-file watch mode to `cfl page sync`
- run one initial sync on startup
- debounce file changes with a fixed `800ms` interval
- print a green `.` on success
- print a red `!` and an error message on failure
- continue watching after errors

### Deliverable

`cfl page sync <file>.md --watch` continuously updates the target Confluence page.

### Exit Criteria

- only the target file is watched
- repeated save bursts collapse into one sync after `800ms`
- watch mode does not exit on normal sync errors

## Phase 8: Validation and Repository Polish

### Scope

- add focused tests for frontmatter parsing, title derivation, and Markdown conversion
- add focused tests for command validation paths where practical
- document required environment variables and command examples
- verify formatting and buildability

### Deliverable

The MVP is documented, testable, and ready for iterative feature work.

### Exit Criteria

- core packages have targeted test coverage
- docs cover setup and the basic page workflows
- the project builds cleanly

## Suggested Implementation Order

1. Phase 1: Project Skeleton
2. Phase 2: Config and Confluence Client
3. Phase 3: Frontmatter and File Model
4. Phase 4: `cfl page new`
5. Phase 5: Markdown Conversion
6. Phase 6: `cfl page sync`
7. Phase 7: `--watch`
8. Phase 8: Validation and Repository Polish

## Risks To Validate Early

- Confluence Cloud API details for root page resolution
- duplicate title detection under a specific parent
- the body representation required for create and update requests
- how strict the frontmatter parser should be in practice

## Notes

- This plan intentionally excludes attachments and Mermaid handling
- This plan assumes the local Markdown file remains the source of truth
- If API constraints differ from the product spec, update the spec before broad implementation work
