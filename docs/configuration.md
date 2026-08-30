# 配置参考 / Configuration Reference

> [English](configuration.en.md) | 中文

配置优先级：**内置默认值 < TOML 配置文件 < 环境变量**。两种方式任选其一或混用（env 只覆盖非空项）。

```bash
newapi-mcp --config /path/to/newapi-mcp.toml   # 不传 --config 则只用默认值+环境变量
```

## `[newapi]` 段（网关连接，必配）

| 字段 | 类型 | 必填 | 默认 | 说明 |
|---|---|---|---|---|
| `base_url` | string | ✅ | — | 网关地址，**无尾斜杠**（env 传入会自动去尾斜杠） |
| `token` | string | 二选一 | — | 面板 PAT。建议留空改用 `token_file`，避免密钥落进配置文件本体 |
| `token_file` | string | 二选一 | — | PAT 文件路径：**读首行**作为 token。文件权限建议 `0600` |
| `writemode` | string | — | `read` | 工具档位：`read`（11 个）/ `ops`（+6）/ `admin`（+6）。低档**不注册**高档工具 |
| `timeout_seconds` | int | — | `10` | 上游 HTTP 超时秒数；≤0 回落为 10 |

`token` 与 `token_file` 同时配置时 `token` 优先；两者都没有时启动报错（base_url 缺失同理）。

## `[report]` 段（报表从库，可选）

`newapi_jiyuan_report` 工具的数据源——**直连 MySQL 从库**聚合 `logs` 与 `model_price_snapshots` 表，不走 new-api HTTP API。未配置时工具保持注册、调用时明确报「报表功能未配置」（不静默降级）。

| 字段 | 类型 | 说明 |
|---|---|---|
| `db_dsn` | string | 直连 DSN：`user:pass@tcp(host:port)/db?charset=utf8mb4`。不推荐明文落盘 |
| `db_dsn_file` | string | DSN 文件路径：读首行，权限 `0600`（推荐） |

DSN 解析优先级：**`NEWAPI_REPORT_DB_DSN` env > `db_dsn` > `db_dsn_file` 首行**。文件读取失败/首行为空时按「未配置」处理（不阻断 MCP 启动，报表是可选能力）。

## 环境变量覆盖表

| 环境变量 | 覆盖字段 | 说明 |
|---|---|---|
| `NEWAPI_BASE_URL` | `newapi.base_url` | 非空才覆盖，自动去尾斜杠 |
| `NEWAPI_TOKEN` | `newapi.token` | 非空才覆盖 |
| `NEWAPI_WRITEMODE` | `newapi.writemode` | `read` / `ops` / `admin` |
| `NEWAPI_TIMEOUT` | `newapi.timeout_seconds` | 非法值忽略，保持原值 |
| `NEWAPI_REPORT_DB_DSN` | `report.db_dsn` | 报表从库 DSN，最高优先级 |

## 完整示例

```toml
[newapi]
base_url = "https://newapi.ashou.site"
token_file = "/home/radxa/.dsh/newapi.pat"   # 0600，读首行作 PAT
writemode = "ops"                             # read(11) / ops(17) / admin(23)
timeout_seconds = 10

[report]                                      # 可选：基元报表从库
db_dsn = ""                                   # 不推荐明文
db_dsn_file = "/home/radxa/.dsh/report-db.dsn"
```

占位模板见仓库根目录 [`newapi-mcp.example.toml`](../newapi-mcp.example.toml)。

## 安全要点

- PAT / DSN 永不写进配置文件本体，用 `token_file` / `db_dsn_file` 间接引用，文件 `0600`
- 任何 MCP 响应不透出完整 key（掩码头尾各 4 位）；完整 key 只在创建/更新请求中由调用方显式传入
- 生产部署示例（systemd/wrapper）见 [`examples/`](../examples/)
