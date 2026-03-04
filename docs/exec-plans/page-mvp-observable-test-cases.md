# Page MVP Observable Test Cases

## Purpose

This document describes the observable behavior of `page-mvp` in an FP-style layout:

- `Purpose`
- `Preconditions`
- `Procedure`
- `Expected`

The focus is user-visible behavior only.

## Definitions

- `local file`: the Markdown file operated on by the CLI
- `remote page`: the corresponding page in Confluence
- `early error`: a failure before any remote mutation is performed
- `success message`: a single-line message that includes the page URL

## Feature: `cfl page new`

### FP-NEW-001 Create a page with an explicit parent

Purpose:
Create a new local Markdown file and a new remote page under the specified parent.

Preconditions:
- no local file exists at `guide.md`
- `space-id=100` exists
- `parent-id=200` exists
- no page titled `guide` exists under parent `200`

Procedure:
```bash
cfl page new guide.md --space-id 100 --parent-id 200
```

Expected:
- the command succeeds
- `guide.md` is created locally
- the frontmatter contains `space-id`, `page-id`, and `parent-id`
- the stored `space-id` is `100`
- the stored `parent-id` is `200`
- the local body is empty
- a remote page titled `guide` is created under parent `200`
- a success message with the page URL is printed

### FP-NEW-002 Create a page without an explicit parent

Purpose:
Create a new local Markdown file and a new remote page under the resolved space root.

Preconditions:
- no local file exists at `guide.md`
- `space-id=100` exists
- no explicit parent is provided
- no page titled `guide` exists under the root page of the target space

Procedure:
```bash
cfl page new guide.md --space-id 100
```

Expected:
- the command succeeds
- `guide.md` is created locally
- the frontmatter contains `space-id`, `page-id`, and `parent-id`
- the stored `space-id` is `100`
- the stored `parent-id` is the resolved root page id
- a remote page titled `guide` is created under the resolved root page
- a success message with the page URL is printed

### FP-NEW-003 Reject an existing local target file

Purpose:
Prevent overwriting an existing Markdown file during `new`.

Preconditions:
- a local file already exists at `guide.md`

Procedure:
```bash
cfl page new guide.md --space-id 100 --parent-id 200
```

Expected:
- the command fails
- the exit code is `1`
- a clear error message is printed
- no remote page is created
- the existing local file is unchanged

### FP-NEW-004 Reject a duplicate title under the same parent

Purpose:
Prevent creating two sibling pages with the same title.

Preconditions:
- no local file exists at `guide.md`
- a remote page titled `guide` already exists under parent `200`

Procedure:
```bash
cfl page new guide.md --space-id 100 --parent-id 200
```

Expected:
- the command fails
- the exit code is `1`
- a clear error message is printed
- no local file is created
- no additional remote page titled `guide` is created under parent `200`

### FP-NEW-005 Use the basename as the title

Purpose:
Derive the page title from the Markdown filename.

Preconditions:
- no local file exists at `architecture-overview.md`
- the target parent contains no page titled `architecture-overview`

Procedure:
```bash
cfl page new architecture-overview.md --space-id 100 --parent-id 200
```

Expected:
- the command succeeds
- the remote title is exactly `architecture-overview`
- the generated local file path is `architecture-overview.md`

## Feature: `cfl page sync`

### FP-SYNC-001 Sync a valid Markdown file

Purpose:
Update the remote page title and body from a valid local file.

Preconditions:
- `guide.md` exists locally
- the file has valid YAML frontmatter with `space-id`, `page-id`, and `parent-id`
- the Markdown body is non-empty
- the referenced remote page exists

Procedure:
```bash
cfl page sync guide.md
```

Expected:
- the command succeeds
- the remote title becomes `guide`
- the remote body is replaced with the converted Markdown body
- a success message with the page URL is printed
- the local file is not rewritten

### FP-SYNC-002 Sync an empty body

Purpose:
Allow sync even when the local Markdown body is empty.

Preconditions:
- `guide.md` exists locally
- the file has valid YAML frontmatter with `space-id`, `page-id`, and `parent-id`
- the Markdown body is empty
- the referenced remote page exists

Procedure:
```bash
cfl page sync guide.md
```

Expected:
- the command succeeds
- the remote title becomes `guide`
- the remote body becomes empty
- a success message with the page URL is printed

### FP-SYNC-003 Reject a file without frontmatter

Purpose:
Fail early when the local file has no YAML frontmatter.

Preconditions:
- `guide.md` exists locally
- the file contains Markdown content without YAML frontmatter

Procedure:
```bash
cfl page sync guide.md
```

Expected:
- the command fails with an early error
- the exit code is `1`
- a clear error message is printed
- the remote page is not modified
- the local file is not modified

### FP-SYNC-004 Reject missing required frontmatter keys

Purpose:
Fail early when one of the required identifiers is missing.

Preconditions:
- `guide.md` exists locally
- the file contains YAML frontmatter
- one of `space-id`, `page-id`, or `parent-id` is missing

Procedure:
```bash
cfl page sync guide.md
```

Expected:
- the command fails with an early error
- the exit code is `1`
- a clear validation error is printed
- the remote page is not modified

### FP-SYNC-005 Reject malformed frontmatter

Purpose:
Fail early when the YAML frontmatter cannot be parsed.

Preconditions:
- `guide.md` exists locally
- the file begins with malformed YAML frontmatter

Procedure:
```bash
cfl page sync guide.md
```

Expected:
- the command fails with an early error
- the exit code is `1`
- a clear error message is printed
- the remote page is not modified

### FP-SYNC-006 Update the title from the basename

Purpose:
Force the remote title to follow the local filename.

Preconditions:
- `renamed-guide.md` exists locally
- the file has valid YAML frontmatter with an existing `page-id`
- the referenced remote page currently has a different title

Procedure:
```bash
cfl page sync renamed-guide.md
```

Expected:
- the command succeeds
- the remote title becomes `renamed-guide`
- a success message with the page URL is printed

### FP-SYNC-007 Overwrite remote manual edits

Purpose:
Treat the local file as the source of truth during sync.

Preconditions:
- `guide.md` exists locally with valid frontmatter
- the referenced remote page exists
- the remote body contains manual edits not present in the local file

Procedure:
```bash
cfl page sync guide.md
```

Expected:
- the command succeeds
- the remote body exactly reflects the converted local Markdown content
- remote manual edits not present in the local file are removed

### FP-SYNC-008 Support the documented Markdown subset

Purpose:
Verify the MVP Markdown subset that must survive conversion.

Preconditions:
- a local file contains valid frontmatter
- the body contains:
  - paragraphs separated by blank lines
  - ATX headings `#`, `##`, and `###`
  - unordered lists using `-`
  - ordered lists using `1.`
  - fenced code blocks using triple backticks

Procedure:
```bash
cfl page sync guide.md
```

Expected:
- the command succeeds
- paragraphs are rendered as paragraphs in the remote page
- `#`, `##`, and `###` are rendered at the matching heading levels
- unordered lists are rendered as unordered lists
- ordered lists are rendered as ordered lists
- fenced code blocks are rendered as code blocks

## Feature: `cfl page sync --watch`

### FP-WATCH-001 Run an initial sync on startup

Purpose:
Ensure watch mode begins from a synchronized state.

Preconditions:
- `guide.md` exists locally with valid frontmatter
- the referenced remote page exists

Procedure:
```bash
cfl page sync guide.md --watch
```

Expected:
- the command starts successfully
- one initial sync is performed before waiting for further changes

### FP-WATCH-002 Sync again after a file change

Purpose:
Refresh the remote page after a single file change.

Preconditions:
- watch mode is already running for `guide.md`
- the local file changes once

Procedure:
- wait until more than `800ms` pass after the last observed change

Expected:
- the remote page is updated to match the latest local file content
- the command prints a green `.`

### FP-WATCH-003 Debounce a rapid burst of file changes

Purpose:
Collapse multiple quick file changes into one sync.

Preconditions:
- watch mode is already running for `guide.md`
- the local file changes multiple times within `800ms`

Procedure:
- stop changing the file
- wait until `800ms` pass

Expected:
- only one sync is observed for the burst
- the remote page reflects the final local file content after the burst
- the command prints one green `.`

### FP-WATCH-004 Continue watching after a sync failure

Purpose:
Keep watch mode alive across transient sync errors.

Preconditions:
- watch mode is already running for `guide.md`
- a file change introduces invalid frontmatter

Procedure:
- wait until more than `800ms` pass after the last observed change
- fix the file
- change the file again
- wait until more than `800ms` pass after the last observed change

Expected:
- the first failed sync prints a red `!` and an error message
- the process keeps running after the failure
- the next sync succeeds
- the command prints a green `.`

### FP-WATCH-005 Ignore unrelated file changes

Purpose:
Watch only the target file.

Preconditions:
- watch mode is already running for `guide.md`
- unrelated files in the same directory change

Procedure:
- change unrelated files only

Expected:
- no sync is triggered for unrelated file changes
- no green `.` is printed for unrelated file changes
