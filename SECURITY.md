# 安全策略 / Security Policy

> 中文 | [English](SECURITY.en.md)

## 密钥与数据处理设计

newapi-mcp 是**网关管理面**的 MCP 封装，安全设计围绕「密钥不落盘、不回显、不放大」：

1. **PAT 间接引用**：PAT 经配置 `token_file`（建议 `0600`）读首行注入，不写进配置文件本体、不进 git；报表库 DSN 同理走 `db_dsn_file`。
2. **响应掩码**：任何 MCP 响应中的渠道 key / sk- 令牌一律掩码（保留头尾各 4 位）；完整 key 只在创建/更新请求中由调用方显式传入，**任何响应不回显**（创建令牌返回 id + 掩码，提示去面板复制）。
3. **能力分档靠不注册**：`read`（默认）/ `ops` / `admin` 三档，低档下高档工具根本不注册，Agent 无法试探。
4. **confirm 门禁**：删除（delete_token / delete_channel）与高危变更（update_option / autoban_codes 变更 / tag_channels）必须显式 `confirm=true`。
5. **最小封装面**：`redemption/`（兑换码）不封装；`option/`（系统设置）仅受控读写（上游已过滤敏感键 + confirm）；PAT 权限天然继承——MCP 能做的最多等于该 PAT 账号在面板里能做的，不额外放大权限。
6. **不打请求体日志**：MCP 进程自身不打含 key 的请求/响应日志；stdio 传输下 stdout 只跑 JSON-RPC。

## 报告漏洞

- 请通过 GitHub **Private Security Advisory**（仓库 Security 标签页 → Report a vulnerability）私下报告，**不要**在公开 Issue 里贴可复现的凭据/配置细节。
- 报告请包含：影响面（哪个工具/档位）、复现步骤、预期 vs 实际。
- 修复后会在 [CHANGELOG.md](CHANGELOG.md) 标注。

## 支持版本

仅跟进 `main` 分支最新代码；历史里程碑（M1–M8）见 [CHANGELOG.md](CHANGELOG.md)。
