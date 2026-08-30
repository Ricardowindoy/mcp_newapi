package mcp

// tools_ops.go ops 档工具（6 个，写操作）。薄壳：参数解析 → 调域方法 → 输出。
// 需 NEWAPI_WRITEMODE=ops|admin 才注册。

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"mcp_newapi/internal/newapi"
)

func testChannelHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetInt("id", 0)
		if id <= 0 {
			return mcp.NewToolResultError("id 必须为正整数"), nil
		}
		model := req.GetString("model", "")
		tr, err := client.TestChannel(ctx, id, model)
		if err != nil {
			return errResult(err)
		}
		out := map[string]any{
			"channel_id":   id,
			"success":      tr.Success,
			"time_seconds": tr.TimeSec,
		}
		if tr.Message != "" {
			out["message"] = tr.Message
		}
		if tr.ErrorCode != "" {
			out["error_code"] = tr.ErrorCode
		}
		if model != "" {
			out["model"] = model
		}
		return jsonResult(out)
	}
}

func testAllHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info, err := client.TestAllChannels(ctx)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{
			"triggered": true,
			"task":      info,
			"hint":      "异步任务已入队；稍后用 newapi_list_channels 查看各渠道 response_time/status",
		})
	}
}

func updateBalanceHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetInt("id", 0)
		if id <= 0 {
			return mcp.NewToolResultError("id 必须为正整数"), nil
		}
		if err := client.UpdateChannelBalance(ctx, id); err != nil {
			return errResult(err)
		}
		ch, err := client.GetChannel(ctx, id)
		if err != nil {
			res, _ := jsonResult(map[string]any{"channel_id": id, "refreshed": true})
			return res, nil
		}
		return jsonResult(map[string]any{
			"channel_id": id,
			"name":       ch.Name,
			"balance":    ch.Balance,
		})
	}
}

func setChannelStatusHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetInt("id", 0)
		enabled := req.GetBool("enabled", false)
		if id <= 0 {
			return mcp.NewToolResultError("id 必须为正整数"), nil
		}
		changed, err := client.SetChannelStatus(ctx, id, enabled)
		if err != nil {
			return errResult(err)
		}
		action := "禁用"
		if enabled {
			action = "启用"
		}
		return jsonResult(map[string]any{
			"channel_id": id,
			"enabled":    enabled,
			"changed":    changed,
			"note":       fmt.Sprintf("已请求%s渠道 %d（%s）", action, id, map[bool]string{true: "有实际变更", false: "状态未变化"}[changed]),
		})
	}
}

func createTokenHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := strings.TrimSpace(req.GetString("name", ""))
		if name == "" {
			return mcp.NewToolResultError("name 不能为空"), nil
		}
		if len(name) > 50 {
			return mcp.NewToolResultError("name 长度不能超过 50"), nil
		}
		t := newapi.TokenCreateReq{
			Name:               name,
			UnlimitedQuota:     req.GetBool("unlimited_quota", false),
			RemainQuota:        int64(req.GetInt("remain_quota", 0)),
			ExpiredTime:        int64(req.GetInt("expired_time", -1)),
			ModelLimits:        req.GetString("model_limits", ""),
			ModelLimitsEnabled: req.GetString("model_limits", "") != "",
			Group:              req.GetString("group", ""),
		}
		if !t.UnlimitedQuota && t.RemainQuota <= 0 {
			// 不给额度默认给无限，避免创建出 0 额度不可用令牌
			t.UnlimitedQuota = true
		}
		if t.ModelLimits != "" && !t.ModelLimitsEnabled {
			t.ModelLimitsEnabled = true
		}
		ts, err := client.CreateToken(ctx, t)
		if err != nil {
			return errResult(err)
		}
		ts2 := map[string]any{
			"id":               ts.ID,
			"name":             ts.Name,
			"key_masked":       ts.Key,
			"unlimited_quota":  t.UnlimitedQuota,
			"note":             "完整 sk- key 请在面板「令牌」页查看（本 MCP 不透出完整 key）",
		}
		if !t.UnlimitedQuota {
			ts2["remain_quota"] = t.RemainQuota
			ts2["remain_quota_usd"] = float64(t.RemainQuota) / newapi.QuotaPerUnit
		}
		return jsonResult(ts2)
	}
}

func deleteTokenHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetInt("id", 0)
		confirm := req.GetBool("confirm", false)
		if id <= 0 {
			return mcp.NewToolResultError("id 必须为正整数"), nil
		}
		if !confirm {
			return mcp.NewToolResultError("删除不可恢复：请确认后显式传 confirm=true"), nil
		}
		if err := client.DeleteToken(ctx, id); err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"deleted": true, "token_id": id})
	}
}
