package mcp

// tools_read.go read 档工具（8 个）。薄壳：参数解析 → 调 internal/newapi 对应域方法 → jsonResult 输出。

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"mcp_newapi/internal/newapi"
)

// statusHandler 处理 newapi_status。
func statusHandler(client *newapi.Client) toolHandler {
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
		return jsonResult(out)
	}
}

// modelsHandler 处理 newapi_list_models。
func modelsHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		m, err := client.Models(ctx)
		if err != nil {
			return errResult(err)
		}
		total := 0
		for _, l := range m {
			total += len(l)
		}
		return jsonResult(map[string]any{
			"groups":      m,
			"group_count": len(m),
			"model_count": total,
		})
	}
}

// channelsHandler 处理 newapi_list_channels。
func channelsHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		page := req.GetInt("page", 1)
		pageSize := req.GetInt("page_size", 20)
		status := req.GetInt("status", 0)
		res, err := client.ListChannels(ctx, page, pageSize, status)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	}
}

// channelDetailHandler 处理 newapi_get_channel。
func channelDetailHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("id")
		if err != nil {
			return errResult(err)
		}
		ch, err := client.GetChannel(ctx, id)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(ch)
	}
}

// tokensHandler 处理 newapi_list_tokens。
func tokensHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		page := req.GetInt("page", 1)
		pageSize := req.GetInt("page_size", 20)
		res, err := client.ListTokens(ctx, page, pageSize)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	}
}

// logsHandler 处理 newapi_logs。
func logsHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := newapi.LogQuery{
			Page:     orDefault(req, "page", 1),
			PageSize: orDefault(req, "page_size", 20),
			Type:     orDefault(req, "type", 0),
		}
		if v := req.GetInt("start_timestamp", 0); v > 0 {
			q.StartTimestamp = int64(v)
		}
		if v := req.GetInt("end_timestamp", 0); v > 0 {
			q.EndTimestamp = int64(v)
		}
		q.ModelName = req.GetString("model_name", "")
		q.TokenName = req.GetString("token_name", "")
		if v := req.GetInt("channel", 0); v > 0 {
			q.Channel = v
		}
		res, err := client.Logs(ctx, q)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	}
}

// usageSummaryHandler 处理 newapi_usage_summary：dashboard 按模型聚合 + 总量。
func usageSummaryHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		days := orDefault(req, "days", 7)
		if days <= 0 || days > 365 {
			days = 7
		}
		now := time.Now().Unix()
		start := now - int64(days)*86400
		usage, err := client.DashboardData(ctx, start, now)
		if err != nil {
			return errResult(err)
		}
		var totalQuota float64
		var totalCalls, totalTokens int
		for _, m := range usage {
			totalQuota += m.Quota
			totalCalls += m.Calls
			totalTokens += m.Tokens
		}
		out := map[string]any{
			"days":        days,
			"model_count": len(usage),
			"total": map[string]any{
				"calls":     totalCalls,
				"tokens":    totalTokens,
				"quota":     totalQuota,
				"quota_usd": totalQuota / newapi.QuotaPerUnit,
			},
			"by_model": usage,
		}
		return jsonResult(out)
	}
}

// pricingHandler 处理 newapi_pricing。
func pricingHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		model := req.GetString("model", "")
		raw, err := client.Pricing(ctx, model)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(json.RawMessage(raw))
	}
}

