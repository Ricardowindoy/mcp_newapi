# Client Integration

> [中文](client-integration.md) | English

newapi-mcp is an **MCP server over stdio transport**; any client that supports stdio MCP can mount it. Process protocol: **stdout carries JSON-RPC only, all logs go to stderr**.

## Claude Desktop / Cline / Generic mcpServers Clients

```json
{
  "mcpServers": {
    "newapi": {
      "command": "/absolute/path/to/bin/newapi-mcp",
      "env": {
        "NEWAPI_BASE_URL": "https://your-newapi.example",
        "NEWAPI_TOKEN": "<dashboard PAT>",
        "NEWAPI_WRITEMODE": "ops"
      }
    }
  }
}
```

A complete ready-to-copy file: [`examples/claude-desktop-config.json`](../../examples/claude-desktop-config.json).

## DSH (DeepSeek Harness)

Register via `cordis.patch.yml` and launch through a wrapper script (configuration in TOML, PAT referenced indirectly via `token_file`):

```yaml
mcp-newapi:
  name: '@deepseek-ai/dsh-mcp-client'
  config:
    serverName: newapi
    transport: stdio
    command: /home/<user>/.dsh/mcp-newapi-wrapper.sh   # exec newapi-mcp --config ~/.dsh/newapi-mcp.toml
    args: []
```

The wrapper and the snippet can be copied directly: [`examples/newapi-mcp-wrapper.sh`](../../examples/newapi-mcp-wrapper.sh), [`examples/cordis-patch.yml`](../../examples/cordis-patch.yml). After changing the wrapper / binary / cordis.patch.yml, a **DSH restart** is required for the change to take effect.

## Choosing a Tier (writemode)

| Tier | Tools | Fits | Risk surface |
|---|---|---|---|
| `read` (default) | 11 | Inspection, status checks, accounting reports | Read-only; channel reads need an admin PAT |
| `ops` | 17 | Daily operations: tests / enable-disable / balance / token management | Has write operations but in a limited scope |
| `admin` | 23 | Channel CRUD, system settings, batch tagging, autoban | High-impact tools present (confirm gate as the backstop) |

- **Tiering works by not registering**: under lower tiers, higher-tier tools simply do not exist, so agents cannot probe for them.
- `ops` is recommended for daily use; switch to `admin` temporarily when batch channel remediation is needed.
- To have "daily read-only + occasional management" at the same time, register two instances (different `serverName`, different writemode) instead of keeping admin around permanently.

## Troubleshooting Quick Reference

| Symptom | What to do |
|---|---|
| Empty tool list / server exits on startup | `base_url` not configured or `writemode` misspelled (only read/ops/admin are accepted); stderr will show the error |
| Channel tools return 403 | A PAT from an admin account is required |
| Slow responses / timeouts | Raise `timeout_seconds`; check the network from the client to the gateway (mind `no_proxy` behind a proxy) |
| Config changes not taking effect | stdio processes start with the client — restart the client (DSH needs a restart) |
