package mcp

// tools_admin.go admin 档工具（3 个，渠道 CRUD）。需 NEWAPI_WRITEMODE=admin。
// 安全约定：创建/更新传 key 是调用方显式行为；删除需 confirm=true。

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp_newapi/internal/newapi"
)

// registerAdminTools 注册 admin 档工具。
func registerAdminTools(s *server.MCPServer, client *newapi.Client) {
	s.AddTool(mcp.NewTool("newapi_create_channel",
		mcp.WithDescription("创建渠道（admin）。需提供名称/类型/key/模型列表；创建后按名称回查返回渠道 id。key 只在此请求中传输。"),
		mcp.WithString("name", mcp.Required(), mcp.Description("渠道名称")),
		mcp.WithNumber("type", mcp.Required(), mcp.Description("渠道类型：1=OpenAI 兼容 等")),
		mcp.WithString("key", mcp.Required(), mcp.Description("上游 API key（仅在创建请求中传输）")),
		mcp.WithString("models", mcp.Required(), mcp.Description("模型列表，逗号分隔")),
		mcp.WithString("base_url", mcp.Description("上游 base URL（OpenAI 兼容网关填此项）")),
		mcp.WithString("group", mcp.Description("分组，逗号分隔，默认 default")),
		mcp.WithString("model_mapping", mcp.Description("模型重定向 JSON，如 {\"alias\":\"real\"}")),
		mcp.WithNumber("priority", mcp.Description("优先级")),
		mcp.WithNumber("weight", mcp.Description("权重")),
		mcp.WithString("test_model", mcp.Description("测试模型名")),
	), createChannelHandler(client))

	s.AddTool(mcp.NewTool("newapi_update_channel",
		mcp.WithDescription("更新渠道（admin，PATCH 语义：只传要改的字段）。注意：不能改 status（用 newapi_set_channel_status）；key 留空则不修改。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
		mcp.WithString("name", mcp.Description("新名称")),
		mcp.WithString("key", mcp.Description("新 key（留空不修改；多 key 渠道慎用）")),
		mcp.WithString("models", mcp.Description("模型列表，逗号分隔")),
		mcp.WithString("base_url", mcp.Description("上游 base URL")),
		mcp.WithString("group", mcp.Description("分组")),
		mcp.WithString("model_mapping", mcp.Description("模型重定向 JSON")),
		mcp.WithNumber("priority", mcp.Description("优先级")),
		mcp.WithNumber("weight", mcp.Description("权重")),
		mcp.WithString("test_model", mcp.Description("测试模型名")),
		mcp.WithNumber("type", mcp.Description("渠道类型")),
	), updateChannelHandler(client))

	s.AddTool(mcp.NewTool("newapi_delete_channel",
		mcp.WithDescription("删除渠道（admin，不可恢复）。必须显式传 confirm=true。"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
		mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("必须为 true 才执行删除")),
	), deleteChannelHandler(client))
}

func createChannelHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := strings.TrimSpace(req.GetString("name", ""))
		if name == "" {
			return mcp.NewToolResultError("name 不能为空"), nil
		}
		typ := req.GetInt("type", 0)
		key := req.GetString("key", "")
		models := req.GetString("models", "")
		if typ <= 0 || key == "" || models == "" {
			return mcp.NewToolResultError("type、key、models 均为必填"), nil
		}
		cr := newapi.ChannelUpsertReq{
			Name: name, Type: typ, Key: key, Models: models,
			BaseURL:      req.GetString("base_url", ""),
			Group:        orDefaultStr(req, "group", "default"),
			ModelMapping: req.GetString("model_mapping", ""),
			Priority:     req.GetInt("priority", 0),
			Weight:       req.GetInt("weight", 0),
			TestModel:    req.GetString("test_model", ""),
		}
		ch, err := client.CreateChannel(ctx, cr)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{
			"created":    true,
			"id":         ch.ID,
			"name":       ch.Name,
			"type":       ch.Type,
			"status":     ch.Status,
			"hint":       "创建成功；key 已设置（不回显），可用 newapi_test_channel 验证连通性",
		})
	}
}

func updateChannelHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetInt("id", 0)
		if id <= 0 {
			return mcp.NewToolResultError("id 必须为正整数"), nil
		}
		fields := map[string]any{}
		for _, k := range []string{"name", "key", "models", "base_url", "group", "model_mapping", "test_model", "type"} {
			if v := req.GetString(k, ""); v != "" {
				fields[k] = v
			}
		}
		for _, k := range []string{"priority", "weight"} {
			if v := req.GetInt(k, 0); v != 0 {
				fields[k] = v
			}
		}
		if len(fields) == 0 {
			return mcp.NewToolResultError("未提供任何要更新的字段"), nil
		}
		if err := client.UpdateChannelFields(ctx, id, fields); err != nil {
			return errResult(err)
		}
		ch, err := client.GetChannel(ctx, id)
		if err != nil {
			res, _ := jsonResult(map[string]any{"updated": true, "channel_id": id, "fields": keysOf(fields)})
			return res, nil
		}
		res, _ := jsonResult(map[string]any{
			"updated": true, "channel_id": id, "fields": keysOf(fields),
			"name": ch.Name, "models": ch.Models, "priority": ch.Priority, "group": ch.Group,
		})
		return res, nil
	}
}

func deleteChannelHandler(client *newapi.Client) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetInt("id", 0)
		confirm := req.GetBool("confirm", false)
		if id <= 0 {
			return mcp.NewToolResultError("id 必须为正整数"), nil
		}
		if !confirm {
			return mcp.NewToolResultError("删除不可恢复：请确认后显式传 confirm=true"), nil
		}
		if err := client.DeleteChannel(ctx, id); err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"deleted": true, "channel_id": id})
	}
}

func orDefaultStr(req mcp.CallToolRequest, key, def string) string {
	if v := req.GetString(key, ""); v != "" {
		return v
	}
	return def
}

func keysOf(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
