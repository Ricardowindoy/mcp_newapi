package handler

// report.go 基元消费报表 handler（工具在 toolRegistry 表内，read 档）。
// rep 为 nil（未配置报表库）时工具仍注册，调用时返回配置不足错误。

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"mcp_newapi/internal/newapi"
	"mcp_newapi/internal/reporter"
)

// JiyuanReportHandler 处理 newapi_jiyuan_report：聚合报表从库，返回结构化消费报表。
func JiyuanReportHandler(client *newapi.Client, rep *reporter.Store) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if rep == nil {
			return mcp.NewToolResultError("报表功能未配置：需要在生产配置 [report] 段设置 db_dsn_file（或 db_dsn / 环境变量 NEWAPI_REPORT_DB_DSN）后重启 MCP"), nil
		}
		nameLike := strings.TrimSpace(req.GetString("name_like", ""))
		if nameLike == "" {
			nameLike = "基元"
		}
		days := req.GetInt("days", 4)
		if days <= 0 || days > 30 {
			days = 4
		}
		res, err := rep.JiyuanReport(ctx, nameLike, days)
		if err != nil {
			return ErrResult(err)
		}
		return JSONResult(res)
	}
}
