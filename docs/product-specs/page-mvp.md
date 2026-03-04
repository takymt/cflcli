# Confluence Page CLI MVP

## Goal

Provide a CLI that can create and update Confluence Cloud pages from a single UTF-8 Markdown file.

The MVP scope is limited to page-related functionality only.

## Scope

The MVP provides the following two commands:

- `cfl page new <title>.md --space-id <space-id> [--parent-id <parent-id>]`
- `cfl page sync <file>.md [--watch]`

## Platform Assumptions

- The target Confluence deployment is Cloud only
- Authentication is provided through environment variables
- Input is limited to a single UTF-8 Markdown file
- The local file is always treated as the source of truth and overwrites the remote page content

## Command: `cfl page new`

### Purpose

Create a new local Markdown file and a new Confluence page.

### Input

```bash
cfl page new <title>.md --space-id <space-id> [--parent-id <parent-id>]
```

### Behavior

- Use the basename of `<title>.md` as the page title
- Return an error if the target file already exists
- Return an error if a page with the same title already exists under the same parent in Confluence
- If `parent-id` is omitted, resolve and use the root page id of the specified space
- Create a new local Markdown file
- Create a new page in Confluence
- Write YAML frontmatter at the top of the generated file
- Leave the initial body empty

### Generated Frontmatter

```yaml
---
space-id: 12345
page-id: 67890
parent-id: 11111
---
```

### Output

- On success, print a single-line message that includes the page URL
- On failure, return exit code `1` with a clear error message

## Command: `cfl page sync`

### Purpose

Synchronize an existing Markdown file with valid frontmatter to a Confluence page.

### Input

```bash
cfl page sync <file>.md
cfl page sync <file>.md --watch
```

### Preconditions

- YAML frontmatter is required
- `space-id`, `page-id`, and `parent-id` are required
- Return an early error if the frontmatter does not match the expected format

### Behavior

- Use the basename as the page title
- Convert the Markdown body to Confluence format using a minimal feature set
- Allow syncing even when the body is empty
- Update both the page title and body on every sync
- Ignore manual edits made in Confluence and always overwrite the full remote page content with the local content

### Output

- On normal success, print a single-line message that includes the page URL
- On failure, return exit code `1` with a clear error message

## `--watch`

### Purpose

Watch the target Markdown file and sync it automatically on changes.

### Behavior

- Run `sync` once at startup
- Watch a single file only
- After a file change is detected, wait until `800ms` have passed since the last change and then sync again
- The debounce interval is fixed at `800ms`
- On success, print only a green `.`
- On failure, print a red `!` and an error message
- Keep watching after sync failures instead of exiting

## Frontmatter Rules

- Only YAML frontmatter is allowed
- The key names are fixed to `space-id`, `page-id`, and `parent-id`
- ID types must match the Confluence API
- Do not store `title` in frontmatter; always derive it from the basename

## Markdown Conversion

The MVP only needs a minimal Markdown-to-Confluence conversion.

Initial support includes:

- paragraph
- heading
- list
- code block

## Non-Goals

The following items are out of scope for the MVP:

- image upload
- general attachments
- local Mermaid rendering and attachment upload
- multi-file or directory sync
- JSON output
- Confluence Server or Data Center support
- conflict detection against remote Confluence edits
