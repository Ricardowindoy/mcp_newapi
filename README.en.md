# newapi-mcp

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8.svg)](https://go.dev)
[![MCP](https://img.shields.io/badge/transport-stdio-8A2BE2.svg)](https://modelcontextprotocol.io)

An MCP (Model Context Protocol) server for operating a [new-api](https://github.com/QuantumNous/new-api) LLM gateway: lets agents inspect gateway status and manage channels & tokens directly.

> 中文文档：[README.md](README.md)（English docs below reference `*.en.md` companions in `docs/`）

## Documentation

| Doc | Contents |
|---|---|
| **[docs/guide/quick-start.en.md](docs/guide/quick-start.en.md)** | Build, configure and mount in 5 minutes |
| **[docs/guide/tools.en.md](docs/guide/tools.en.md)** | All 23 tools: parameters, examples, caveats |
| **[docs/guide/configuration.en.md](docs/guide/configuration.en.md)** | Full TOML/env reference incl. `[report]` replica DB |
| **[docs/guide/client-integration.en.md](docs/guide/client-integration.en.md)** | Claude Desktop / DSH / generic stdio clients |
| **[docs/design/upstream-compat.en.md](docs/design/upstream-compat.en.md)** | new-api version adaptation & known pitfalls |
| [CHANGELOG.md](docs/project/CHANGELOG.md) / [CONTRIBUTING.en.md](docs/project/CONTRIBUTING.en.md) / [SECURITY.en.md](docs/project/SECURITY.en.md) | Changelog / Contributing / Security |
| [DESIGN.md](docs/design/DESIGN.md) | Design details (architecture, upstream contracts, milestones) |
| [.docs/](docs/README.md) | Module-level maintenance docs (read before changing code) |

## Features

- **23 tools, 3 permission tiers** (`NEWAPI_WRITEMODE`: `read` 11 / `ops` +6 / `admin` +6) — lower tiers simply **don't register** higher-tier tools
- **Single Go binary**, mcp-go + stdio transport, zero runtime dependencies
- **Key safety**: full keys never appear in any response (masked head/tail 4 chars); destructive/high-impact tools require explicit `confirm=true`
- **Optional reporting**: `jiyuan_report` aggregates channel spend directly from a MySQL read replica (`[report]` config; absent → tool errors clearly, everything else works)
- **Upstream decoupling**: all endpoint paths centralized in `internal/newapi/routes.go`; upstream updates touch one file

## Quick start

```bash
go build -o bin/newapi-mcp ./cmd/newapi-mcp

export NEWAPI_BASE_URL=https://your-newapi.example   # gateway URL, no trailing slash
export NEWAPI_TOKEN=<panel PAT>                       # new-api panel → personal settings → system access token
export NEWAPI_WRITEMODE=read                          # read (default) / ops / admin
./bin/newapi-mcp                                      # stdio JSON-RPC; mount it into any MCP client
```

Full walkthrough (TOML config, `token_file` secret indirection) in **[docs/guide/quick-start.en.md](docs/guide/quick-start.en.md)**.

## Tools at a glance

| Tier | Tools | Purpose |
|---|---|---|
| read | `newapi_status` | Site status + relay liveness probe |
| read | `newapi_list_models` | All models grouped |
| read | `newapi_list_channels` / `newapi_get_channel` | Channel list / detail (admin PAT) |
| read | `newapi_list_tokens` | Your sk- tokens |
| read | `newapi_logs` | Consumption / error log search |
| read | `newapi_usage_summary` | N-day per-model usage ($ converted) |
| read | `newapi_pricing` | Model ratios (may be disabled per instance) |
| read | `newapi_list_options` | System options (sensitive keys filtered upstream) |
| read | `newapi_success_rate` | Request success rate from log counts |
| read | `newapi_jiyuan_report` | Channel spend report (replica DB aggregation, needs `[report]`) |
| ops | `newapi_test_channel` / `newapi_test_all_channels` | Single / all-channel tests |
| ops | `newapi_update_channel_balance` | Refresh channel balance |
| ops | `newapi_set_channel_status` | Enable / disable channel |
| ops | `newapi_create_token` / `newapi_delete_token` | Token lifecycle |
| admin | `newapi_create_channel` / `newapi_update_channel` / `newapi_delete_channel` | Channel CRUD (update is PATCH, incl. tag/auto_ban) |
| admin | `newapi_update_option` | Modify a system option (confirm-gated) |
| admin | `newapi_autoban_codes` | Auto-ban status codes CRUD (range algebra) |
| admin | `newapi_tag_channels` | Batch edit / enable / disable by tag |

Per-tool parameters and response examples: **[docs/guide/tools.en.md](docs/guide/tools.en.md)**.

## Architecture

```
internal/config/   config loading (TOML: defaults < file < env, token_file indirection)
internal/mcp/      tool layer: registry.go (single table, 23 tools) + handler/ subpackage
internal/newapi/   API layer: client.go transport + routes.go endpoint coupling point
  └─ domain pkgs: status/ models/ channels/ tokens/ logs/ options/
internal/reporter/ reporting leaf package (direct replica aggregation, DSN via config)
```

## Configuration

| Env var | Required | Meaning |
|---|---|---|
| `NEWAPI_BASE_URL` | ✅ | Gateway URL (no trailing slash) |
| `NEWAPI_TOKEN` | ✅* | Panel PAT (admin PAT for channel access); or TOML `token_file` |
| `NEWAPI_WRITEMODE` | — | `read` (default) / `ops` / `admin` |
| `NEWAPI_TIMEOUT` | — | HTTP timeout seconds (default 10) |
| `NEWAPI_REPORT_DB_DSN` | — | Replica DB DSN for `jiyuan_report` |

\* Either `NEWAPI_TOKEN` or TOML `token`/`token_file`. Full field reference: **[docs/guide/configuration.en.md](docs/guide/configuration.en.md)**; template: [`newapi-mcp.example.toml`](newapi-mcp.example.toml).

## Development

```bash
go vet ./... && go build ./...
go test ./...        # pure-logic unit tests (httptest), no real gateway
```

See [CONTRIBUTING.en.md](docs/project/CONTRIBUTING.en.md) for the six-step "add a tool" workflow. Upstream reference snapshots live in `.upstream/` (gitignored).

## License

[GPL-3.0](LICENSE)
