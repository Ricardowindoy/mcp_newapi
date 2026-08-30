package mcp

// tools_admin.go admin 档工具 handler 实现（声明见 registry.go）。渠道 CRUD，需 writemode=admin。
// 安全约定：创建/更新传 key 是调用方显式行为；删除需 confirm=true。

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"mcp_newapi/internal/newapi"
)

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
