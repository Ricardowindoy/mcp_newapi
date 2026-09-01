# 工具参考 / Tools Reference

> [English](tools.en.md) | 中文

全部 25 个工具的参数与语义。档位由 `writemode` 决定：**read 13 个（默认注册）/ ops +6 / admin +6**，低档不注册高档工具。

## 通用约定

- **调用格式**：标准 MCP `tools/call`，返回统一为 JSON 文本（`content[0].text`）。
- **错误格式**：`[reachable=true status=403] message` 工具错误文本——`reachable=true` 表示网关应答了（业务错），`reachable=false` 表示网关不可达（网络错），Agent 可据此区分。
- **密钥掩码**：任何响应中的渠道 key / sk- 令牌一律掩码（保留头尾各 4 位）。完整 key 只在创建/更新请求中由调用方显式传入，**任何响应不回显**。
- **confirm 门禁**：删除与高危变更类工具（delete_token / delete_channel / update_option / autoban_codes 变更 / tag_channels）必须显式传 `confirm: true`，否则直接拒绝。
- **业务失败 ≠ 工具错误**：渠道测试不通等是 `success:false` 的正常 JSON 结果，不是 error。
- **quota 换算**：`500000 quota = $1`。

---

## read 档（13 个，默认注册）

### `newapi_status`
站点公开状态：版本、启动时间、注册开关；附带一次 `/v1/models` 探测（5s 独立超时），**返回 401/403 也算 relay 活着**（能应答即可达）。无需 PAT。

### `newapi_list_models`
全站可用模型，`data` 为 `分组 → 模型数组` 透传并组内排序；附 `group_count` / `model_count` 汇总。无需 PAT。

### `newapi_list_channels`
分页渠道列表（**需管理员 PAT**）。

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `page` | number | — | 页码，从 1 起（默认 1） |
| `page_size` | number | — | 每页条数（默认 20） |
| `status` | number | — | 状态过滤：0=全部（默认）1=启用 2=手动禁用 3=自动禁用 |

```json
{ "page": 1, "page_size": 20, "total": 42, "items": [
  { "id": 31, "name": "基元2_x", "type": 1, "status": 1, "status_reason": "",
    "balance": 12.5, "base_url": "https://…", "models": "deepseek-v4-pro,…",
    "group": "default", "model_mapping": "{\"deepseek-v4-flash\":\"deepseek-v4-flash-0731\"}",
    "priority": 0, "weight": 0, "test_model": "",
    "response_time": 2340, "used_quota": 812345.6, "key": "sk-a***7890" } ] }
```

渠道项含 `model_mapping`（模型重定向 JSON 字符串原样，含上游缩进格式；无映射 / `null` 时不出该字段）。

### `newapi_get_channel`
单渠道详情（需管理员 PAT）。参数：`id`（number，必填）。返回结构同上单条。

### `newapi_list_tokens`
当前 PAT 用户的 sk- 令牌列表。参数：`page?`、`page_size?`。令牌 `status`：1=启用 2=禁用 3=过期 4=耗尽；`key` 已掩码。

### `newapi_logs`
日志检索。参数：`page?`、`page_size?`、`type?`（0=全部 2=消费 5=错误）、`start_timestamp?` / `end_timestamp?`（Unix 秒）、`model_name?`、`token_name?`、`channel?`（渠道 ID）。零值过滤参数不发送。

### `newapi_usage_summary`
近 N 天用量汇总（走 `/api/data/` 聚合）：按模型的调用次数/tokens/消费额（quota 与 $ 双单位），按消费额降序。参数：`days?`（默认 7，夹取 1–365）。

### `newapi_pricing`
模型倍率定价。参数：`model?`（按模型名过滤）。**实例可能禁用该端点**——报错即禁用，属预期。

### `newapi_list_options`
系统设置键值对（需管理员 PAT），按 key 排序。**上游已过滤敏感键**（`*Token` / `*Secret` / `*Key` 等后缀与 `theme.frontend`），并附合成键 `CompletionRatioMeta`。查 autoban 配置（`AutomaticDisableChannelEnabled` / `AutomaticDisableStatusCodes` / `AutomaticDisableKeywords`）的入口。写操作见 admin 档 `newapi_update_option`。

### `newapi_success_rate`
请求成功率：日志 `type=2`（消费）vs `type=5`（错误）计数求比率。时间窗默认近 24h。

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `hours` | number | — | 窗口小时数，1–720（默认 24）；与起止时间戳互斥（后者优先） |
| `start_timestamp` / `end_timestamp` | number | — | 显式时间窗（Unix 秒），须成对 |
| `channel` | number | — | 按渠道 ID 过滤 |
| `model_name` / `token_name` | string | — | 按模型 / 令牌名过滤 |

注意：上游重试会对同一请求产生多条错误日志，比率为**近似值**。

### `newapi_autoban_config`
autoban 配置一次性总览（只读，需管理员 PAT），无参数。返回：

- `options`：自动封禁生态的全部系统设置键值（`AutomaticDisableChannelEnabled` / `AutomaticDisableStatusCodes` / `AutomaticRetryStatusCodes` / `AutomaticDisableKeywords` / `AutomaticEnableChannelEnabled` / `ChannelDisableThreshold` / `RetryTimes` / `monitor_setting.*` / `channel_affinity_setting.*`），并附 `global_switch_enabled` 布尔快捷判断
- `channels_auto_ban`：渠道级 auto_ban 普查——`on` / `off` / `unset` 计数 + `not_enabled` 清单（含原因）。**`unset`（上游 NULL）会被视为关**，是「欠费渠道不被自动禁用」的常见根因（gorm default 只覆盖新建行）
- `note`：写入口指引——全局开关与关键词 `newapi_update_option`、状态码 `newapi_autoban_codes`、渠道级 `newapi_update_channel(auto_ban)`

### `newapi_autoban_analysis`
自动封禁原因分析数据获取（只读，需管理员 PAT）。排查「渠道为什么被自动封禁」用这个。

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `channel` | number | — | 指定渠道 ID；缺省=全部 status=3（自动禁用）或带 status_reason 的渠道（含手动恢复后原因残留的） |
| `hours` | number | — | 错误日志回溯窗口，默认 24（1–720） |
| `sample` | number | — | 每渠道错误采样条数，默认 10（1–50） |

每渠道返回：`status` / `status_reason` / `balance` / `auto_ban` / `models` / `test_model` + 窗口内 type=5 错误日志的 `errors_total`（精确计数）、`by_content`（按错误内容聚合 top5，含 last_seen）、`by_model`（top5）、`last_error_at`、`likely_cause`（启发式：`quota_exhausted` → `model_issue` → `timeout` → `upstream_unreachable` → `other`，关键词首中；`no_error_logs` = 窗口内无错误）。`likely_cause` 仅供参考，以 `by_content` 采样明细为准；当前配置用 `newapi_autoban_config` 对照。

### `newapi_jiyuan_report`
基元渠道消费报表（需管理员 PAT + `[report]` 从库配置；未配置时调用明确报错）。默认区间=今天+前 3 天。

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name_like` | string | — | 渠道名 LIKE 过滤（默认「基元」） |
| `days` | number | — | 统计天数，含今天，1–30（默认 4） |

口径：tokens 只计成功请求（type=2）；消费 =（Prompt−缓存)/1M×输入价 + 缓存/1M×缓存价 + Completion/1M×输出价；计费模型取 `other.upstream_model_name`（映射后）回退 `model_name`；缺价模型按 0 计并列出。返回渠道汇总 / 按模型 / 渠道×模型明细 / 每日趋势 / 合计。

---

## ops 档（+6，`NEWAPI_WRITEMODE=ops`）

### `newapi_test_channel`
单渠道测试。参数：`id`（必填）、`model?`（测试模型名，缺省用渠道默认）。**测试失败是有效结果**——返回 `success:false` + 错误信息的正常 JSON：

```json
{ "success": false, "time_consuming": 3.2, "error": "余额不足 (402)" }
```

### `newapi_test_all_channels`
触发全量渠道测试（上游异步系统任务），返回 `task_id`；结果在面板任务中心或稍后用 `newapi_list_channels` 观察 `response_time` / 状态。

### `newapi_update_channel_balance`
刷新单渠道余额（先触发刷新再回查详情）。参数：`id`（必填）。部分渠道类型不支持，会返回业务错误。

### `newapi_set_channel_status`
启用/禁用渠道（记录为 manual operation）。参数：`id`（必填）、`enabled`（boolean，必填）。返回是否有实际变更（`changed`）。

### `newapi_create_token`
创建 API 令牌。参数：`name`（必填，≤50 字符）、`unlimited_quota?`、`remain_quota?`（quota 单位，500000=$1；unlimited 时忽略）、`expired_time?`（Unix 秒，-1=永不过期）、`model_limits?`（逗号分隔）、`group?`。返回 **id + 掩码 key**——完整 key 只在面板可见，无额度入参时自动 unlimited（避免 0 额度不可用令牌）。

### `newapi_delete_token`
删除令牌（不可恢复）。参数：`id`（必填）、`confirm`（必填，true）。

---

## admin 档（+6，`NEWAPI_WRITEMODE=admin`）

### `newapi_create_channel`
创建渠道。参数：`name` / `type`（1=OpenAI 兼容等）/ `key` / `models`（逗号分隔）必填；`base_url?`、`group?`（默认 default）、`model_mapping?`（JSON）、`priority?`、`weight?`、`test_model?`。key 只在此请求中传输；上游不回 id，**按名回查**返回渠道信息（id / name / type / status / model_mapping）。

### `newapi_update_channel`
更新渠道，**PATCH 语义：只发显式传入的字段**。参数：`id` 必填；其余 `name?`、`key?`（留空不改，多 key 渠道慎用）、`models?`、`base_url?`、`group?`、`model_mapping?`、`priority?`、`weight?`、`test_model?`、`type?`、`auto_ban?`（渠道级自动禁用开关，按「是否出现」收集：true→1 / false→0 / 不传→不改）、`tag?`（空串清除）。**不能改 status**——启停走 `newapi_set_channel_status`。成功后回查回显 `name` / `models` / `priority` / `group` / `model_mapping`（显式输出，改后为空串 = 映射已清空）。

### `newapi_delete_channel`
删除渠道（不可恢复）。参数：`id`、`confirm`（均必填）。

### `newapi_update_option`
修改系统设置 option。参数：`key` / `value` / `confirm`（均必填）。**危险操作：全局生效、key 不做白名单校验**；值均为字符串形态（布尔传 `"true"/"false"`，数字传字符串如 `"20"`、状态码 `"401,402,429"`）。先 `newapi_list_options` 核对现值，勿凭记忆造键。

### `newapi_autoban_codes`
autoban 状态码增删查改（写 `AutomaticDisableStatusCodes` / `AutomaticRetryStatusCodes`，保留其余配置）。

| 参数 | 类型 | 说明 |
|---|---|---|
| `action` | string | `list`（默认）/ `add` / `remove` / `set` |
| `target` | string | `disable`（自动禁用，默认）/ `retry`（自动重试） |
| `codes` | string | 逗号分隔 token：单码或闭区间，如 `402,400-499`（范围 100–599）；add/remove/set 必填 |
| `confirm` | boolean | 变更动作必传 true |

本地实现上游的区间代数：解析→排序→合并相邻/重叠区间，产出上游可解析的规范串。`add` 已覆盖项进 `already_covered`；`remove` 未覆盖项进 `not_found`（必要时拆分包含区间）；`set` 全量重写为规范形。注意：**上游匹配依赖有序区间**，手工写乱序串会静默失效——本工具保证规范形。

### `newapi_tag_channels`
按标签批量操作渠道（影响该 tag 下**所有**渠道）。参数：`action`（`edit` / `enable` / `disable`）、`tag`、`confirm` 必填；edit 另接受 `new_tag?`（重命名，不能为空）、`priority?`、`weight?`、`models?`、`model_mapping?`、`group?`（至少一项）。单渠道打标签/清标签用 `newapi_update_channel` 的 `tag` 字段。
