# 客户端接入 / Client Integration

> [English](client-integration.en.md) | 中文

newapi-mcp 是 **stdio 传输的 MCP server**，任何支持 stdio MCP 的客户端都能挂。进程协议：**stdout 只跑 JSON-RPC，日志一律 stderr**。

## Claude Desktop / Cline / 通用 mcpServers 客户端

```json
{
  "mcpServers": {
    "newapi": {
      "command": "/absolute/path/to/bin/newapi-mcp",
      "env": {
        "NEWAPI_BASE_URL": "https://your-newapi.example",
        "NEWAPI_TOKEN": "<面板 PAT>",
        "NEWAPI_WRITEMODE": "ops"
      }
    }
  }
}
```

完整可抄文件：[`examples/claude-desktop-config.json`](../examples/claude-desktop-config.json)。

## DSH（DeepSeek Harness）

经 `cordis.patch.yml` 注册 + wrapper 脚本启动（配置走 TOML，PAT 用 `token_file` 间接引用）：

```yaml
mcp-newapi:
  name: '@deepseek-ai/dsh-mcp-client'
  config:
    serverName: newapi
    transport: stdio
    command: /home/<user>/.dsh/mcp-newapi-wrapper.sh   # exec newapi-mcp --config ~/.dsh/newapi-mcp.toml
    args: []
```

wrapper 与片段可直接抄：[`examples/newapi-mcp-wrapper.sh`](../examples/newapi-mcp-wrapper.sh)、[`examples/cordis-patch.yml`](../examples/cordis-patch.yml)。改 wrapper / 二进制 / cordis.patch.yml 后需**重启 DSH** 生效。

## 档位（writemode）怎么选

| 档位 | 工具数 | 适合 | 风险面 |
|---|---|---|---|
| `read`（默认） | 11 | 巡检、查状态、记账报表 | 只读；渠道读需管理员 PAT |
| `ops` | 17 | 日常运维：测试/启停/余额/令牌管理 | 有写操作但范围有限 |
| `admin` | 23 | 渠道 CRUD、系统设置、标签批量、autoban | 高危工具在场（confirm 门禁兜底） |

- **能力分档靠不注册**：低档下高档工具根本不存在，Agent 无法试探。
- 日常建议 `ops`；需要批量整改渠道时临时切 `admin`。
- 想同时要「日常只读 + 偶尔管理」，可注册两个实例（不同 `serverName`、不同 writemode），避免常驻 admin。

## 排错速查

| 现象 | 处理 |
|---|---|
| 工具列表为空 / 启动退出 | `base_url` 未配置或 `writemode` 拼错（只认 read/ops/admin），stderr 会有报错 |
| 渠道工具 403 | 需要管理员账号的 PAT |
| 响应慢/超时 | 调大 `timeout_seconds`；确认客户端到网关的网络（代理环境注意 `no_proxy`） |
| 改了配置不生效 | stdio 进程随客户端启动——重启客户端（DSH 需重启） |
