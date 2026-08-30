# newapi-mcp

操作 [new-api](https://github.com/QuantumNous/new-api) 网关的 MCP (Model Context Protocol) 服务器：让 Agent 直接读取网关运行状态、管理渠道与令牌。

## 特性

- **23 个工具，三档权限**（`NEWAPI_WRITEMODE`：`read` 11 个 / `ops` +6 / `admin` +6），低档不注册写工具
- **Go 单二进制**，mcp-go + stdio 传输，无运行时依赖
- **密钥安全**：任何响应不透出完整 key（掩码头尾 4 位）；删除操作强制 `confirm=true`
- **上游解耦**：端点路径集中在 `internal/newapi/routes.go`，域模块一域一文件，上游更新只动对应文件（见 DESIGN.md §5 维护映射表）

## 快速开始

```bash
# 构建（Go 1.26+）
go build -o bin/newapi-mcp ./cmd/newapi-mcp

# 手动运行
export NEWAPI_BASE_URL=https://your-newapi.example
export NEWAPI_TOKEN=<面板 PAT>          # 个人设置 → 系统访问令牌
export NEWAPI_WRITEMODE=ops            # read(默认)/ops/admin
./bin/newapi-mcp                       # stdio JSON-RPC
```

## 工具一览

| 档位 | 工具 | 说明 |
|---|---|---|
| read | `newapi_status` | 站点状态 + relay 活性探测 |
| read | `newapi_list_models` | 全站模型（按分组） |
| read | `newapi_list_channels` / `newapi_get_channel` | 渠道列表/详情（管理员 PAT） |
| read | `newapi_list_tokens` | 当前用户令牌列表 |
| read | `newapi_logs` | 消费/错误日志检索 |
| read | `newapi_usage_summary` | 近 N 天按模型聚合用量（$ 换算） |
| read | `newapi_pricing` | 模型倍率（实例可能禁用） |
| read | `newapi_list_options` | 系统设置键值对（上游已脱敏） |
| read | `newapi_success_rate` | 请求成功率（日志 type=2/5 计数，支持渠道/模型/令牌过滤） |
| read | `newapi_jiyuan_report` | 基元渠道消费报表（从库 logs×价格快照聚合，需 [report] 配置） |
| ops | `newapi_test_channel` / `newapi_test_all_channels` | 单渠道/全量测试 |
| ops | `newapi_update_channel_balance` | 刷新渠道余额 |
| ops | `newapi_set_channel_status` | 启用/禁用渠道 |
| ops | `newapi_create_token` / `newapi_delete_token` | 令牌生命周期 |
| admin | `newapi_create_channel` / `newapi_update_channel` / `newapi_delete_channel` | 渠道 CRUD（update_channel 含 tag/auto_ban PATCH） |
| admin | `newapi_update_option` | 系统设置修改（confirm 门禁） |
| admin | `newapi_autoban_codes` | autoban 状态码增删查改（disable/retry，区间代数） |
| admin | `newapi_tag_channels` | 按标签批量编辑/启停渠道 |

## 架构（分层 + 域子包，单向依赖）

```
internal/config/   配置模块（TOML：默认 < 文件 < 环境变量，token_file 间接引用）
internal/mcp/      工具层：registry.go 表驱动汇总表（23 工具唯一索引）+ handler/ 子包
internal/newapi/   API 层：client.go 传输 + routes.go 端点耦合点
  └─ 域子包：status/ models/ channels/(读+运维+管理+标签) tokens/ logs/ options/(系统设置+状态码代数)
internal/reporter/ 报表域：直连从库聚合消费报表（叶子包，DSN 经 config 注入）
```

设计细节与上游契约注释见 [DESIGN.md](DESIGN.md)。

## 配置

| 环境变量 | 必填 | 说明 |
|---|---|---|
| `NEWAPI_BASE_URL` | ✅ | 网关地址（无尾斜杠） |
| `NEWAPI_TOKEN` | ✅ | 面板 PAT（管理员 PAT 才有渠道读权限） |
| `NEWAPI_WRITEMODE` | — | `read`（默认）/ `ops` / `admin` |
| `NEWAPI_TIMEOUT` | — | HTTP 超时秒数，默认 10 |

## 开发

```bash
go vet ./... && go build ./... 
go test ./...        # 纯逻辑单测（不碰网络）
```

上游契约验证用的参考源码快照在 `.upstream/`（gitignore）。
