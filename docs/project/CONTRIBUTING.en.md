# Contributing

> [中文](CONTRIBUTING.md) | English

## Development Environment

- Go **1.26+** (single binary, no runtime dependencies)
- Dependencies: `github.com/mark3labs/mcp-go` (MCP SDK), `github.com/BurntSushi/toml`, `github.com/go-sql-driver/mysql`

## Build & Test (hard requirements before pushing)

```bash
go vet ./... && go build ./...
go test ./...        # pure-logic unit tests (httptest mocks the upstream), never touches a real gateway
```

- Test convention: **pure logic is tested with `httptest`** (see `internal/newapi/client_test.go`, `channels/ops_test.go`); no real gateway is involved.
- Bug fixes must add a matching test case; changing code without tests is not allowed.
- Commit messages follow the milestone style (`M1:` / `M2:` …), stating the motivation and verification results.

## Architecture at a Glance

Four layers with one-way dependencies; reading the corresponding module docs before changing code is recommended:

```
cmd/newapi-mcp → internal/mcp (registry table + server assembly)
                 └─ handler/ (thin tool shells: parse params → call domain functions → output)
                      ├─ internal/newapi/{status,models,channels,tokens,logs,options} (domain sub-packages)
                      │    └─ internal/newapi root package (client transport + routes.go single coupling point)
                      └─ internal/reporter (report leaf package, connects to the replica directly)
internal/config (config loading, cross-cutting module)
```

- **Layering rules and prohibitions**: [.docs/_mustread/模块职责边界设计规范.md](../../.docs/_mustread/模块职责边界设计规范.md)
- **Build/test/security red lines**: [.docs/_mustread/开发规范.md](../../.docs/_mustread/开发规范.md)
- Per-module "spec / implementation details": [.docs/README.md](../../.docs/README.md) (module index)
- Design background and decisions: [DESIGN.md](../design/DESIGN.md)

## Flow for Adding a New Tool

1. Create/extend a domain sub-package under `internal/newapi/<domain>/` (raw DTO + Summary + masking + domain functions + upstream contract comment at the top); for external-data-source tools (e.g. reporting), create an `internal/<domain>/` leaf package
2. Add endpoint constants to `routes.go` (skip if there is no HTTP endpoint)
3. Add a table entry in `internal/mcp/registry.go` (Name / Tier / parameter declarations / Handler factory)
4. Implement the thin-shell handler in `internal/mcp/handler/<tier>.go`
5. Sync the `.docs/<module>/` docs in the same round
6. Cover the new contract with httptest unit tests (envelope unwrapping / masking / error semantics)

## Security Red Lines (summary; full text in [SECURITY.en.md](SECURITY.en.md))

- PAT / channel keys / full sk- **never committed to the repo, never appear in any response** (masked, keeping 4 chars at each end)
- Deletion and high-impact change tools require `confirm=true`; `redemption/` (redemption codes) is not wrapped
- Over stdio transport, stdout carries JSON-RPC only; all logs go to stderr

## Bilingual Docs Convention

In `docs/`, Chinese is the primary doc (`<name>.md`) and the English translation lives alongside as `<name>.en.md` — after changing a Chinese page, sync its English translation.
