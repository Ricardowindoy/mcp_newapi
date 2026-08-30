#!/usr/bin/env bash
# newapi-mcp 挂载 wrapper 示例（DSH / cordis 用）
# 生产用法：二进制 + 配置文件 + token_file 间接引用，环境变量一个都不需要。
#   ~/.dsh/newapi-mcp.toml 内容见仓库 newapi-mcp.example.toml
#   ~/.dsh/newapi.pat 为面板 PAT（chmod 600）
# 注意：stdio 传输下 stdout 只跑 JSON-RPC；wrapper 内不要 echo 任何东西。

exec /opt/newapi-mcp/bin/newapi-mcp --config "$HOME/.dsh/newapi-mcp.toml"
