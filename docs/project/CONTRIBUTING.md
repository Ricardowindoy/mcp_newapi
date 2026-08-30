# 贡献指南 / Contributing

> 中文 | [English](CONTRIBUTING.en.md)

## 开发环境

- Go **1.26+**（单二进制，无运行时依赖）
- 依赖：`github.com/mark3labs/mcp-go`（MCP SDK）、`github.com/BurntSushi/toml`、`github.com/go-sql-driver/mysql`

## 构建与测试（推送前硬性要求）

```bash
go vet ./... && go build ./...
go test ./...        # 纯逻辑单测（httptest 模拟上游），不碰真实网关
```

- 测试约定：**纯逻辑用 `httptest`**（参考 `internal/newapi/client_test.go`、`channels/ops_test.go`），不依赖真实网关。
- 修 bug 必须补对应测试用例，禁止只改代码不加测试。
- 提交信息沿用里程碑风格（`M1:` / `M2:` …），写清动机与验证结果。

## 架构速览

四层单向依赖，改代码前建议先读对应模块文档：

```
cmd/newapi-mcp → internal/mcp（registry 表 + server 装配）
                 └─ handler/（工具薄壳：解析参数→调域函数→输出）
                      ├─ internal/newapi/{status,models,channels,tokens,logs,options}（域子包）
                      │    └─ internal/newapi 根包（client 传输 + routes.go 唯一耦合点）
                      └─ internal/reporter（报表叶子包，直连从库）
internal/config（配置装载，横向模块）
```

- **分层规范与禁令**：[.docs/_mustread/模块职责边界设计规范.md](../../.docs/_mustread/模块职责边界设计规范.md)
- **构建/测试/安全红线**：[.docs/_mustread/开发规范.md](../../.docs/_mustread/开发规范.md)
- 各模块「功能说明 / 实现详细说明」：[.docs/README.md](../../.docs/README.md)（模块索引）
- 设计背景与决策：[DESIGN.md](../design/DESIGN.md)

## 新增一个工具的流程

1. `internal/newapi/<域>/` 新建/扩展域子包（raw DTO + Summary + 掩码 + 域函数 + 顶部上游契约注释）；外部数据源型（如报表）建 `internal/<域>/` 叶子包
2. `routes.go` 加端点常量（无 HTTP 端点则跳过）
3. `internal/mcp/registry.go` 加表项（Name / Tier / 参数声明 / Handler 工厂）
4. `internal/mcp/handler/<档位>.go` 实现薄壳 handler
5. `.docs/<模块>/` 文档同回合同步
6. httptest 单测覆盖新契约（含信封解包/掩码/错误语义）

## 安全红线（摘要，全文见 [SECURITY.md](SECURITY.md)）

- PAT / 渠道 key / 完整 sk- **永不入库、永不出现在任何响应**（掩码头尾各 4 位）
- 删除与高危变更类工具必须 `confirm=true`；`redemption/`（兑换码）不封装
- stdio 传输下 stdout 只跑 JSON-RPC，日志一律 stderr

## 文档双语约定

`docs/` 下中文为主文档（`<name>.md`），英译伴生文件为 `<name>.en.md`——改动中文篇后请同步英译。
