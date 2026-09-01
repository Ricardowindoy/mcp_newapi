package handler

// admin.go admin 档工具 handler 实现（声明见 ../registry.go）。渠道 CRUD，需 writemode=admin。
// 安全约定：创建/更新传 key 是调用方显式行为；删除需 confirm=true。

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"mcp_newapi/internal/newapi"
	"mcp_newapi/internal/newapi/channels"
	"mcp_newapi/internal/newapi/options"
	"mcp_newapi/internal/reporter"
)

// CreateChannelHandler 处理 newapi_create_channel。
func CreateChannelHandler(client *newapi.Client, rep *reporter.Store) Handler {
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
		cr := channels.UpsertReq{
			Name: name, Type: typ, Key: key, Models: models,
			BaseURL:      req.GetString("base_url", ""),
			Group:        orDefaultStr(req, "group", "default"),
			ModelMapping: req.GetString("model_mapping", ""),
			Priority:     req.GetInt("priority", 0),
			Weight:       req.GetInt("weight", 0),
			TestModel:    req.GetString("test_model", ""),
		}
		ch, err := channels.Create(ctx, client, cr)
		if err != nil {
			return ErrResult(err)
		}
		return JSONResult(map[string]any{
			"created":       true,
			"id":            ch.ID,
			"name":          ch.Name,
			"type":          ch.Type,
			"status":        ch.Status,
			"model_mapping": ch.ModelMapping,
			"hint":          "创建成功；key 已设置（不回显），可用 newapi_test_channel 验证连通性",
		})
	}
}

// UpdateChannelHandler 处理 newapi_update_channel。
func UpdateChannelHandler(client *newapi.Client, rep *reporter.Store) Handler {
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
		// auto_ban 布尔三态：按「是否出现」判断，出现即写 1/0（上游 AutoBan *int，nil 视为 false）
		if _, present := req.GetArguments()["auto_ban"]; present {
			v := 0
			if req.GetBool("auto_ban", false) {
				v = 1
			}
			fields["auto_ban"] = v
		}
		// tag 三态：出现即写（空串=清除 tag；上游 PatchChannel 内嵌 model.Channel.Tag）
		if _, present := req.GetArguments()["tag"]; present {
			fields["tag"] = req.GetString("tag", "")
		}
		if len(fields) == 0 {
			return mcp.NewToolResultError("未提供任何要更新的字段"), nil
		}
		if err := channels.UpdateFields(ctx, client, id, fields); err != nil {
			return ErrResult(err)
		}
		ch, err := channels.Get(ctx, client, id)
		if err != nil {
			res, _ := JSONResult(map[string]any{"updated": true, "channel_id": id, "fields": keysOf(fields)})
			return res, nil
		}
		res, _ := JSONResult(map[string]any{
			"updated": true, "channel_id": id, "fields": keysOf(fields),
			"name": ch.Name, "models": ch.Models, "priority": ch.Priority, "group": ch.Group,
			// model_mapping 显式输出（含改后为空串=已清空），便于核对映射变更
			"model_mapping": ch.ModelMapping,
		})
		return res, nil
	}
}

// DeleteChannelHandler 处理 newapi_delete_channel。
func DeleteChannelHandler(client *newapi.Client, rep *reporter.Store) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetInt("id", 0)
		confirm := req.GetBool("confirm", false)
		if id <= 0 {
			return mcp.NewToolResultError("id 必须为正整数"), nil
		}
		if !confirm {
			return mcp.NewToolResultError("删除不可恢复：请确认后显式传 confirm=true"), nil
		}
		if err := channels.Delete(ctx, client, id); err != nil {
			return ErrResult(err)
		}
		return JSONResult(map[string]any{"deleted": true, "channel_id": id})
	}
}

// UpdateOptionHandler 处理 newapi_update_option（系统设置，全局生效，危险操作）。
func UpdateOptionHandler(client *newapi.Client, rep *reporter.Store) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key := strings.TrimSpace(req.GetString("key", ""))
		value := req.GetString("value", "")
		confirm := req.GetBool("confirm", false)
		if key == "" {
			return mcp.NewToolResultError("key 不能为空；先用 newapi_list_options 查现有键名"), nil
		}
		if value == "" {
			return mcp.NewToolResultError("value 不能为空（布尔传 \"true\"/\"false\"，数字传字符串）"), nil
		}
		if !confirm {
			return mcp.NewToolResultError("系统设置全局生效：请确认后显式传 confirm=true"), nil
		}
		if err := options.Update(ctx, client, key, value); err != nil {
			return ErrResult(err)
		}
		return JSONResult(map[string]any{
			"updated": true,
			"key":     key,
			"value":   value,
			"hint":    "部分设置需重启网关或等待下一轮任务才生效；可用 newapi_list_options 回查",
		})
	}
}

// AutobanCodesHandler 处理 newapi_autoban_codes（状态码增删查改，写上游 option）。
func AutobanCodesHandler(client *newapi.Client, rep *reporter.Store) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action := req.GetString("action", "list")
		target := req.GetString("target", "disable")
		switch action {
		case "list":
			res, err := options.StatusCodesList(ctx, client, target)
			if err != nil {
				return ErrResult(err)
			}
			return JSONResult(res)
		case "add", "remove", "set":
		default:
			return mcp.NewToolResultError("action 仅支持 list|add|remove|set"), nil
		}
		if target != "disable" && target != "retry" {
			return mcp.NewToolResultError("target 仅支持 disable|retry"), nil
		}
		confirm := req.GetBool("confirm", false)
		if !confirm {
			return mcp.NewToolResultError("修改 autoban 状态码全局生效：请确认后显式传 confirm=true"), nil
		}
		codes := req.GetString("codes", "")
		if strings.TrimSpace(codes) == "" {
			return mcp.NewToolResultError("codes 不能为空（单码或区间，逗号分隔，如 402,400-499）"), nil
		}
		res, err := options.StatusCodesModify(ctx, client, target, action, strings.Split(codes, ","))
		if err != nil {
			return ErrResult(err)
		}
		return JSONResult(res)
	}
}

// TagChannelsHandler 处理 newapi_tag_channels（按标签批量编辑/启停渠道）。
func TagChannelsHandler(client *newapi.Client, rep *reporter.Store) Handler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action := req.GetString("action", "")
		if action != "edit" && action != "enable" && action != "disable" {
			return mcp.NewToolResultError("action 仅支持 edit|enable|disable"), nil
		}
		tag := strings.TrimSpace(req.GetString("tag", ""))
		if tag == "" {
			return mcp.NewToolResultError("tag 不能为空"), nil
		}
		confirm := req.GetBool("confirm", false)
		if !confirm {
			return mcp.NewToolResultError("批量操作影响该 tag 下所有渠道：请确认后显式传 confirm=true"), nil
		}
		switch action {
		case "enable", "disable":
			data, err := channels.SetTagStatus(ctx, client, tag, action == "enable")
			if err != nil {
				return ErrResult(err)
			}
			return JSONResult(map[string]any{
				"action": action, "tag": tag, "updated": true, "data": json.RawMessage(data),
			})
		}
		tr := channels.TagReq{Tag: tag}
		args := req.GetArguments()
		n := 0
		if v, ok := args["new_tag"]; ok {
			s, _ := v.(string)
			s = strings.TrimSpace(s)
			if s == "" {
				return mcp.NewToolResultError("new_tag 不能为空（清空单个渠道 tag 请用 newapi_update_channel 传 tag=\"\"）"), nil
			}
			tr.NewTag = &s
			n++
		}
		if _, ok := args["priority"]; ok {
			p := int64(req.GetInt("priority", 0))
			tr.Priority = &p
			n++
		}
		if _, ok := args["weight"]; ok {
			w := req.GetInt("weight", 0)
			if w < 0 {
				return mcp.NewToolResultError("weight 不能为负"), nil
			}
			wu := uint(w)
			tr.Weight = &wu
			n++
		}
		if v, ok := args["models"]; ok {
			s, _ := v.(string)
			tr.Models = &s
			n++
		}
		if v, ok := args["model_mapping"]; ok {
			s, _ := v.(string)
			tr.ModelMapping = &s
			n++
		}
		if v, ok := args["group"]; ok {
			s, _ := v.(string)
			tr.Groups = &s
			n++
		}
		if n == 0 {
			return mcp.NewToolResultError("edit 未提供任何要编辑的字段（new_tag/priority/weight/models/model_mapping/group）"), nil
		}
		data, err := channels.EditByTag(ctx, client, tr)
		if err != nil {
			return ErrResult(err)
		}
		return JSONResult(map[string]any{
			"action": action, "tag": tag, "updated": true, "fields": n, "data": json.RawMessage(data),
		})
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
