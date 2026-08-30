package mcp

// tools_ops.go ops 档工具（6 个，写操作）。薄壳：参数解析 → 调域方法 → 输出。
// 需 NEWAPI_WRITEMODE=ops|admin 才注册。

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp_newapi/internal/newapi"
)

// registerOpsTools 注册 ops 档工具（写操作，需 NEWAPI_WRITEMODE=ops/admin）。
func registerOpsTools(s *server.MCPServer, client *newapi.Client) {
	s.AddTool(mcp.NewTool("newapi_test_channel",
		mcp.WithDescription("对单个渠道发一次测试请求，返回是否成功、耗时（秒）与错误信息。测试失败是有效结果（success:false），不代表调用出错。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
		mcp.WithString("model", mcp.Description("测试用的模型名；缺省用渠道默认测试模型")),
	), testChannelHandler(client))

	s.AddTool(mcp.NewTool("newapi_test_all_channels",
		mcp.WithDescription("触发全量渠道测试（异步系统任务），返回 task_id。结果可在面板任务中心或稍后用 newapi_list_channels 观察响应时间/状态。"),
	), testAllHandler(client))

	s.AddTool(mcp.NewTool("newapi_update_channel_balance",
		mcp.WithDescription("刷新单个渠道的余额（部分渠道类型不支持，会返回业务错误）。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
	), updateBalanceHandler(client))

	s.AddTool(mcp.NewTool("newapi_set_channel_status",
		mcp.WithDescription("启用或禁用渠道（记录为 manual operation）。返回是否有实际变更。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
		mcp.WithBoolean("enabled", mcp.Required(), mcp.Description("true=启用 false=禁用")),
	), setChannelStatusHandler(client))

	s.AddTool(mcp.NewTool("newapi_create_token",
		mcp.WithDescription("创建一个 API 令牌（sk- key）。返回 id 与掩码 key；完整 key 请在面板查看。"),
		mcp.WithString("name", mcp.Required(), mcp.Description("令牌名称（≤50 字符）")),
		mcp.WithBoolean("unlimited_quota", mcp.Description("无限额度，默认 false")),
		mcp.WithNumber("remain_quota", mcp.Description("额度（quota 单位，500000=$1）；unlimited 时忽略")),
		mcp.WithNumber("expired_time", mcp.Description("过期 Unix 时间戳（秒）；-1 或缺省=永不过期")),
		mcp.WithString("model_limits", mcp.Description("模型限制，逗号分隔模型名")),
		mcp.WithString("group", mcp.Description("分组，缺省 default")),
	), createTokenHandler(client))

	s.AddTool(mcp.NewTool("newapi_delete_token",
		mcp.WithDescription("删除当前用户的 API 令牌（不可恢复）。必须显式传 confirm=true。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("令牌 ID")),
		mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("必须为 true 才执行删除")),
	), deleteTokenHandler(client))
}

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
