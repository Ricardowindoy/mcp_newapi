// Package mcp 装配 MCP server（工具注册全部走 registry.go 表驱动）。
package mcp

import (
	"github.com/mark3labs/mcp-go/server"

	"mcp_newapi/internal/newapi"
	"mcp_newapi/internal/reporter"
)

// NewServer 装配 MCP server。
// writemode: read（缺省）/ ops / admin，决定注册哪些档位的工具。
// rep: 报表从库连接（entry 从 config 注入；nil=报表未配置，工具仍注册、调用时报配置错误）。
func NewServer(client *newapi.Client, writemode string, rep *reporter.Store) *server.MCPServer {
	s := server.NewMCPServer("newapi-mcp", "0.5.0",
		server.WithToolCapabilities(false),
	)
	registerTools(s, client, writemode, rep)
	return s
}
