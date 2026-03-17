# Auth Command Spec

## Goal

Provide a lightweight way to persist Confluence credentials locally while keeping environment variables as the highest-priority source.

## Commands

### `cfl auth`

- Alias of `cfl auth login`

### `cfl auth login`

- Stores credentials in `${XDG_CONFIG_HOME:-~/.config}/cflcli/config.yml`
- Supports both flags and interactive prompts
- If one or more values are missing from flags, prompt only for the missing values
- Prompt for `api_token` using hidden input
- Validates the resolved `domain` + `email` + `api_token` combination before saving by default

Supported flags:

- `--domain`
- `--email`
- `--api-token`
- `--no-validate`

Stored YAML shape:

```yaml
domain: example.atlassian.net
email: user@example.com
api_token: xxxxx
```

Behavior:

- Create the config directory if it does not exist
- Overwrite the stored values with the values provided in the current login flow
- Run a single side-effect-free authenticated `GET` request to Confluence before saving
- Validate using the effective `config < env` credential set
- Use a 5 second timeout for validation
- If validation fails, do not update `config.yml`
- `--no-validate` skips the online validation step and allows saving without a network check
- Show the raw validation error message, including HTTP status and response body when available

### `cfl auth logout`

- Deletes `domain`, `email`, and `api_token` from `${XDG_CONFIG_HOME:-~/.config}/cflcli/config.yml`
- If the config file does not exist, treat logout as a successful no-op
- `logout` does not unset environment variables

## Credential Resolution

Runtime credential lookup is resolved per key, not as an all-or-nothing source switch.

Resolution order:

- `domain`: `CONFLUENCE_DOMAIN`, then `config.yml`
- `email`: `CONFLUENCE_EMAIL`, then `ATLASSIAN_EMAIL`, then `config.yml`
- `api_token`: `CONFLUENCE_API_TOKEN`, then `ATLASSIAN_API_TOKEN`, then `config.yml`

Examples:

- If only `CONFLUENCE_DOMAIN` is set, use that domain and load `email` and `api_token` from `config.yml`
- If `CONFLUENCE_API_TOKEN` is set, use that token even when `config.yml` also contains `api_token`

## Validation

- Require all three resolved keys before creating the HTTP client
- Login validation checks the credential combination, not `api_token` in isolation

## Non-Goals

- OAuth or browser-based login
- Multiple named profiles
- Keychain or OS-native secret storage
