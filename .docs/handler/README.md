handler — MCP 工具层 handler 实现（薄壳约定）。read.go 8 个只读工具 / ops.go 6 个运维写工具 / admin.go 3 个渠道 CRUD；helpers.go 提供 Handler 签名、JSONResult/ErrResult 统一输出与参数小工具。工具声明（name/tier/参数）在 registry 模块汇总表。
