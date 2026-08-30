# Configuration Reference

> [中文](configuration.md) | English

Configuration precedence: **built-in defaults < TOML config file < environment variables**. Use either approach alone, or mix them (env only overrides non-empty values).

```bash
newapi-mcp --config /path/to/newapi-mcp.toml   # without --config, only defaults + environment variables apply
```

## `[newapi]` Section (gateway connection, required)

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `base_url` | string | ✅ | — | Gateway address, **no trailing slash** (a trailing slash passed via env is stripped automatically) |
| `token` | string | one of the two | — | Dashboard PAT. Prefer leaving it empty and using `token_file` instead, to keep the secret out of the config file itself |
| `token_file` | string | one of the two | — | Path to a PAT file: **the first line** is read as the token. File permission `0600` recommended |
| `writemode` | string | — | `read` | Tool tier: `read` (11 tools) / `ops` (+6) / `admin` (+6). Lower tiers **do not register** higher-tier tools |
| `timeout_seconds` | int | — | `10` | Upstream HTTP timeout in seconds; ≤0 falls back to 10 |

When `token` and `token_file` are both configured, `token` takes precedence; when neither is set, startup fails (same for a missing `base_url`).

## `[report]` Section (reporting read replica, optional)

The data source of the `newapi_jiyuan_report` tool — it queries a **MySQL reporting read replica** directly to aggregate the `logs` and `model_price_snapshots` tables, bypassing the new-api HTTP API. When unconfigured, the tool stays registered and calls fail with an explicit "reporting not configured" error (no silent degradation).

| Field | Type | Description |
|---|---|---|
| `db_dsn` | string | Direct DSN: `user:pass@tcp(host:port)/db?charset=utf8mb4`. Storing it in plaintext is not recommended |
| `db_dsn_file` | string | Path to a DSN file: the first line is read, permission `0600` (recommended) |

DSN resolution precedence: **`NEWAPI_REPORT_DB_DSN` env > `db_dsn` > first line of `db_dsn_file`**. A file read failure / empty first line is treated as "not configured" (it does not block MCP startup; reporting is an optional capability).

## Environment Variable Overrides

| Environment Variable | Overrides | Description |
|---|---|---|
| `NEWAPI_BASE_URL` | `newapi.base_url` | Overrides only when non-empty; trailing slash stripped automatically |
| `NEWAPI_TOKEN` | `newapi.token` | Overrides only when non-empty |
| `NEWAPI_WRITEMODE` | `newapi.writemode` | `read` / `ops` / `admin` |
| `NEWAPI_TIMEOUT` | `newapi.timeout_seconds` | Invalid values are ignored, keeping the original value |
| `NEWAPI_REPORT_DB_DSN` | `report.db_dsn` | Reporting read replica DSN, highest precedence |

## Full Example

```toml
[newapi]
base_url = "https://newapi.ashou.site"
token_file = "/home/radxa/.dsh/newapi.pat"   # 0600, first line read as the PAT
writemode = "ops"                             # read(11) / ops(17) / admin(23)
timeout_seconds = 10

[report]                                      # optional: reporting read replica for the Jiyuan report
db_dsn = ""                                   # plaintext not recommended
db_dsn_file = "/home/radxa/.dsh/report-db.dsn"
```

A placeholder template is available at the repo root: [`newapi-mcp.example.toml`](../newapi-mcp.example.toml).

## Security Notes

- Never write the PAT / DSN into the config file itself; reference them indirectly via `token_file` / `db_dsn_file` with file permission `0600`
- No MCP response ever exposes a full key (masked, keeping 4 chars at each end); a full key is only passed in explicitly by the caller in create/update requests
- For production deployment examples (systemd/wrapper) see [`examples/`](../examples/)
