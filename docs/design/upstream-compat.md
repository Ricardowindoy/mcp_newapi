# 上游兼容与已知坑 / Upstream Compatibility & Known Pitfalls

> [English](upstream-compat.en.md) | 中文

new-api 迭代快，本项目按「**唯一耦合点**」策略适配：所有上游端点路径集中在 `internal/newapi/routes.go` 一处常量表，响应字段漂移只改对应域包的 DTO。对接新版本前以目标版本上游 router/controller 为准逐个核对（参考源码快照放 `.upstream/`，已 gitignore）。

## 已知坑清单（实测沉淀）

### 渠道管理

| # | 坑 | 规避 |
|---|---|---|
| 1 | **PUT 渠道是 PATCH 语义，但 `status` 字段被拒**——更新时带 status 会整体失败 | 启停走专门端点（`newapi_set_channel_status` 已封装），`newapi_update_channel` 的字段白名单已排除 status |
| 2 | **`created_time` / `test_time` / `response_time` / `balance` 是只读字段**——更新时传入会被清零 | 同上，只发要改的字段 |
| 3 | **创建渠道/令牌不回传 id** | 工具内部按名回查：create_channel 按名 List 回查返回 id；create_token 按 keyword 搜回 id + 掩码 key |
| 4 | **`pricing` 端点可能被实例禁用** | `newapi_pricing` 报错即禁用，属预期非 bug |
| 5 | **多 key 渠道的 key 追加**：上游 update 对 key 的处理对多 key 渠道有覆盖风险 | 不封装 key 追加；`newapi_update_channel` 传 key 时文档标注「多 key 渠道慎用」 |

### autoban / 自动禁用

| # | 坑 | 规避 |
|---|---|---|
| 6 | **渠道级 `auto_ban` 为 NULL 时自动禁用不生效**：上游 `GetAutoBan()` 对 nil 返回 false，gorm 的 `default:1` 只覆盖新建行，存量行是 NULL | 排查顺序：渠道 `auto_ban` → 全局开关 → 状态码表；用 `newapi_update_channel` 显式置 true/false 消除 NULL |
| 7 | **全局开关与状态码缺一不可**：`AutomaticDisableChannelEnabled` 未开或 `AutomaticDisableStatusCodes` 不含目标码（如 402）都不禁用 | `newapi_list_options` 查现值，`newapi_autoban_codes add 402` 追加（自动合并区间，勿手改覆盖） |
| 8 | **状态码区间串乱序会静默失效**：上游匹配依赖有序区间，手改乱序串不报错但不匹配 | 只用 `newapi_autoban_codes` 写入——本地代数保证产出规范形（排序+合并相邻/重叠） |
| 9 | **关键词黑名单 `AutomaticDisableKeywords`** 换行分隔、大小写不敏感 | 同走 list_options 查看 |

### 模型与路由

| # | 坑 | 规避 |
|---|---|---|
| 10 | **model_mapping 单次应用不串联**：映射键的目标值不会再过一遍映射表 | 做「模型替换」时把所有请求入口键（别名+真名）都直接指到最终上游模型 |
| 11 | **渠道报「模型已关闭」可能掩蔽欠费**：模型校验先于余额校验，真实 402 被挡住 | 换测试模型重测暴露真身：`newapi_update_channel` PATCH `test_model` 后 `newapi_test_channel`（测试模型无需在渠道 models 列表内） |
| 12 | **实际路由只按渠道 models 列表分发**：测试通过 ≠ 该模型会被路由 | 核对 `newapi_get_channel` 返回的 `models` 字段 |

### 其他

| # | 坑 | 规避 |
|---|---|---|
| 13 | **系统设置 option 的敏感键过滤在上游侧**：`*Token`/`*Secret`/`*Key` 后缀与 `theme.frontend` 读不到 | `newapi_list_options` 看到的就是上游给的；写操作 key 不做白名单校验，先 List 核对再改 |
| 14 | **PAT 调不了登录会话管理接口**（refresh/logout/sessions） | 本 MCP 不使用这些端点 |
| 15 | **渠道测试类端点是顶层字段响应**（`success`/`time_consuming`/`error` 直接在顶层，非信封 data） | 传输层已区分处理（DoTopLevel），工具把测试失败当有效结果返回 |

## 上游漂移时的维护落点

| 上游变化 | 只需改 |
|---|---|
| 端点路径/方法漂移 | `internal/newapi/routes.go` 常量表 |
| 某域响应字段变化 | 对应域子包 DTO（raw→Summary） |
| 新增业务域 | 新域子包 + routes 常量 + registry.go 表项 + handler |
| 鉴权方式变化 | 仅 `internal/newapi/client.go` |
| 配置项变化 | 仅 `internal/config` |
| 报表口径/SQL 变化 | 仅 `internal/reporter` |

更多设计细节见 [DESIGN.md](DESIGN.md)；模块级实现文档见 [.docs/](../../.docs/README.md)。
