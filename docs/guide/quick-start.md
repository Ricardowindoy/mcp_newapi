# 快速上手 / Quick Start

> [English](quick-start.en.md) | 中文

5 分钟把 newapi-mcp 跑起来，挂到任意 MCP 客户端上。

## 前置要求

- **Go 1.26+**（仅构建时需要；产物是单二进制）
- 一个可访问的 [new-api](https://github.com/QuantumNous/new-api) 实例（自建或任意 OpenAI 风格部署）
- 面板 **PAT**：new-api 个人设置 → 系统访问令牌。普通 PAT 可用 read/ops 档；**渠道读写需要管理员账号的 PAT**

## 1. 构建

```bash
git clone https://github.com/Ricardowindoy/mcp_newapi.git
cd mcp_newapi
go build -o bin/newapi-mcp ./cmd/newapi-mcp
```

## 2. 最小配置（环境变量方式）

```bash
export NEWAPI_BASE_URL=https://your-newapi.example   # 网关地址，无尾斜杠
export NEWAPI_TOKEN=<你的面板 PAT>
export NEWAPI_WRITEMODE=read                          # read(默认) / ops / admin
```

不想把 PAT 放环境变量？见 [配置参考](configuration.md) 的 `token_file` 间接引用方式（推荐）。

## 3. 验证

二进制是 **stdio 传输的 MCP server**，直接运行会等待 JSON-RPC 输入——单独跑没有输出是正常的。挂到一个 MCP 客户端里验证：

- **Claude Desktop / Cline 等**：把二进制路径填进 `mcpServers` 配置，见 [客户端接入](client-integration.md)。
- **DSH**：见 [客户端接入](client-integration.md) 的 cordis 挂载节，或直接抄 [examples/cordis-patch.yml](../../examples/cordis-patch.yml)。

挂载成功后，客户端工具列表里应出现 `newapi_status` 等 11 个 read 档工具（ops/admin 档随 `NEWAPI_WRITEMODE` 增加）。先调 `newapi_status` 确认网关连通与 relay 活性。

## 4. 下一步

- **[工具参考](tools.md)** —— 25 个工具的参数、示例与注意事项
- **[配置参考](configuration.md)** —— TOML 文件、环境变量全表、报表从库 `[report]` 段
- **[客户端接入](client-integration.md)** —— Claude Desktop / DSH / 通用 stdio 接入与档位选型
- **[上游兼容与已知坑](../design/upstream-compat.md)** —— 对接自己 new-api 部署前建议先读

## 常见问题

- **调渠道工具报 403/无权限？** 渠道列表/详情/管理需要**管理员**账号的 PAT。
- **`newapi_jiyuan_report` 报「报表功能未配置」？** 这是可选能力，需配置报表从库 `[report]` 段（见[配置参考](configuration.md)）；未配置时工具仍注册但调用会明确报错，属预期行为。
- **工具列表里没有写工具？** `NEWAPI_WRITEMODE` 默认 `read`，写工具**不注册**（而非注册后拒绝），调高档位即可。
