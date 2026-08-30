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
		registerAdminTools(s, client)
		fallthrough
	case ModeOps:
		registerOpsTools(s, client)
	}
	return s
}

// registerReadTools 注册只读档工具（M1+M2 共 8 个）。
func registerReadTools(s *server.MCPServer, client *newapi.Client) {
	s.AddTool(mcp.NewTool("newapi_status",
		mcp.WithDescription("获取 new-api 网关状态：版本、启动时间、注册开关，并探测 relay (/v1/models) 活性。无需鉴权。"),
	), statusHandler(client))

	s.AddTool(mcp.NewTool("newapi_list_models",
		mcp.WithDescription("获取网关全站可用模型列表（按分组返回）。无需鉴权。"),
	), modelsHandler(client))

	s.AddTool(mcp.NewTool("newapi_list_channels",
		mcp.WithDescription("分页获取渠道列表（需管理员 PAT）：id、名称、类型、状态、余额、模型、分组、响应延迟。key 已掩码。"),
		mcp.WithNumber("page", mcp.Description("页码，从 1 开始"), mcp.DefaultNumber(1)),
		mcp.WithNumber("page_size", mcp.Description("每页条数"), mcp.DefaultNumber(20)),
		mcp.WithNumber("status", mcp.Description("状态过滤：0=全部 1=启用 2=禁用"), mcp.DefaultNumber(0)),
	), channelsHandler(client))

	s.AddTool(mcp.NewTool("newapi_get_channel",
		mcp.WithDescription("获取单个渠道详情（需管理员 PAT）。key 已掩码。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
	), channelDetailHandler(client))

	s.AddTool(mcp.NewTool("newapi_list_tokens",
		mcp.WithDescription("获取当前用户的令牌（sk- key）列表：名称、状态、用量、剩余额度。key 已掩码。"),
		mcp.WithNumber("page", mcp.Description("页码，从 1 开始"), mcp.DefaultNumber(1)),
		mcp.WithNumber("page_size", mcp.Description("每页条数"), mcp.DefaultNumber(20)),
	), tokensHandler(client))

	s.AddTool(mcp.NewTool("newapi_logs",
		mcp.WithDescription("检索网关日志（消费/错误等）。type: 2=消费 5=错误；不传=全部。"),
		mcp.WithNumber("page", mcp.Description("页码，从 1 开始"), mcp.DefaultNumber(1)),
		mcp.WithNumber("page_size", mcp.Description("每页条数"), mcp.DefaultNumber(20)),
		mcp.WithNumber("type", mcp.Description("日志类型：0=全部 2=消费 5=错误")),
		mcp.WithNumber("start_timestamp", mcp.Description("起始 Unix 时间戳（秒）")),
		mcp.WithNumber("end_timestamp", mcp.Description("结束 Unix 时间戳（秒）")),
		mcp.WithString("model_name", mcp.Description("按模型过滤")),
		mcp.WithString("token_name", mcp.Description("按令牌名过滤")),
		mcp.WithNumber("channel", mcp.Description("按渠道 ID 过滤")),
	), logsHandler(client))

	s.AddTool(mcp.NewTool("newapi_usage_summary",
		mcp.WithDescription("近 N 天用量汇总：按模型聚合的调用次数/tokens/消费额（quota 及美元，500000 quota=$1）。"),
		mcp.WithNumber("days", mcp.Description("统计天数，默认 7，最大 365"), mcp.DefaultNumber(7)),
	), usageSummaryHandler(client))

	s.AddTool(mcp.NewTool("newapi_pricing",
		mcp.WithDescription("获取模型倍率定价（注意：实例可能禁用此端点，报错即禁用）。"),
		mcp.WithString("model", mcp.Description("按模型名过滤")),
	), pricingHandler(client))
}
