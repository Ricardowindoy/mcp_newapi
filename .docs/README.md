# 项目文档（.docs vault）

> Obsidian vault：人和 AI 共读的项目文档知识库。每模块一个文件夹（module.json + 功能说明 + 实现详细说明）。

## 模块目录

按依赖自底向上：

| 模块 | 一句话 |
|---|---|
| config | TOML 配置装载：默认值 < 文件 < 环境变量，token_file 间接引用 |
| client | new-api HTTP 传输：鉴权/信封解包/APIError + routes.go 端点常量 + 掩码 |
| domains | 一域一包（status/models/channels/tokens/logs/options）：DTO 裁剪 + 域函数 + 上游契约 |
| reporter | 报表域：直连从库聚合基元消费报表（叶子包，DSN 经 entry 从 config 注入） |
| handler | 23 个 MCP 工具 handler 薄壳：解析参数 → 调域函数 → 统一输出 |
| registry | toolRegistry 表驱动汇总表（23 工具唯一索引）+ writemode 档位过滤装配 |
| entry | main.go：--config 加载 → 装配 → stdio 启动，零业务逻辑 |

## 必读文档

- `_mustread/开发规范.md` — 构建/测试/安全红线/上游兼容（更新需用户显式同意）
- `_mustread/模块职责边界设计规范.md` — 分层单向依赖与各层禁令（同上）
- `_mustread/文档编写规范.md` — 三层体系/双语/链接/四件套/风格/变更流程（同上）

## 规则

1. 阅读/引用某模块代码前，先读该模块「功能说明」
2. 修改某模块代码前，必须读「实现详细说明」——含 module.json `uses` 声明的底层模块
3. 修改代码后同一回合更新对应模块文档，避免漂移
4. 必读文档（_mustread/、本 README）修改必须走 doc_propose，用户 approve 后生效
5. 大小上限：本 README 2KB / 模块 README 0.5KB / 功能说明 8KB / 实现详细说明 16KB，超出拆章节

## 上游契约

本项目是 new-api 的外挂 MCP 服务器。端点路径集中在 client 模块 routes.go，DTO 裁剪在各域包；上游漂移按「模块职责边界设计规范」的维护映射落点修改。上游参考快照在 `.upstream/`（gitignore）。
