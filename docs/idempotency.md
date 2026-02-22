# Idempotency Notes (Phase 1.4)

This document records the current idempotency target for migrate export related conversion:

- pipeline: `storage -> markdown -> storage -> markdown`
- objective: migration-critical constructs remain traceable and reusable

## Current Acceptance Cases

1. Mermaid macro survives roundtrip

- input storage contains Confluence `code` macro with `language=mermaid`
- after one roundtrip, markdown still contains:
  - ```` ```mermaid ````
  - original diagram source text

2. Unsupported macro trace remains

- input storage contains unsupported macro (example: `toc`)
- after one roundtrip, markdown still contains:
  - `cfl:migrate-unsupported-macro`
  - macro `name="..."`
  - `storage-base64="..."`

3. Attachment path remains under migrate attachment directory

- input storage contains `ri:attachment`
- after one roundtrip, markdown still contains attachment path under:
  - `attachments/_migrate/<page-id>/<filename>`

## Test Coverage

These acceptance cases are implemented in:

- `internal/migrate/idempotency_test.go`
