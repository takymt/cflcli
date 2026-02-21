# Testing Strategy

This repository uses a three-layer testing strategy modeled after the approach used in `gogcli`.

## Scope And Granularity

1. Unit tests (`UT`)

- Fast, deterministic tests that run with `go test ./...`.
- Use stdlib `testing` and `net/http/httptest`.
- Focus on:
  - validation and parsing logic
  - output contracts (`json` and table)
  - request construction (query params, auth headers)
  - error paths

2. Integration tests (`IT`, opt-in)

- Build-tagged tests in `internal/integration/`.
- Hit live Confluence Cloud APIs with real credentials.
- Not run by default in CI.
- Command:
  - `go test -tags=integration ./internal/integration`

3. End-to-end smoke tests (`E2E`, opt-in)

- Live CLI checks through `scripts/live-test.sh`.
- Executed via integration wrapper test:
  - `CFL_LIVE=1 go test -tags=integration ./internal/integration -run Live`
- Intended to verify real CLI flows across config + command execution.

## Test Case Selection

Prioritize cases that catch regressions in CLI behavior:

- Output compatibility:
  - `--output json` payload shape
  - table headers / key columns
- Input validation:
  - boundary values (`limit`, required args)
  - invalid enum values
- API behavior:
  - expected query/auth headers
  - non-2xx responses and error message propagation
- Configuration flows:
  - profile selection and fallback behavior
  - config persistence and current-profile handling

## Value Heuristics

When adding or keeping tests, optimize for maintenance value over branch count:

- Prefer behavior-level tests at command/API boundaries over direct tests of private helper functions.
- Avoid asserting internal call order, temporary variables, or implementation-only structure.
- Avoid duplicate coverage of the same behavior across multiple layers unless each layer has distinct user-facing risk.
- For parser/validation logic, keep only representative boundary/error cases that protect CLI contracts.
- Remove or merge tests that fail only after harmless refactors (renames/extractions) but do not detect behavior regressions.

## Coverage Policy

- No hard coverage threshold in CI for now.
- Coverage is still measured and should be monitored:
  - `go test -cover ./cmd/... ./internal/...`
- Prefer adding focused tests for changed behavior instead of chasing a single number.

## Naming Conventions

Test files:

- Standard: `*_test.go`
- Additional focused cases: `*_more_test.go` (optional)
- Integration: `internal/integration/*` with `//go:build integration`

Test function names:

- `Test<Subject>_<Scenario>`
- Include output mode or behavior when relevant:
  - `TestRunPageListWithConfig_JSON`
  - `TestRunPageListWithConfig_Table`
  - `TestNormalizeOutputFormat_Invalid`

Subtests:

- Prefer table-driven tests with `t.Run(caseName, ...)`.

## Commands

- Unit tests:
  - `go test ./...`
- Integration tests:
  - `go test -tags=integration ./internal/integration`
- Live E2E wrapper:
  - `CFL_LIVE=1 go test -tags=integration ./internal/integration -run Live`
- Coverage:
  - `go test -cover ./cmd/... ./internal/...`
