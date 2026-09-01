# Changelog / 更新日志

格式：里程碑（M1–M8），全部于 **2026-08-30** 完成。语义化版本随首个人口发布引入。

All milestones shipped on **2026-08-30** as a single-day build-out; each entry lists 中文 + English.

## Unreleased — 2026-09-01（渠道 model_mapping 回传）

- 渠道 `Summary` DTO 新增 `model_mapping`：`newapi_list_channels` / `newapi_get_channel` 随读返回（上游 JSON 字符串原样，`null`→空串，`omitempty` 无映射不出字段）；`newapi_create_channel` / `newapi_update_channel` 成功回显含改后映射（空串=已清空）
- EN: Channel `Summary` DTO now carries `model_mapping` — returned by `newapi_list_channels` / `newapi_get_channel` (upstream JSON string verbatim, `null`→empty, omitted via `omitempty` when absent); `newapi_create_channel` / `newapi_update_channel` echo the post-write mapping (empty string = cleared).

## M8 — 2026-08-30（17 → 23 工具）

- 新增 options 域：系统设置读写（上游敏感键过滤）+ autoban 状态码区间代数（add/remove/set，规范形输出）
- 新增渠道标签批量操作（edit/enable/disable）、`success_rate` 成功率、`jiyuan_report` 基元消费报表
- 新增 `internal/reporter` 报表叶子包：直连 MySQL 从库聚合（DSN 经 `[report]` 配置间接注入）
- EN: Options domain (sanitized system settings read/write + autoban status-code range algebra), tag batch ops, success-rate tool, and a direct-to-replica reporting package (`jiyuan_report`).

## M7 — 2026-08-30（架构演进）

- handler 独立子包（read/ops/admin/report 薄壳）+ newapi 按域分包（status/models/channels/tokens/logs），根包只留传输/路由/共享件
- EN: Split handlers into a subpackage and the API layer into per-domain packages; root package keeps transport/routes/shared only.

## M6 — 2026-08-30（配置 + 注册表）

- `internal/config`：TOML 配置模块（默认 < 文件 < env，`token_file` 间接引用）
- `registry.go` 表驱动注册（工具唯一汇总表）
- EN: Standalone TOML config module (defaults < file < env, `token_file` indirection) and a table-driven tool registry.

## M5 — 2026-08-30（打磨）

- README + 单元测试（client 信封/掩码/状态语义 + config 模块）
- EN: README and unit tests for envelope unwrapping, key masking, status semantics, and config.

## M4 — 2026-08-30（admin 档起步）

- 渠道 CRUD 3 工具（创建带 key 只进不出；更新 PATCH 语义；删除 confirm），全闭环实测
- EN: Channel CRUD (key only-in, PATCH-semantics update, confirm-gated delete), closed-loop verified.

## M3 — 2026-08-30（ops 档）

- 渠道测试/全量测试/余额刷新/启停 + 令牌管理（创建后按名回查 id + 掩码），分层重构
- EN: Channel test/test-all/balance/status tools + token lifecycle (lookup-by-name), with layering refactor.

## M2 — 2026-08-30（read 档）

- 8 个只读工具 + 掩码层，全部对实装端点核对（发现 PUT status 被拒、pricing 可能被禁、/api/data/ 聚合源）
- EN: 8 read-only tools + masking layer, verified against live endpoints (uncovered: PUT status refused, pricing may be disabled, /api/data/ aggregation).

## M1 — 2026-08-30（骨架）

- Go 骨架：client（鉴权/解包/超时）+ mcp-go stdio server + `newapi_status` 打通
- EN: Go skeleton — authenticated client, envelope unwrapping, timeout; mcp-go stdio server; `newapi_status` wired end-to-end.
