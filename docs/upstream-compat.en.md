# Upstream Compatibility & Known Pitfalls

> [中文](upstream-compat.md) | English

new-api iterates fast; this project adapts with a "**single coupling point**" strategy: all upstream endpoint paths live in one constant table at `internal/newapi/routes.go`, and response-field drift only requires changing the DTO of the corresponding domain package. Before wiring up a new version, verify endpoint by endpoint against the target version's upstream router/controller (a reference source snapshot is kept in `.upstream/`, gitignored).

## Known-Pitfall List (distilled from hands-on testing)

### Channel Management

| # | Pitfall | Mitigation |
|---|---|---|
| 1 | **PUT channel is PATCH semantics, but the `status` field is rejected** — including status in an update fails the whole request | Enable/disable goes through the dedicated endpoint (already wrapped as `newapi_set_channel_status`); the field whitelist of `newapi_update_channel` already excludes status |
| 2 | **`created_time` / `test_time` / `response_time` / `balance` are read-only fields** — passing them in an update zeroes them out | Same as above: only send the fields to change |
| 3 | **Creating a channel/token does not return the id** | The tool looks it up by name internally: create_channel lists channels by name and returns the id; create_token searches by keyword and returns the id + masked key |
| 4 | **The `pricing` endpoint may be disabled on the instance** | An error from `newapi_pricing` means disabled — expected, not a bug |
| 5 | **Appending keys on multi-key channels**: the upstream's key handling on update carries an overwrite risk for multi-key channels | Key appending is not wrapped; the docs mark `newapi_update_channel`'s `key` as "use with care on multi-key channels" |

### autoban / auto-disable

| # | Pitfall | Mitigation |
|---|---|---|
| 6 | **Auto-disable does not take effect when channel-level `auto_ban` is NULL**: upstream `GetAutoBan()` returns false for nil; gorm's `default:1` only covers newly created rows, so existing rows are NULL | Check in order: channel `auto_ban` → global switch → status-code table; set true/false explicitly via `newapi_update_channel` to eliminate NULL |
| 7 | **The global switch and the status codes are both required**: no auto-disable happens when `AutomaticDisableChannelEnabled` is off or `AutomaticDisableStatusCodes` lacks the target code (e.g. 402) | Check current values with `newapi_list_options`, then append with `newapi_autoban_codes add 402` (ranges are merged automatically; do not hand-edit and overwrite) |
| 8 | **Out-of-order status-code range strings fail silently**: upstream matching relies on ordered ranges; a hand-edited out-of-order string raises no error but never matches | Write only through `newapi_autoban_codes` — the local algebra guarantees canonical output (sorted + adjacent/overlapping merged) |
| 9 | **Keyword blacklist `AutomaticDisableKeywords`** is newline-separated and case-insensitive | View it the same way via list_options |

### Models & Routing

| # | Pitfall | Mitigation |
|---|---|---|
| 10 | **model_mapping applies once, not chained**: the target value of a mapping key is not run through the mapping table again | When doing "model replacement", point all request-entry keys (alias + real name) directly at the final upstream model |
| 11 | **A channel reporting "model disabled" can mask overdue balance**: model validation runs before balance validation, so a real 402 is blocked | Re-test with a different test model to expose the real cause: `newapi_update_channel` PATCH `test_model`, then `newapi_test_channel` (the test model need not be in the channel's models list) |
| 12 | **Actual routing dispatches only by the channel's models list**: a passing test ≠ the model will be routed | Cross-check the `models` field returned by `newapi_get_channel` |

### Others

| # | Pitfall | Mitigation |
|---|---|---|
| 13 | **Sensitive-key filtering of system settings options happens upstream**: suffixes like `*Token`/`*Secret`/`*Key` and `theme.frontend` cannot be read | What `newapi_list_options` shows is exactly what upstream gives; write-operation keys are not whitelist-validated — List first to verify, then change |
| 14 | **A PAT cannot call the login-session management endpoints** (refresh/logout/sessions) | This MCP does not use these endpoints |
| 15 | **Channel-test endpoints respond with top-level fields** (`success`/`time_consuming`/`error` sit at the top level, not inside an envelope `data`) | The transport layer handles this distinction (DoTopLevel); tools return a failed test as a valid result |

## Maintenance Touch Points When Upstream Drifts

| Upstream change | Only touch |
|---|---|
| Endpoint path/method drift | `internal/newapi/routes.go` constant table |
| Response fields changed in a domain | The corresponding domain sub-package DTO (raw→Summary) |
| New business domain | New domain sub-package + routes constants + registry.go table entry + handler |
| Auth method change | Only `internal/newapi/client.go` |
| Config item change | Only `internal/config` |
| Report methodology/SQL change | Only `internal/reporter` |

For more design details see [DESIGN.md](../DESIGN.md); for per-module implementation docs see [.docs/](../.docs/README.md).
