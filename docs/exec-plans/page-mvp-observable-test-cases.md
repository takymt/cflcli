# Page MVP Observable Test Cases

## Purpose

This document defines test cases as a mapping of the observable behavior expected from the `page-mvp` product.

The focus is user-visible behavior:

- command input
- preconditions
- observable output
- observable local file changes
- observable remote Confluence changes

Implementation details, environment constraints, and tool constraints are intentionally ignored.

## Conventions

- "local file" means the Markdown file operated on by the CLI
- "remote page" means the corresponding Confluence page
- "success message" means a single-line message that includes the page URL
- "early error" means the command fails before performing remote mutation

## Test Cases: `cfl page new`

### NEW-001 Create a page with an explicit parent

#### Given

- no local file exists at `guide.md`
- the target `space-id` exists
- the target `parent-id` exists
- no page with title `guide` exists under that parent

#### When

```bash
cfl page new guide.md --space-id 100 --parent-id 200
```

#### Then

- the command succeeds
- a local file `guide.md` is created
- the local file contains YAML frontmatter with `space-id`, `page-id`, and `parent-id`
- the stored `space-id` is `100`
- the stored `parent-id` is `200`
- the local Markdown body is empty
- a remote page titled `guide` is created under parent `200`
- the command prints a success message with the page URL

### NEW-002 Create a page without an explicit parent

#### Given

- no local file exists at `guide.md`
- the target `space-id` exists
- no `parent-id` is provided
- no page with title `guide` exists under the root page of the target space

#### When

```bash
cfl page new guide.md --space-id 100
```

#### Then

- the command succeeds
- a local file `guide.md` is created
- the local file contains YAML frontmatter with `space-id`, `page-id`, and `parent-id`
- the stored `space-id` is `100`
- the stored `parent-id` is the resolved root page id of the space
- a remote page titled `guide` is created under the resolved root page
- the command prints a success message with the page URL

### NEW-003 Fail when the local file already exists

#### Given

- a local file already exists at `guide.md`

#### When

```bash
cfl page new guide.md --space-id 100 --parent-id 200
```

#### Then

- the command fails
- the command returns exit code `1`
- the command prints a clear error message
- no remote page is created
- the existing local file is not modified

### NEW-004 Fail when a duplicate page title exists under the same parent

#### Given

- no local file exists at `guide.md`
- a remote page titled `guide` already exists under parent `200`

#### When

```bash
cfl page new guide.md --space-id 100 --parent-id 200
```

#### Then

- the command fails
- the command returns exit code `1`
- the command prints a clear error message
- no local file is created
- no additional remote page titled `guide` is created under parent `200`

### NEW-005 Use basename as the page title

#### Given

- no local file exists at `architecture-overview.md`
- the target parent contains no page titled `architecture-overview`

#### When

```bash
cfl page new architecture-overview.md --space-id 100 --parent-id 200
```

#### Then

- the command succeeds
- the created remote page title is exactly `architecture-overview`
- the generated local file path is `architecture-overview.md`

## Test Cases: `cfl page sync`

### SYNC-001 Sync a valid Markdown file

#### Given

- a local file `guide.md` exists
- the local file contains valid YAML frontmatter with `space-id`, `page-id`, and `parent-id`
- the local file contains a non-empty Markdown body
- the referenced remote page exists

#### When

```bash
cfl page sync guide.md
```

#### Then

- the command succeeds
- the remote page title becomes `guide`
- the remote page body is replaced with the converted content of the local Markdown body
- the command prints a success message with the page URL
- the local file is not rewritten by the command

### SYNC-002 Sync an empty body

#### Given

- a local file `guide.md` exists
- the local file contains valid YAML frontmatter with `space-id`, `page-id`, and `parent-id`
- the local Markdown body is empty
- the referenced remote page exists

#### When

```bash
cfl page sync guide.md
```

#### Then

- the command succeeds
- the remote page title becomes `guide`
- the remote page body becomes empty
- the command prints a success message with the page URL

### SYNC-003 Fail when frontmatter is missing

#### Given

- a local file `guide.md` exists
- the file contains Markdown content without YAML frontmatter

#### When

```bash
cfl page sync guide.md
```

#### Then

- the command fails with an early error
- the command returns exit code `1`
- the command prints a clear error message
- the remote page is not modified
- the local file is not modified

### SYNC-004 Fail when a required frontmatter key is missing

#### Given

- a local file `guide.md` exists
- the file contains YAML frontmatter
- one of `space-id`, `page-id`, or `parent-id` is missing

#### When

```bash
cfl page sync guide.md
```

#### Then

- the command fails with an early error
- the command returns exit code `1`
- the command prints a clear error message naming the validation problem
- the remote page is not modified

### SYNC-005 Fail when frontmatter format is invalid

#### Given

- a local file `guide.md` exists
- the file begins with malformed YAML frontmatter

#### When

```bash
cfl page sync guide.md
```

#### Then

- the command fails with an early error
- the command returns exit code `1`
- the command prints a clear error message
- the remote page is not modified

### SYNC-006 Sync updates the title from the basename

#### Given

- a local file `renamed-guide.md` exists
- the file contains valid YAML frontmatter with an existing `page-id`
- the referenced remote page currently has a different title

#### When

```bash
cfl page sync renamed-guide.md
```

#### Then

- the command succeeds
- the remote page title becomes `renamed-guide`
- the command prints a success message with the page URL

### SYNC-007 Sync overwrites remote manual edits

#### Given

- a local file `guide.md` exists with valid frontmatter
- the referenced remote page exists
- the remote page body contains manual edits not present in the local file

#### When

```bash
cfl page sync guide.md
```

#### Then

- the command succeeds
- the remote page body exactly reflects the local Markdown content after conversion
- the remote manual edits no longer remain if they are absent from the local file

### SYNC-008 Sync supports the documented Markdown syntax subset

#### Given

- a local file contains valid frontmatter
- the Markdown body contains the following syntax:
  - plain paragraphs separated by blank lines
  - ATX headings `#`, `##`, and `###`
  - unordered lists using `-`
  - ordered lists using `1.`
  - fenced code blocks using triple backticks

#### When

```bash
cfl page sync guide.md
```

#### Then

- the command succeeds
- plain paragraphs are rendered as paragraphs in the remote page
- `#`, `##`, and `###` headings are rendered as headings at the expected levels in the remote page
- unordered lists using `-` are rendered as unordered lists in the remote page
- ordered lists using `1.` are rendered as ordered lists in the remote page
- fenced code blocks are rendered as code blocks in the remote page

## Test Cases: `cfl page sync --watch`

### WATCH-001 Run an initial sync on startup

#### Given

- a local file `guide.md` exists with valid frontmatter
- the referenced remote page exists

#### When

```bash
cfl page sync guide.md --watch
```

#### Then

- the command starts successfully
- an initial sync is performed before waiting for further file changes

### WATCH-002 Sync again after a file change

#### Given

- watch mode is already running for `guide.md`
- the local file content changes once

#### When

- more than `800ms` pass after the last observed change

#### Then

- the remote page is updated to match the latest local file content
- the command prints a green `.`

### WATCH-003 Debounce a burst of rapid file changes

#### Given

- watch mode is already running for `guide.md`
- the local file changes multiple times within `800ms`

#### When

- the file stops changing and `800ms` pass

#### Then

- only one sync is observed for the burst
- the remote page reflects the final local file content after the burst
- the command prints one green `.`

### WATCH-004 Continue watching after a sync failure

#### Given

- watch mode is already running for `guide.md`
- a file change introduces invalid frontmatter

#### When

- more than `800ms` pass after the last observed change

#### Then

- the command prints a red `!` and an error message
- the process keeps running

#### And When

- the file is fixed and changed again
- more than `800ms` pass after the last observed change

#### Then

- sync succeeds
- the command prints a green `.`

### WATCH-005 Watch mode observes only the target file

#### Given

- watch mode is already running for `guide.md`
- unrelated files in the same directory change

#### When

- those unrelated file changes occur

#### Then

- no sync is triggered for unrelated file changes
- no green `.` is printed for unrelated file changes

## Error Handling Coverage

The product should expose clear user-visible errors for at least the following cases:

- target file already exists during `new`
- duplicate page title during `new`
- missing or malformed frontmatter during `sync`
- missing required frontmatter keys during `sync`
- invalid Confluence identifiers
- missing authentication configuration
- remote page not found during `sync`
- permission or authorization failures

## Acceptance Summary

The MVP is behaviorally complete when:

- `cfl page new` creates both the local Markdown file and the remote page
- `cfl page sync` updates the remote page title and body from a valid local file
- `cfl page sync --watch` keeps the remote page up to date after file changes
- invalid inputs fail early with clear messages
- successful flows expose a URL to the created or updated page
