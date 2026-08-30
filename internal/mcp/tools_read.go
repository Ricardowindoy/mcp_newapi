package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"mcp_newapi/internal/newapi"
)

// statusHandler 处理 newapi_status。
func statusHandler(client *newapi.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		st, err := client.Status(ctx)
		if err != nil {
			return errResult(err)
		}
		out := map[string]any{
			"version":          st.Version,
			"start_time":       st.StartTime,
			"uptime_seconds":   time.Now().Unix() - st.StartTime,
			"register_enabled": st.RegisterEnabled,
		}
		// relay 探测：失败不致命，只标注
		pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := client.RelayProbe(pctx); err != nil {
			out["relay_reachable"] = false
			out["relay_error"] = err.Error()
		} else {
			out["relay_reachable"] = true
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	}
}

// errResult 把 newapi 错误转成 MCP 工具错误结果（保留可达性信息）。
func errResult(err error) (*mcp.CallToolResult, error) {
	var apiErr *newapi.APIError
	if errors.As(err, &apiErr) {
		return mcp.NewToolResultError(fmt.Sprintf("[reachable=%v status=%d] %s", apiErr.Reachable, apiErr.StatusCode, apiErr.Message)), nil
	}
	return mcp.NewToolResultError(err.Error()), nil
}
