// Package mcp 装配 MCP server（工具注册全部走 registry.go 表驱动）。
package mcp

import (
	"github.com/mark3labs/mcp-go/server"

	"mcp_newapi/internal/newapi"
)

// NewServer 装配 MCP server。
// writemode: read（缺省）/ ops / admin，决定注册哪些档位的工具。
func NewServer(client *newapi.Client, writemode string) *server.MCPServer {
	s := server.NewMCPServer("newapi-mcp", "0.2.0",
		server.WithToolCapabilities(false),
	)
	registerTools(s, client, writemode)
	return s
}
