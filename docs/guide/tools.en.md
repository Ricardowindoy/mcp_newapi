# Tools Reference

> [中文](tools.md) | English

Parameters and semantics of all 25 tools. The tier is decided by `writemode`: **read = 13 tools (registered by default) / ops +6 / admin +6**; lower tiers do not register higher-tier tools.

## Common Conventions

- **Call format**: standard MCP `tools/call`; results are always returned as JSON text (`content[0].text`).
- **Error format**: tool error text like `[reachable=true status=403] message` — `reachable=true` means the gateway answered (business error), `reachable=false` means the gateway is unreachable (network error); agents can use this to tell the two apart.
- **Key masking**: channel keys / sk- tokens in any response are always masked (keeping 4 chars at each end). A full key is only passed in explicitly by the caller in create/update requests and is **never echoed back in any response**.
- **confirm gate**: deletion and high-impact change tools (delete_token / delete_channel / update_option / autoban_codes changes / tag_channels) require an explicit `confirm: true`, otherwise the call is rejected outright.
- **Business failure ≠ tool error**: a failed channel test and the like are normal JSON results with `success:false`, not errors.
- **quota conversion**: `500000 quota = $1`.

---

## read Tier (11 tools, registered by default)

### `newapi_status`
Public site status: version, start time, registration switch; plus one `/v1/models` probe (5s independent timeout) — **a 401/403 response still counts as the relay being alive** (any answer means reachable). No PAT required.

### `newapi_list_models`
Site-wide available models; `data` passes through as `group → model array` with in-group sorting; plus `group_count` / `model_count` summaries. No PAT required.

### `newapi_list_channels`
Paginated channel list (**requires an admin PAT**).

| Parameter | Type | Required | Description |
|---|---|---|---|
| `page` | number | — | Page number, starting from 1 (default 1) |
| `page_size` | number | — | Items per page (default 20) |
| `status` | number | — | Status filter: 0=all (default) 1=enabled 2=manually disabled 3=auto disabled |

```json
{ "page": 1, "page_size": 20, "total": 42, "items": [
  { "id": 31, "name": "基元2_x", "type": 1, "status": 1, "status_reason": "",
    "balance": 12.5, "base_url": "https://…", "models": "deepseek-v4-pro,…",
    "group": "default", "model_mapping": "{\"deepseek-v4-flash\":\"deepseek-v4-flash-0731\"}",
    "priority": 0, "weight": 0, "test_model": "",
    "response_time": 2340, "used_quota": 812345.6, "key": "sk-a***7890" } ] }
```

Each channel item includes `model_mapping` (the model-redirect JSON string verbatim, including upstream indentation; omitted when there is no mapping or the upstream value is `null`).

### `newapi_get_channel`
Single-channel detail (requires an admin PAT). Parameter: `id` (number, required). Returns the same structure as a single item above.

### `newapi_list_tokens`
sk- token list of the current PAT user. Parameters: `page?`, `page_size?`. Token `status`: 1=enabled 2=disabled 3=expired 4=exhausted; `key` is masked.

### `newapi_logs`
Log search. Parameters: `page?`, `page_size?`, `type?` (0=all 2=consumption 5=error), `start_timestamp?` / `end_timestamp?` (Unix seconds), `model_name?`, `token_name?`, `channel?` (channel ID). Zero-valued filter parameters are not sent.

### `newapi_usage_summary`
Usage summary for the last N days (via the `/api/data/` aggregation): per-model call counts / tokens / spend (in both quota and $), sorted by spend descending. Parameter: `days?` (default 7, clamped to 1–365).

### `newapi_pricing`
Model-ratio pricing. Parameter: `model?` (filter by model name). **The instance may have this endpoint disabled** — an error means it is disabled; expected behavior.

### `newapi_list_options`
System settings key-value pairs (requires an admin PAT), sorted by key. **Sensitive keys are filtered upstream** (suffixes like `*Token` / `*Secret` / `*Key` and `theme.frontend`), with the synthesized key `CompletionRatioMeta` appended. Entry point for inspecting autoban configuration (`AutomaticDisableChannelEnabled` / `AutomaticDisableStatusCodes` / `AutomaticDisableKeywords`). For writes, see the admin-tier `newapi_update_option`.

### `newapi_success_rate`
Request success rate: ratio of log `type=2` (consumption) vs `type=5` (error) counts. Time window defaults to the last 24h.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `hours` | number | — | Window in hours, 1–720 (default 24); mutually exclusive with explicit start/end timestamps (the latter wins) |
| `start_timestamp` / `end_timestamp` | number | — | Explicit time window (Unix seconds), must be paired |
| `channel` | number | — | Filter by channel ID |
| `model_name` / `token_name` | string | — | Filter by model / token name |

Note: upstream retries produce multiple error-log entries for one request, so the rate is an **approximation**.

### `newapi_autoban_config`
One-shot autoban configuration overview (read-only, requires an admin PAT), no parameters. Returns:

- `options`: all system-setting key/values in the auto-disable ecosystem (`AutomaticDisableChannelEnabled` / `AutomaticDisableStatusCodes` / `AutomaticRetryStatusCodes` / `AutomaticDisableKeywords` / `AutomaticEnableChannelEnabled` / `ChannelDisableThreshold` / `RetryTimes` / `monitor_setting.*` / `channel_affinity_setting.*`), plus the `global_switch_enabled` boolean shortcut
- `channels_auto_ban`: channel-level auto_ban census — `on` / `off` / `unset` counts + the `not_enabled` list (with reasons). **`unset` (upstream NULL) is treated as off** — a common root cause of "out-of-credit channels never get auto-disabled" (the gorm default only covers newly created rows)
- `note`: write-entry pointers — global switch & keywords via `newapi_update_option`, status codes via `newapi_autoban_codes`, channel-level via `newapi_update_channel(auto_ban)`

### `newapi_autoban_analysis`
Auto-ban reason analysis data fetch (read-only, requires an admin PAT). Use this to investigate "why was this channel auto-disabled".

| Parameter | Type | Required | Description |
|---|---|---|---|
| `channel` | number | — | Specific channel ID; default = all channels with status=3 (auto-disabled) or a non-empty status_reason (including manually re-enabled ones with a lingering reason) |
| `hours` | number | — | Error-log lookback window, default 24 (1–720) |
| `sample` | number | — | Error samples per channel, default 10 (1–50) |

Per channel it returns: `status` / `status_reason` / `balance` / `auto_ban` / `models` / `test_model` plus, for type=5 error logs within the window: `errors_total` (exact count), `by_content` (top 5 grouped by error content, with last_seen), `by_model` (top 5), `last_error_at`, and `likely_cause` (heuristic: `quota_exhausted` → `model_issue` → `timeout` → `upstream_unreachable` → `other`, first keyword hit wins; `no_error_logs` = no errors in window). Treat `likely_cause` as a hint — `by_content` samples are the ground truth; cross-check the current config with `newapi_autoban_config`.

### `newapi_jiyuan_report`
Jiyuan channel consumption report (requires an admin PAT + the `[report]` replica configuration; unconfigured calls fail with an explicit error). Default window = today + the previous 3 days.

| Parameter | Type | Required | Description |
|---|---|---|---|
| `name_like` | string | — | Channel-name LIKE filter (default `基元`, i.e. "Jiyuan") |
| `days` | number | — | Days covered, including today, 1–30 (default 4) |

Methodology: tokens count successful requests only (type=2); spend = (Prompt − cached)/1M × input price + cached/1M × cache price + Completion/1M × output price; the billing model uses `other.upstream_model_name` (post-mapping) with fallback to `model_name`; models without pricing are counted as 0 and listed. Returns per-channel summary / per-model / channel×model breakdown / daily trend / totals.

---

## ops Tier (+6, `NEWAPI_WRITEMODE=ops`)

### `newapi_test_channel`
Single-channel test. Parameters: `id` (required), `model?` (test model name; defaults to the channel default). **A failed test is a valid result** — a normal JSON response with `success:false` plus the error message:

```json
{ "success": false, "time_consuming": 3.2, "error": "余额不足 (402)" }
```

### `newapi_test_all_channels`
Triggers a full channel test (an upstream async system task) and returns `task_id`; results appear in the dashboard task center, or later via `newapi_list_channels` (watch `response_time` / status).

### `newapi_update_channel_balance`
Refreshes a single channel's balance (triggers the refresh first, then re-fetches the detail). Parameter: `id` (required). Some channel types do not support this and return a business error.

### `newapi_set_channel_status`
Enables/disables a channel (recorded as a manual operation). Parameters: `id` (required), `enabled` (boolean, required). Returns whether an actual change occurred (`changed`).

### `newapi_create_token`
Creates an API token. Parameters: `name` (required, ≤50 chars), `unlimited_quota?`, `remain_quota?` (quota units, 500000=$1; ignored when unlimited), `expired_time?` (Unix seconds, -1=never expires), `model_limits?` (comma-separated), `group?`. Returns **id + masked key** — the full key is only visible in the dashboard; with no quota parameter given, the token is automatically unlimited (avoiding an unusable 0-quota token).

### `newapi_delete_token`
Deletes a token (irreversible). Parameters: `id` (required), `confirm` (required, true).

---

## admin Tier (+6, `NEWAPI_WRITEMODE=admin`)

### `newapi_create_channel`
Creates a channel. Required: `name` / `type` (1=OpenAI-compatible, etc.) / `key` / `models` (comma-separated); optional: `base_url?`, `group?` (default default), `model_mapping?` (JSON), `priority?`, `weight?`, `test_model?`. The key is transmitted only in this request; the upstream does not return an id, so the tool **looks the channel up by name** and returns channel info (id / name / type / status / model_mapping).

### `newapi_update_channel`
Updates a channel, **PATCH semantics: only explicitly passed fields are sent**. Parameters: `id` required; the rest optional: `name?`, `key?` (empty = unchanged; use with care on multi-key channels), `models?`, `base_url?`, `group?`, `model_mapping?`, `priority?`, `weight?`, `test_model?`, `type?`, `auto_ban?` (channel-level auto-disable switch, collected by presence: true→1 / false→0 / absent→unchanged), `tag?` (empty string clears it). **status cannot be changed** — use `newapi_set_channel_status` to enable/disable. On success it echoes back `name` / `models` / `priority` / `group` / `model_mapping` (explicitly output; an empty string after the update means the mapping was cleared).

### `newapi_delete_channel`
Deletes a channel (irreversible). Parameters: `id`, `confirm` (both required).

### `newapi_update_option`
Modifies a system settings option. Parameters: `key` / `value` / `confirm` (all required). **Dangerous operation: takes effect globally, keys are not whitelist-validated**; values are always strings (booleans as `"true"/"false"`, numbers as strings like `"20"`, status codes `"401,402,429"`). Verify current values with `newapi_list_options` first; do not invent keys from memory.

### `newapi_autoban_codes`
autoban status-code add/remove/list/set (writes `AutomaticDisableStatusCodes` / `AutomaticRetryStatusCodes`, keeping the rest of the configuration).

| Parameter | Type | Description |
|---|---|---|
| `action` | string | `list` (default) / `add` / `remove` / `set` |
| `target` | string | `disable` (auto-disable, default) / `retry` (auto-retry) |
| `codes` | string | Comma-separated tokens: single codes or closed ranges, e.g. `402,400-499` (range 100–599); required for add/remove/set |
| `confirm` | boolean | Mutating actions must pass true |

Implements the upstream's range algebra locally: parse → sort → merge adjacent/overlapping ranges, producing a canonical string the upstream can parse. `add` reports already-covered items in `already_covered`; `remove` reports uncovered items in `not_found` (splitting containing ranges when necessary); `set` rewrites everything into canonical form. Note: **upstream matching relies on ordered ranges** — a hand-written out-of-order string fails silently; this tool guarantees canonical form.

### `newapi_tag_channels`
Batch channel operations by tag (affects **all** channels under that tag). Required parameters: `action` (`edit` / `enable` / `disable`), `tag`, `confirm`; `edit` additionally accepts `new_tag?` (rename, must not be empty), `priority?`, `weight?`, `models?`, `model_mapping?`, `group?` (at least one of them). To tag/untag a single channel, use the `tag` field of `newapi_update_channel`.
