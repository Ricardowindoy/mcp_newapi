package newapi

// channels_admin.go 渠道域·管理操作（ChannelSensitiveWrite，admin 档）。
// 上游契约（controller/channel.go）：
//   AddChannel:    POST /api/channel/   body {mode, channel:{...}}，channel 需带 type/key/models
//   UpdateChannel: PUT  /api/channel/   PATCH 语义（只覆盖出现的字段）；
//                  status 字段被拒（启停走 channel_ops.go 的专用端点）；
//                  created_time/test_time/response_time/balance 等只读字段若传入会被清零
//   DeleteChannel: DELETE /api/channel/:id
// 敏感约束：完整 key 只在创建/更新时由调用方显式传入，任何响应不回传完整 key。

import (
	"context"
	"fmt"
)

// ChannelUpsertReq 是创建/更新渠道的请求体。
type ChannelUpsertReq struct {
	Name           string `json:"name"`
	Type           int    `json:"type"` // 渠道类型，如 1=OpenAI
	Key            string `json:"key"`
	BaseURL        string `json:"base_url,omitempty"`
	Models         string `json:"models"` // 逗号分隔
	Group          string `json:"group"`  // 逗号分隔，default
	ModelMapping   string `json:"model_mapping,omitempty"`
	Priority       int    `json:"priority,omitempty"`
	Weight         int    `json:"weight,omitempty"`
	TestModel      string `json:"test_model,omitempty"`
	OpenAIOrganization string `json:"openai_organization,omitempty"`
	Other          string `json:"other,omitempty"`
}

// CreateChannel 创建渠道，返回新渠道（GET 详情拿 id 需由调用方列表查询；上游不回传 id）。
func (c *Client) CreateChannel(ctx context.Context, req ChannelUpsertReq) (*ChannelSummary, error) {
	body := map[string]any{
		"mode":    "single",
		"channel": req,
	}
	if _, err := c.Do(ctx, "POST", RouteChannels, nil, body); err != nil {
		return nil, err
	}
	// 上游不回传 id：按名称搜回
	list, err := c.ListChannels(ctx, 1, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("渠道已创建，但回查失败: %w", err)
	}
	for _, ch := range list.Items {
		if ch.Name == req.Name {
			return &ch, nil
		}
	}
	return &ChannelSummary{Name: req.Name, Type: req.Type, Status: 1}, nil
}

// UpdateChannelFields 更新渠道（PATCH 语义：只发要改的字段 + id）。
// 注意不要传 status/created_time/response_time/balance 等只读字段（上游会拒绝或清零）。
func (c *Client) UpdateChannelFields(ctx context.Context, id int, fields map[string]any) error {
	if id <= 0 {
		return fmt.Errorf("id 必须为正整数")
	}
	body := map[string]any{"id": id}
	for k, v := range fields {
		body[k] = v
	}
	_, err := c.Do(ctx, "PUT", RouteChannels, nil, body)
	return err
}

// DeleteChannel 删除渠道。
func (c *Client) DeleteChannel(ctx context.Context, id int) error {
	_, err := c.Do(ctx, "DELETE", fmt.Sprintf(RouteChannelDetail, id), nil, nil)
	return err
}
