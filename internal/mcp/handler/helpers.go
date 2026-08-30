package handler

// Package handler 是 MCP 工具层：所有 handler 实现。
// 薄壳约定：解析参数 → 调 internal/newapi 域函数 → 统一输出。业务逻辑一律不下沉到本包。
// 工具声明（name/tier/参数）见 ../registry.go 汇总表。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"mcp_newapi/internal/newapi"
)

// Handler 是工具处理函数签名。
type Handler = func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)

func JSONResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("结果编码失败: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

// ErrResult 把 newapi 错误转成 MCP 工具错误结果（保留可达性信息）。
func ErrResult(err error) (*mcp.CallToolResult, error) {
	var apiErr *newapi.APIError
	if errors.As(err, &apiErr) {
		return mcp.NewToolResultError(fmt.Sprintf("[reachable=%v status=%d] %s", apiErr.Reachable, apiErr.StatusCode, apiErr.Message)), nil
	}
	return mcp.NewToolResultError(err.Error()), nil
}

func orDefault(req mcp.CallToolRequest, key string, def int) int {
	if v := req.GetInt(key, def); v > 0 {
		return v
	}
	return def
}
