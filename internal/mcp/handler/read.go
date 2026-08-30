package handler

// read.go read 档工具 handler 实现（声明见 ../registry.go）。
// 薄壳：参数解析 → 调域函数 → JSONResult 输出。

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"mcp_newapi/internal/newapi"
	"mcp_newapi/internal/newapi/channels"
	"mcp_newapi/internal/newapi/logs"
	"mcp_newapi/internal/newapi/models"
	"mcp_newapi/internal/newapi/status"
	"mcp_newapi/internal/newapi/tokens"
)

// StatusHandler 处理 newapi_status。
func StatusHandler(client *newapi.Client) Handler {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		st, err := status.Get(ctx, client)
		if err != nil {
			return ErrResult(err)
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
		if err := status.RelayProbe(pctx, client); err != nil {
			out["relay_reachable"] = false
			out["relay_error"] = err.Error()
		} else {
			out["relay_reachable"] = true
		}
		return JSONResult(out)
	}
}

// ModelsHandler 处理 newapi_list_models。
func ModelsHandler(client *newapi.Client) Handler {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		m, err := models.List(ctx, client)
		if err != nil {
			return ErrResult(err)
		}
		total := 0
		for _, l := range m {
			total += len(l)
		}
		return JSONResult(map[string]any{
			"groups":      m,
			"group_count": len(m),
			"model_count": total,
		})
	}
}

// ChannelsHandler 处理 newapi_list_channels。
func ChannelsHandler(client *newapi.Client) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		page := req.GetInt("page", 1)
		pageSize := req.GetInt("page_size", 20)
		st := req.GetInt("status", 0)
		res, err := channels.List(ctx, client, page, pageSize, st)
		if err != nil {
			return ErrResult(err)
		}
		return JSONResult(res)
	}
}

// ChannelDetailHandler 处理 newapi_get_channel。
func ChannelDetailHandler(client *newapi.Client) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("id")
		if err != nil {
			return ErrResult(err)
		}
		ch, err := channels.Get(ctx, client, id)
		if err != nil {
			return ErrResult(err)
		}
		return JSONResult(ch)
	}
}

// TokensHandler 处理 newapi_list_tokens。
func TokensHandler(client *newapi.Client) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		page := req.GetInt("page", 1)
		pageSize := req.GetInt("page_size", 20)
		res, err := tokens.List(ctx, client, page, pageSize)
		if err != nil {
			return ErrResult(err)
		}
		return JSONResult(res)
	}
}

// LogsHandler 处理 newapi_logs。
func LogsHandler(client *newapi.Client) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := logs.Query{
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
		res, err := logs.Search(ctx, client, q)
		if err != nil {
			return ErrResult(err)
		}
		return JSONResult(res)
	}
}

// UsageSummaryHandler 处理 newapi_usage_summary：按模型聚合 + 总量。
func UsageSummaryHandler(client *newapi.Client) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		days := orDefault(req, "days", 7)
		if days <= 0 || days > 365 {
			days = 7
		}
		now := time.Now().Unix()
		start := now - int64(days)*86400
		usage, err := logs.AggregateByModel(ctx, client, start, now)
		if err != nil {
			return ErrResult(err)
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
		return JSONResult(out)
	}
}

// PricingHandler 处理 newapi_pricing。
func PricingHandler(client *newapi.Client) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		model := req.GetString("model", "")
		raw, err := models.Pricing(ctx, client, model)
		if err != nil {
			return ErrResult(err)
		}
		return JSONResult(json.RawMessage(raw))
	}
}
