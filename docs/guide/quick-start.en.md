# Quick Start

> [中文](quick-start.md) | English

Get newapi-mcp up and running in 5 minutes and mount it onto any MCP client.

## Prerequisites

- **Go 1.26+** (needed at build time only; the artifact is a single binary)
- A reachable [new-api](https://github.com/QuantumNous/new-api) instance (self-hosted or any OpenAI-style deployment)
- Dashboard **PAT**: new-api personal settings → system access token. A regular PAT works for the read/ops tiers; **channel read/write requires a PAT from an admin account**

## 1. Build

```bash
git clone https://github.com/Ricardowindoy/mcp_newapi.git
cd mcp_newapi
go build -o bin/newapi-mcp ./cmd/newapi-mcp
```

## 2. Minimal Configuration (Environment Variables)

```bash
export NEWAPI_BASE_URL=https://your-newapi.example   # gateway address, no trailing slash
export NEWAPI_TOKEN=<your dashboard PAT>
export NEWAPI_WRITEMODE=read                          # read (default) / ops / admin
```

Prefer not to put the PAT in an environment variable? See the `token_file` indirect reference in the [Configuration Reference](configuration.en.md) (recommended).

## 3. Verify

The binary is an **MCP server over stdio transport**; run directly, it waits for JSON-RPC input — no output when run standalone is normal. Verify by mounting it into an MCP client:

- **Claude Desktop / Cline, etc.**: put the binary path into the `mcpServers` config, see [Client Integration](client-integration.en.md).
- **DSH**: see the cordis mounting section of [Client Integration](client-integration.en.md), or copy [examples/cordis-patch.yml](../../examples/cordis-patch.yml) directly.

Once mounted, the client's tool list should show 11 read-tier tools including `newapi_status` (ops/admin tiers add more as `NEWAPI_WRITEMODE` rises). Call `newapi_status` first to confirm gateway connectivity and relay liveness.

## 4. Next Steps

- **[Tools Reference](tools.en.md)** — parameters, examples, and caveats for all 23 tools
- **[Configuration Reference](configuration.en.md)** — TOML file, full environment-variable table, and the reporting read replica `[report]` section
- **[Client Integration](client-integration.en.md)** — Claude Desktop / DSH / generic stdio setup and tier selection
- **[Upstream Compatibility & Known Pitfalls](../design/upstream-compat.en.md)** — recommended reading before wiring up your own new-api deployment

## FAQ

- **Channel tools return 403 / no permission?** Channel list/detail/management requires a PAT from an **admin** account.
- **`newapi_jiyuan_report` reports "reporting not configured"?** This is an optional capability; configure the reporting read replica `[report]` section (see the [Configuration Reference](configuration.en.md)). When unconfigured the tool stays registered but calls fail with an explicit error — expected behavior.
- **No write tools in the tool list?** `NEWAPI_WRITEMODE` defaults to `read`, and write tools are **not registered** (rather than registered-then-rejected). Raise the tier to get them.
