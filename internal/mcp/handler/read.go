package handler

// read.go read 档工具 handler 实现（声明见 ../registry.go）。
// 薄壳：参数解析 → 调域函数 → JSONResult 输出。

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"mcp_newapi/internal/newapi"
	"mcp_newapi/internal/newapi/channels"
	"mcp_newapi/internal/newapi/logs"
	"mcp_newapi/internal/newapi/models"
	"mcp_newapi/internal/newapi/options"
	"mcp_newapi/internal/reporter"
	"mcp_newapi/internal/newapi/status"
	"mcp_newapi/internal/newapi/tokens"
)

// StatusHandler 处理 newapi_status。
func StatusHandler(client *newapi.Client, rep *reporter.Store) Handler {
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
func ModelsHandler(client *newapi.Client, rep *reporter.Store) Handler {
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
func ChannelsHandler(client *newapi.Client, rep *reporter.Store) Handler {
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
func ChannelDetailHandler(client *newapi.Client, rep *reporter.Store) Handler {
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
func TokensHandler(client *newapi.Client, rep *reporter.Store) Handler {
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
func LogsHandler(client *newapi.Client, rep *reporter.Store) Handler {
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
func UsageSummaryHandler(client *newapi.Client, rep *reporter.Store) Handler {
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
func PricingHandler(client *newapi.Client, rep *reporter.Store) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		model := req.GetString("model", "")
		raw, err := models.Pricing(ctx, client, model)
		if err != nil {
			return ErrResult(err)
		}
		return JSONResult(json.RawMessage(raw))
	}
}

// ListOptionsHandler 处理 newapi_list_options。
func ListOptionsHandler(client *newapi.Client, rep *reporter.Store) Handler {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entries, err := options.List(ctx, client)
		if err != nil {
			return ErrResult(err)
		}
		return JSONResult(map[string]any{
			"count":   len(entries),
			"options": entries,
		})
	}
}

// SuccessRateHandler 处理 newapi_success_rate。
func SuccessRateHandler(client *newapi.Client, rep *reporter.Store) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		startTs := req.GetInt("start_timestamp", 0)
		endTs := req.GetInt("end_timestamp", 0)
		hours := req.GetInt("hours", 24)
		if hours <= 0 || hours > 720 {
			hours = 24
		}
		if startTs <= 0 || endTs <= 0 {
			endTs = int(time.Now().Unix())
			startTs = endTs - hours*3600
		}
		q := logs.Query{
			StartTimestamp: int64(startTs),
			EndTimestamp:   int64(endTs),
			Channel:        req.GetInt("channel", 0),
			ModelName:      req.GetString("model_name", ""),
			TokenName:      req.GetString("token_name", ""),
		}
		successQ, errQ := q, q
		successQ.Type, errQ.Type = 2, 5
		success, err := logs.Count(ctx, client, successQ)
		if err != nil {
			return ErrResult(err)
		}
		fail, err := logs.Count(ctx, client, errQ)
		if err != nil {
			return ErrResult(err)
		}
		total := success + fail
		out := map[string]any{
			"window":        map[string]any{"start": startTs, "end": endTs},
			"success_count": success,
			"error_count":   fail,
			"total":         total,
			"filters": map[string]any{
				"channel":    q.Channel,
				"model_name": q.ModelName,
				"token_name": q.TokenName,
			},
			"note": "基于日志条目计数（type=2 消费 vs type=5 错误）；上游重试会产生多条错误日志，比率为近似值",
		}
		if total == 0 {
			out["success_rate"] = nil
			out["note"] = "时间窗内无匹配日志"
		} else {
			rate := float64(success) / float64(total)
			out["success_rate"] = math.Round(rate*10000) / 10000
			out["success_rate_pct"] = fmt.Sprintf("%.2f%%", rate*100)
		}
		return JSONResult(out)
	}
}

// ==================== autoban 配置总览与封禁原因分析（read 档） ====================
// 编排放 handler 层（跨 channels/logs/options 三域组合，与 success_rate 同先例）；域间互不依赖。

// isAutobanOptionKey 判断 option 键是否属于自动封禁/渠道监控生态（实测键集见上游 option 列表）。
func isAutobanOptionKey(key string) bool {
	lk := strings.ToLower(key)
	return strings.Contains(lk, "automatic") ||
		lk == "channeldisablethreshold" ||
		lk == "retrytimes" ||
		strings.HasPrefix(lk, "monitor_setting.") ||
		strings.HasPrefix(lk, "channel_affinity_setting.")
}

// allChannels 分页拉全量渠道（pageSize=100 循环，防上游单页上限截断）。
func allChannels(ctx context.Context, client *newapi.Client) ([]channels.Summary, error) {
	var out []channels.Summary
	for page := 1; page <= 100; page++ {
		res, err := channels.List(ctx, client, page, 100, 0)
		if err != nil {
			return nil, err
		}
		out = append(out, res.Items...)
		if len(res.Items) == 0 || len(out) >= res.Total {
			break
		}
	}
	return out, nil
}

// AutobanConfigHandler 处理 newapi_autoban_config（只读总览）。
func AutobanConfigHandler(client *newapi.Client, rep *reporter.Store) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entries, err := options.List(ctx, client)
		if err != nil {
			return ErrResult(err)
		}
		sel := make([]options.Entry, 0, 16)
		switchEnabled := false
		for _, e := range entries {
			if !isAutobanOptionKey(e.Key) {
				continue
			}
			sel = append(sel, e)
			if e.Key == "AutomaticDisableChannelEnabled" {
				switchEnabled = strings.EqualFold(strings.TrimSpace(e.Value), "true")
			}
		}
		chs, err := allChannels(ctx, client)
		if err != nil {
			return ErrResult(err)
		}
		on, off, unset := 0, 0, 0
		notEnabled := make([]map[string]any, 0)
		for _, ch := range chs {
			switch {
			case ch.AutoBan == nil:
				unset++
				notEnabled = append(notEnabled, map[string]any{"id": ch.ID, "name": ch.Name, "auto_ban": nil, "reason": "未设置（上游 NULL 视为关）"})
			case *ch.AutoBan == 1:
				on++
			default:
				off++
				notEnabled = append(notEnabled, map[string]any{"id": ch.ID, "name": ch.Name, "auto_ban": *ch.AutoBan, "reason": "显式关闭"})
			}
		}
		return JSONResult(map[string]any{
			"global_switch_enabled": switchEnabled,
			"options":               sel,
			"channels_auto_ban": map[string]any{
				"total": len(chs), "on": on, "off": off, "unset": unset,
				"not_enabled": notEnabled,
			},
			"note": "渠道级 auto_ban 为 NULL/0 时该渠道不参与自动禁用（『ban 不生效』常见根因，gorm default 只覆盖新建行）；写入口：全局开关与关键词→newapi_update_option，状态码→newapi_autoban_codes，渠道级→newapi_update_channel(auto_ban)",
		})
	}
}

// contentAgg 是错误内容聚合计数。
type contentAgg struct {
	count int
	last  int64
}

// errStats 聚合错误采样：按错误内容 topN、按模型 topN、最近错误时间。
func errStats(entries []logs.Entry, topN int) (byContent, byModel []map[string]any, lastAt int64) {
	cm := map[string]*contentAgg{}
	mm := map[string]int{}
	for _, e := range entries {
		content := e.Content
		if content == "" {
			content = "(空错误信息)"
		}
		a := cm[content]
		if a == nil {
			a = &contentAgg{}
			cm[content] = a
		}
		a.count++
		if e.CreatedAt > a.last {
			a.last = e.CreatedAt
		}
		if e.ModelName != "" {
			mm[e.ModelName]++
		}
		if e.CreatedAt > lastAt {
			lastAt = e.CreatedAt
		}
	}
	type ckv struct {
		k string
		v *contentAgg
	}
	cl := make([]ckv, 0, len(cm))
	for k, v := range cm {
		cl = append(cl, ckv{k, v})
	}
	sort.Slice(cl, func(i, j int) bool {
		if cl[i].v.count != cl[j].v.count {
			return cl[i].v.count > cl[j].v.count
		}
		return cl[i].k < cl[j].k
	})
	if len(cl) > topN {
		cl = cl[:topN]
	}
	byContent = make([]map[string]any, 0, len(cl))
	for _, e := range cl {
		byContent = append(byContent, map[string]any{"content": e.k, "count": e.v.count, "last_seen": e.v.last})
	}
	type mkv struct {
		k string
		v int
	}
	ml := make([]mkv, 0, len(mm))
	for k, v := range mm {
		ml = append(ml, mkv{k, v})
	}
	sort.Slice(ml, func(i, j int) bool {
		if ml[i].v != ml[j].v {
			return ml[i].v > ml[j].v
		}
		return ml[i].k < ml[j].k
	})
	if len(ml) > topN {
		ml = ml[:topN]
	}
	byModel = make([]map[string]any, 0, len(ml))
	for _, e := range ml {
		byModel = append(byModel, map[string]any{"model": e.k, "count": e.v})
	}
	return byContent, byModel, lastAt
}

// classifyBanCause 封禁原因启发式分类（关键词首中即返回，按优先级排序）。
func classifyBanCause(contents []string) string {
	joined := strings.ToLower(strings.Join(contents, "\n"))
	for _, c := range []struct {
		cause string
		keys  []string
	}{
		{"quota_exhausted", []string{"402", "余额", "欠费", "quota", "insufficient", "credit balance"}},
		{"model_issue", []string{"模型", "无可用渠道", "no available channel", "模型已关闭", "does not exist"}},
		{"timeout", []string{"timeout", "超时", "deadline exceeded", "context canceled"}},
		{"upstream_unreachable", []string{"connection refused", "connect:", "dial", "no such host", "unreachable"}},
	} {
		for _, k := range c.keys {
			if strings.Contains(joined, k) {
				return c.cause
			}
		}
	}
	return "other"
}

// AutobanAnalysisHandler 处理 newapi_autoban_analysis（自动封禁原因分析数据获取）。
func AutobanAnalysisHandler(client *newapi.Client, rep *reporter.Store) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		channelID := req.GetInt("channel", 0)
		hours := req.GetInt("hours", 24)
		if hours <= 0 || hours > 720 {
			hours = 24
		}
		sample := req.GetInt("sample", 10)
		if sample <= 0 || sample > 50 {
			sample = 10
		}
		now := time.Now().Unix()
		start := now - int64(hours)*3600

		var targets []channels.Summary
		if channelID > 0 {
			ch, err := channels.Get(ctx, client, channelID)
			if err != nil {
				return ErrResult(err)
			}
			targets = []channels.Summary{*ch}
		} else {
			all, err := allChannels(ctx, client)
			if err != nil {
				return ErrResult(err)
			}
			for _, ch := range all {
				if ch.Status == 3 || ch.StatusReason != "" {
					targets = append(targets, ch)
				}
			}
		}
		if len(targets) == 0 {
			return JSONResult(map[string]any{
				"targets_analyzed": 0,
				"note":             "当前无自动禁用（status=3）或带 status_reason 的渠道；可用 channel 参数指定任意渠道分析",
			})
		}

		out := make([]map[string]any, 0, len(targets))
		for _, ch := range targets {
			view := map[string]any{
				"id": ch.ID, "name": ch.Name, "status": ch.Status,
				"status_reason": ch.StatusReason, "balance": ch.Balance,
				"auto_ban": ch.AutoBan, "models": ch.Models, "test_model": ch.TestModel,
			}
			q := logs.Query{Type: 5, Channel: ch.ID, StartTimestamp: start, EndTimestamp: now}
			total, err := logs.Count(ctx, client, q)
			if err != nil {
				view["error"] = "日志统计失败: " + err.Error()
				out = append(out, view)
				continue
			}
			view["errors_total"] = total
			var entries []logs.Entry
			if total > 0 {
				sq := q
				sq.Page, sq.PageSize = 1, sample
				if res, serr := logs.Search(ctx, client, sq); serr == nil {
					entries = res.Items
				}
			}
			contents := make([]string, 0, len(entries))
			for _, e := range entries {
				contents = append(contents, e.Content)
			}
			byContent, byModel, lastAt := errStats(entries, 5)
			view["last_error_at"] = lastAt
			view["by_content"] = byContent
			view["by_model"] = byModel
			if total == 0 {
				view["likely_cause"] = "no_error_logs"
			} else {
				view["likely_cause"] = classifyBanCause(contents)
			}
			out = append(out, view)
		}
		return JSONResult(map[string]any{
			"window":           map[string]any{"hours": hours, "start": start, "end": now},
			"targets_analyzed": len(targets),
			"channels":         out,
			"note":             "likely_cause 为启发式分类（quota/model/timeout/unreachable 关键词首中，no_error_logs=窗口内无错误），以 by_content 采样明细为准；当前 autoban 配置用 newapi_autoban_config 查看",
		})
	}
}
