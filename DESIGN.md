# newapi-mcp 项目设计文档

> 一个用于操作 [new-api](https://github.com/QuantumNous/new-api) 网关的 MCP (Model Context Protocol) 服务器，让 Agent 能够读取 new-api 运行状态、管理渠道/令牌/用户，并对网关进行受控运维操作。

- **版本**：v0.1（设计稿）
- **目标部署**：本机 DSH（DeepSeek Harness，127.0.0.1:3082）作为 MCP client，经 cordis.patch.yml 挂载；new-api 实例为 newapi.ashou.site（及任意 OpenAI 风格 new-api 部署）。

---

## 1. 背景与目标

new-api 是流行的 LLM 聚合分发网关（渠道管理、令牌分发、计费统计）。目前 Agent 想知道「网关有哪些模型可用、某渠道是否挂了、token 用量多少、某渠道余额还剩多少」，只能靠 curl 手动查或上浏览器面板。

本项目提供一个 MCP server，把 new-api 的管理面（`/api/*`）与中继面状态封装成结构化工具，供 Agent 直接调用：

**目标**
1. **只读状态优先**：模型列表、渠道健康、余额、用量日志、系统状态，低权限即可用。
2. **受控写操作**：渠道的启停/测试/余额刷新、令牌管理，需显式开启写模式。
3. **最小配置**：只需 `NEWAPI_BASE_URL` + `NEWAPI_TOKEN`（面板 PAT）两个环境变量。
4. **安全边界清晰**：敏感操作（删除、系统设置修改）默认不暴露。

**非目标**
- 不做中继流量代理（Agent 调模型仍走 DSH 自己的 llm 路由）。
- 不做 new-api 前端/数据库的直接操作（全部经 HTTP API，兼容官方版本，不依赖 DB schema）。

## 2. 技术选型

| 项 | 选择 | 理由 |
|---|---|---|
| 语言 | Go 1.26（本机 /opt/go/bin） | 单二进制、无运行时依赖、交叉编译方便；本机已有完整工具链 |
| MCP SDK | `github.com/mark3labs/mcp-go` | 成熟的 Go MCP 实现，支持 stdio + streamable HTTP |
| 传输 | stdio（首选）+ Streamable HTTP（可选） | stdio 由 DSH/cordis 直接拉起；HTTP 便于远程部署 |
| HTTP 客户端 | net/http | 直连，注意走 `no_proxy`（newapi.ashou.site 已在 DSH 的 no_proxy 列表） |

## 3. new-api API 契约要点（依据官方 docs/authentication.md 与 router 源码）

- **鉴权**：面板 PAT（用户设置里生成的 System Access Token），`Authorization: Bearer <pat>`（也兼容无 Bearer 前缀的裸值）。**不再需要** `New-Api-User` 请求头（官方已移除该要求）。PAT 无法调用登录会话管理接口（refresh/logout/sessions），本 MCP 也不使用它们。
- **响应格式**：统一 `{ "success": bool, "message": string, "data": ... }`。
- **分页**：列表类接口用 `p`（页码，1 起）+ `page_size` 查询参数。
- **主要端点**（基于 one-api 血统的 new-api 路由，实现时以目标版本 router 为准逐个核对）：

| 分组 | 端点 | 用途 |
|---|---|---|
| 状态 | `GET /api/status` | 站点公开状态（无需鉴权）：版本、公告、注册开关 |
| 模型 | `GET /api/models`、`GET /api/pricing` | 可用模型列表 / 模型倍率定价 |
| 渠道 | `GET /api/channel/`、`GET /api/channel/:id`、`POST /api/channel/`、`PUT /api/channel/`、`DELETE /api/channel/:id` | 渠道 CRUD（需管理员 PAT） |
| 渠道运维 | `GET /api/channel/test/:id?model=`、`GET /api/channel/update_balance/:id`、`PUT /api/channel/`（status 字段切换启停）、`GET /api/channel/tag/:tag` | 测试渠道可用性 / 刷新余额 / 启停 / 按标签操作 |
| 令牌 | `GET /api/token/`、`POST /api/token/`、`PUT /api/token/`、`DELETE /api/token/:id` | 用户级 sk- 令牌管理（普通 PAT 即可） |
| 用户 | `GET /api/user/:id`、`GET /api/user/self`、`GET /api/user/` | 用户/额度查询（管理员可列全部） |
| 日志 | `GET /api/log/`、`GET /api/log/stat`、`GET /api/log/token` | 消费/错误日志与统计（Agent 判断「网关最近是否报错」的核心数据源） |
| 兑换码 | `GET /api/redemption/`、`POST /api/redemption/` | 额度兑换码（管理员） |
| 系统设置 | `GET /api/option/`、`PUT /api/option/` | 运营设置（高危，默认不暴露） |

> 兼容性策略：new-api 迭代快，MCP 内所有路径集中在 `internal/newapi/routes.go` 一处常量表，端点漂移只改一处。

## 4. 工具集设计（MCP Tools）

按权限分三档：**read**（默认）/ **ops**（`NEWAPI_WRITEMODE=ops`）/ **admin**（`NEWAPI_WRITEMODE=admin`）。每档都是环境变量显式开启，未开启时对应工具不注册（而非注册了再拒绝），减少 Agent 误选。

### 4.1 read 档（默认）

| 工具 | 参数 | 说明 |
|---|---|---|
| `newapi_status` | — | `GET /api/status`：版本、系统配置摘要；附带一次对 `/v1/models` 的探测判断 relay 活性 |
| `newapi_list_models` | — | 全站可用模型 + 分组归属 |
| `newapi_list_channels` | `page?, page_size?, status?` | 渠道列表（id、名称、类型、状态、余额、标签、模型数） |
| `newapi_get_channel` | `id` | 单渠道详情（**掩码 key**，见 §7 安全） |
| `newapi_list_tokens` | `page?` | 当前 PAT 用户的令牌列表（sk- 值默认掩码） |
| `newapi_logs` | `type?, start_timestamp?, end_timestamp?, page?` | 消费/错误日志检索 |
| `newapi_usage_summary` | `days?`（默认 7） | 聚合 `log/stat`：按模型/渠道的 tokens、次数、消费额 |
| `newapi_pricing` | `model?` | 模型倍率与定价 |

### 4.2 ops 档（+）

| 工具 | 参数 | 说明 |
|---|---|---|
| `newapi_test_channel` | `id, model?` | 对渠道发一次测试请求，返回延迟/错误 |
| `newapi_update_channel_balance` | `id` | 刷新渠道余额 |
| `newapi_set_channel_status` | `id, enabled: bool` | 启用/禁用渠道（PUT status） |
| `newapi_create_token` / `newapi_delete_token` | 令牌名、额度、过期等 | 令牌生命周期（返回的完整 sk- 仅本次响应可见） |
| `newapi_batch_test_channels` | `tag?` | 按标签批量测试并汇总健康报告 |

### 4.3 admin 档（+）

| 工具 | 参数 | 说明 |
|---|---|---|
| `newapi_create_channel` / `newapi_update_channel` | 渠道全字段 | 渠道 CRUD（写 key 必须显式传入） |
| `newapi_delete_channel` | `id, confirm: true` | 删除需双重确认参数 |
| `newapi_list_users` | `page?` | 用户与额度 |
| **明确不做** | — | `option/`（系统设置）、`redemption/`（兑换码）暂不封装，避免 Agent 误改全局配置 |

### 4.4 资源（MCP Resources，可选）

- `newapi://status`、`newapi://models`——便于 client 侧作为上下文预取，不占工具调用。

## 5. 架构：分层 + 域模块自治 + 表驱动注册

四层职责，单向依赖（mcp 工具层 → newapi 域方法 → client 传输；config 独立横向模块）：

```
┌─ internal/mcp/（工具层·薄壳）────────────────────────────────┐
│ registry.go    ★对外服务汇总表（表驱动）：17 个工具的唯一声明  │
│                （Name/Tier/描述/参数/Handler 工厂）           │
│ server.go      装配：遍历表按 writemode 过滤注册              │
│ ── handler/ 子包（handler 实现，全部薄壳）────────────────    │
│   read.go     read 档：参数解析→调域函数→输出                │
│   ops.go      ops 档（含 confirm 校验）                      │
│   admin.go    admin 档                                       │
│   helpers.go  JSONResult/ErrResult（唯一公共件）             │
└──────────────────────────────────────────────────────────┘
                    ↓ 只调域函数，不含业务逻辑
┌─ internal/newapi/（API 层·根=传输层，域=子包）────────────┐
│ client.go      传输层：HTTP+鉴权+信封解包+APIError，零业务│
│ routes.go      上游端点路径常量（★唯一耦合点）            │
│ page.go        分页壳 Paged/PageResult + Itoa 工具        │
│ consts.go      QuotaPerUnit 等业务常量                    │
│ mask.go        密钥掩码工具                               │
│ ── 域子包（一域一包：DTO+域函数+上游契约注释）─────────  │
│ status/        站点状态 + relay 探测                      │
│ models/        模型列表 / 定价                            │
│ channels/      渠道：channel.go 读 / ops.go 运维 / admin.go 管理 │
│ tokens/        令牌管理（列表/创建/删除，读写一体）       │
│ logs/          日志 / 统计 / dashboard 按模型聚合         │
└──────────────────────────────────────────────────────┘
cmd/newapi-mcp/main.go  # 入口：--config 加载配置，装配，stdio 启动
```

**上游更新维护映射**（高内聚低耦合的落点）：

| 上游变化 | 只需改 |
|---|---|
| 端点路径/方法漂移 | `routes.go` 常量 |
| 某域响应字段变化 | 对应域文件的 DTO（raw→summary） |
| 新增业务域 | 新域子包 + routes 常量 + registry.go 表项 + handler |
| 新增/调整对外工具 | `registry.go` 表项（+handler/handler 包实现），server.go 不动 |
| 鉴权方式变化 | 仅 `client.go` |
| 配置项变化 | 仅 `internal/config` |

**请求链路**：MCP tool → 参数校验 → `newapi.Client` 域方法 → HTTP（10s 超时）→ 统一解包 `{success,message,data}`（渠道测试等顶层字段端点走 `DoTopLevel`）→ DTO 裁剪+掩码 → JSON 返回给 Agent。

**错误约定**：new-api 返回 `success:false` 时，MCP 工具返回结构化错误（含 HTTP 状态码 + message），不 panic、不吞错；网络错误明确标注「网关不可达」以便 Agent 区分「网关挂了」和「查询本身错」。业务性失败（渠道测试不通）是**有效结果**而非错误。

## 6. 配置（独立模块 internal/config）

TOML 配置文件，优先级：**内置默认值 < TOML 文件 < 环境变量**。

```toml
[newapi]
base_url = "https://newapi.ashou.site"   # 必填
token = ""                               # 面板 PAT；建议留空用 token_file
token_file = "/home/radxa/.dsh/newapi.pat" # 读首行作 PAT（0600），密钥不落配置本体
writemode = "ops"                        # read / ops / admin
timeout_seconds = 10
```

- 加载：`newapi-mcp --config <path>`；不传则只用默认值+环境变量（向后兼容原 env 方式）
- 环境变量覆盖：`NEWAPI_BASE_URL` / `NEWAPI_TOKEN` / `NEWAPI_WRITEMODE` / `NEWAPI_TIMEOUT`
- 配置模块自带单测（TOML 解析/env 覆盖/token_file/校验），示例见 `newapi-mcp.example.toml`

DSH 挂载（cordis.patch.yml，改后需重启 DSH 生效）：

```yaml
mcp-newapi:
  name: '@deepseek-ai/dsh-mcp-client'
  config:
    serverName: newapi
    transport: stdio
    command: /home/radxa/.dsh/mcp-newapi-wrapper.sh   # exec newapi-mcp --config ~/.dsh/newapi-mcp.toml
    args: []
# 生产配置在 ~/.dsh/newapi-mcp.toml（0600）：base_url + token_file(~/.dsh/newapi.pat) + writemode
```

> 注意：PAT 经 token_file 引用 `~/.dsh/newapi.pat`（0600），不写进配置文件本体，不进 git。

## 7. 安全设计（实现修订）

1. **掩码原则**：所有工具返回中的上游渠道 key、sk- 令牌值一律掩码（保留头尾各 4 位）。渠道详情端点上游本就不回 key；令牌完整 key 创建后只在面板可见——**任何 MCP 响应都不透出完整 key**（「创建令牌」返回 id + 掩码，提示去面板复制）。
2. **能力分档靠不注册**：低档模式下写工具根本不存在，Agent 无法「试探」。当前生产档位：`ops`（14 工具）；`admin` 档（17 工具）按需在 wrapper 里切换。
3. **删除类工具带 `confirm` 必填参数**（delete_token/delete_channel），不传直接拒绝。
4. **不暴露 option/redemption**：全局运营配置的修改风险远大于收益；也不封装多 key 渠道的 key 追加模式。
5. **PAT 权限天然继承**：MCP 能做的最多等于该 PAT 账号在面板里能做的，不额外放大权限。
6. **日志与凭据**：MCP 进程自身不打请求体日志（避免 key 落盘）；PAT 经配置 `token_file` 引用 `~/.dsh/newapi.pat`（0600），不进配置文件本体、不进 git。

## 8. 实现里程碑（进度）

1. ✅ **M1 骨架**：client.go（鉴权+解包+超时）、mcp-go stdio server、`newapi_status` 打通，DSH 挂载。
2. ✅ **M2 read 档**：8 工具 + 掩码层，全部对实装端点核对（发现：PUT status 被上游拒绝、pricing 被实例禁用、/api/data/ 为聚合数据源）。
3. ✅ **M3 ops 档**：渠道测试/启停/余额/全量测试 + 令牌管理（创建后按名回查 id+掩码），分层重构（§5）。
4. ✅ **M4 admin 档**：渠道 CRUD（创建带 key 只进不出；更新 PATCH 语义；删除 confirm），全闭环实测（创建 id=108 → 更新 → 删除）。
5. ✅ **M5 打磨**：README、单元测试（client 解包/掩码/状态语义 + config 模块）。
6. ✅ **M6 架构升级**：internal/config 配置模块（TOML，默认<文件<env，token_file 间接引用）+ registry.go 表驱动注册（17 工具唯一汇总表）；生产配置 ~/.dsh/newapi-mcp.toml，wrapper 简化为 --config 启动。
7. ⬜ **M7 可选**：Streamable HTTP 传输、admin 档生产启用。

## 9. 验收场景

- Agent 问「网关现在有哪些模型、glm-5.3 走哪个渠道」→ `newapi_list_models` + `newapi_list_channels`。
- Agent 巡检：「测试所有启用渠道并汇报失败的」→ `newapi_test_all_channels`。
- Agent 运维：「渠道 X 余额快用完了/持续 5xx，先禁用它」→ `newapi_test_channel` → `newapi_set_channel_status`。
- Agent 记账：「最近 7 天各模型消费多少」→ `newapi_usage_summary`。
