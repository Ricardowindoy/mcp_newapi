registry — 工具注册层（internal/mcp 包根）。toolRegistry 汇总表是全部 17 个对外 MCP 工具的唯一索引（Name/Tier/声明/Handler 工厂）；registerTools 按 writemode 档位过滤注册（低档不含高档）；server.go 装配 MCPServer。
