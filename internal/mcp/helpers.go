package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"mcp_newapi/internal/newapi"
)

// 工具层公共助手：所有工具 handler 都应是薄壳——
// 解析参数 → 调 newapi 域方法 → 统一输出。业务逻辑一律不下沉到本包。

type toolHandler = func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("结果编码失败: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

// errResult 把 newapi 错误转成 MCP 工具错误结果（保留可达性信息）。
func errResult(err error) (*mcp.CallToolResult, error) {
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
