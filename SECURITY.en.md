# Security Policy

> [中文](SECURITY.md) | English

## Secret & Data Handling Design

newapi-mcp is an MCP wrapper over the **gateway management plane**; the security design revolves around "secrets never written to disk, never echoed back, never amplified":

1. **Indirect PAT reference**: the PAT is injected by reading the first line of the file configured as `token_file` (permission `0600` recommended); it is never written into the config file itself nor committed to git. The reporting DB DSN follows the same pattern via `db_dsn_file`.
2. **Response masking**: channel keys / sk- tokens in any MCP response are always masked (keeping 4 chars at each end); a full key is only passed in explicitly by the caller in create/update requests and is **never echoed back in any response** (token creation returns the id + masked key, prompting the user to copy the full key from the dashboard).
3. **Tiering works by not registering**: three tiers — `read` (default) / `ops` / `admin`; under lower tiers, higher-tier tools are simply not registered, so agents cannot probe for them.
4. **confirm gate**: deletions (delete_token / delete_channel) and high-impact changes (update_option / autoban_codes changes / tag_channels) require an explicit `confirm=true`.
5. **Minimal wrapping surface**: `redemption/` (redemption codes) is not wrapped; `option/` (system settings) supports only controlled read/write (upstream already filters sensitive keys + confirm); PAT permissions are inherited as-is — what the MCP can do is at most what the PAT's account can do in the dashboard, amplifying nothing.
6. **No request-body logging**: the MCP process itself never logs key-bearing requests/responses; over stdio transport, stdout carries JSON-RPC only.

## Reporting a Vulnerability

- Please report privately via a GitHub **Private Security Advisory** (repo Security tab → Report a vulnerability); do **not** post reproducible credentials/config details in public Issues.
- Please include: impact scope (which tool/tier), reproduction steps, expected vs actual behavior.
- Fixes will be noted in [CHANGELOG.md](CHANGELOG.md).

## Supported Versions

Only the latest code on the `main` branch is tracked; historical milestones (M1–M8) are in [CHANGELOG.md](CHANGELOG.md).
