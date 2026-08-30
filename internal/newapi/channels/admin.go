package channels

// admin.go 渠道域·管理操作（ChannelSensitiveWrite，admin 档）。
// 上游契约（controller/channel.go）：
//   AddChannel:    POST /api/channel/   body {mode, channel:{...}}，channel 需带 type/key/models
//   UpdateChannel: PUT  /api/channel/   PATCH 语义（只覆盖出现的字段）；
//                  status 字段被拒（启停走 ops.go 的 SetStatus）；
//                  created_time/test_time/response_time/balance 等只读字段若传入会被清零
//   DeleteChannel: DELETE /api/channel/:id
// 敏感约束：完整 key 只在创建/更新时由调用方显式传入，任何响应不回传完整 key。

import (
	"context"
	"encoding/json"
	"fmt"

	"mcp_newapi/internal/newapi"
)

// UpsertReq 是创建渠道的请求体。
type UpsertReq struct {
	Name               string `json:"name"`
	Type               int    `json:"type"` // 渠道类型，如 1=OpenAI
	Key                string `json:"key"`
	BaseURL            string `json:"base_url,omitempty"`
	Models             string `json:"models"` // 逗号分隔
	Group              string `json:"group"`  // 逗号分隔，default
	ModelMapping       string `json:"model_mapping,omitempty"`
	Priority           int    `json:"priority,omitempty"`
	Weight             int    `json:"weight,omitempty"`
	TestModel          string `json:"test_model,omitempty"`
	OpenAIOrganization string `json:"openai_organization,omitempty"`
	Other              string `json:"other,omitempty"`
}

// Create 创建渠道。上游不回传 id：按名称搜回。
func Create(ctx context.Context, c *newapi.Client, req UpsertReq) (*Summary, error) {
	body := map[string]any{
		"mode":    "single",
		"channel": req,
	}
	if _, err := c.Do(ctx, "POST", newapi.RouteChannels, nil, body); err != nil {
		return nil, err
	}
	list, err := List(ctx, c, 1, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("渠道已创建，但回查失败: %w", err)
	}
	for _, ch := range list.Items {
		if ch.Name == req.Name {
			return &ch, nil
		}
	}
	return &Summary{Name: req.Name, Type: req.Type, Status: 1}, nil
}

// UpdateFields 更新渠道（PATCH 语义：只发要改的字段 + id）。
// 注意不要传 status/created_time/response_time/balance 等只读字段（上游会拒绝或清零）。
func UpdateFields(ctx context.Context, c *newapi.Client, id int, fields map[string]any) error {
	if id <= 0 {
		return fmt.Errorf("id 必须为正整数")
	}
	body := map[string]any{"id": id}
	for k, v := range fields {
		body[k] = v
	}
	_, err := c.Do(ctx, "PUT", newapi.RouteChannels, nil, body)
	return err
}

// TagReq 按 tag 批量编辑渠道（上游 controller/channel.go ChannelTag 的裁剪版，
// 不含 param_override/header_override 敏感项）。指针字段 nil=不改。
type TagReq struct {
	Tag          string  `json:"tag"`
	NewTag       *string `json:"new_tag,omitempty"`
	Priority     *int64  `json:"priority,omitempty"`
	Weight       *uint   `json:"weight,omitempty"`
	ModelMapping *string `json:"model_mapping,omitempty"`
	Models       *string `json:"models,omitempty"`
	Groups       *string `json:"groups,omitempty"`
}

// EditByTag 批量编辑同 tag 渠道（PUT /api/channel/tag，上游 EditTagChannels）。
func EditByTag(ctx context.Context, c *newapi.Client, req TagReq) (json.RawMessage, error) {
	data, err := c.Do(ctx, "PUT", newapi.RouteChannelTag, nil, req)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// SetTagStatus 按 tag 批量启停（POST /api/channel/tag/enabled|disabled）。
func SetTagStatus(ctx context.Context, c *newapi.Client, tag string, enabled bool) (json.RawMessage, error) {
	path := newapi.RouteChannelTagEnable
	if !enabled {
		path = newapi.RouteChannelTagDisable
	}
	data, err := c.Do(ctx, "POST", path, nil, map[string]any{"tag": tag})
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Delete 删除渠道。
func Delete(ctx context.Context, c *newapi.Client, id int) error {
	_, err := c.Do(ctx, "DELETE", fmt.Sprintf(newapi.RouteChannelDetail, id), nil, nil)
	return err
}
