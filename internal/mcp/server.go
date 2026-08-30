// Package mcp 装配 MCP server 与工具（按 NEWAPI_WRITEMODE 分档注册）。
package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp_newapi/internal/newapi"
)

// Writemode 权限档位。
const (
	ModeRead  = "read"  // 缺省
	ModeOps   = "ops"
	ModeAdmin = "admin"
)

// NewServer 装配 MCP server。
func NewServer(client *newapi.Client, writemode string) *server.MCPServer {
	s := server.NewMCPServer("newapi-mcp", "0.1.0",
		server.WithToolCapabilities(false),
	)

	registerReadTools(s, client) // read 档始终注册
	switch writemode {
	case ModeAdmin:
		// M4: registerAdminTools(s, client)
		fallthrough
	case ModeOps:
		// M3: registerOpsTools(s, client)
	}
	return s
}

// registerReadTools 注册只读档工具（M1: newapi_status；M2 扩展其余）。
func registerReadTools(s *server.MCPServer, client *newapi.Client) {
	statusTool := mcp.NewTool("newapi_status",
		mcp.WithDescription("获取 new-api 网关状态：版本、启动时间、注册开关，并探测 relay (/v1/models) 活性。无需鉴权。"),
	)
	s.AddTool(statusTool, statusHandler(client))
}
