package newapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ChannelSummary 是渠道列表/详情的裁剪 DTO。
// key 已在构造时掩码；完整 key 永不出本包。
type ChannelSummary struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Type        int     `json:"type"`
	Status      int     `json:"status"` // 1=启用 2=手动禁用 3=自动禁用
	StatusReason string `json:"status_reason,omitempty"`
	Balance     float64 `json:"balance"`
	BaseURL     string  `json:"base_url,omitempty"`
	Models      string  `json:"models"`
	Group       string  `json:"group"`
	Priority    int     `json:"priority"`
	Weight      int     `json:"weight"`
	TestModel   string  `json:"test_model,omitempty"`
	ResponseTime int    `json:"response_time"` // ms，0=未测
	UsedQuota   float64 `json:"used_quota"`
	Key         string  `json:"key"` // 掩码后
}

type channelRaw struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Type         int     `json:"type"`
	Key          string  `json:"key"`
	Status       int     `json:"status"`
	Balance      float64 `json:"balance"`
	BaseURL      string  `json:"base_url"`
	Models       string  `json:"models"`
	Group        string  `json:"group"`
	Priority     int     `json:"priority"`
	Weight       int     `json:"weight"`
	TestModel    string  `json:"test_model"`
	ResponseTime int     `json:"response_time"`
	UsedQuota    float64 `json:"used_quota"`
	OtherInfo    string  `json:"other_info"`
}

func (c channelRaw) toSummary() ChannelSummary {
	s := ChannelSummary{
		ID: c.ID, Name: c.Name, Type: c.Type, Status: c.Status,
		Balance: c.Balance, BaseURL: c.BaseURL, Models: c.Models,
		Group: c.Group, Priority: c.Priority, Weight: c.Weight,
		TestModel: c.TestModel, ResponseTime: c.ResponseTime,
		UsedQuota: c.UsedQuota, Key: MaskKey(c.Key),
	}
	// status_reason 藏在 other_info JSON 里
	if c.OtherInfo != "" {
		var oi struct {
			StatusReason string `json:"status_reason"`
		}
		if json.Unmarshal([]byte(c.OtherInfo), &oi) == nil && oi.StatusReason != "" {
			s.StatusReason = oi.StatusReason
		}
	}
	return s
}

// paged 是列表端点的统一分页壳。
type paged[T any] struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int   `json:"total"`
	Items    []T   `json:"items"`
}

// PageResult 是给 MCP 层的分页结果。
type PageResult[T any] struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
	Items    []T `json:"items"`
}

// ListChannels 拉取渠道列表（需管理员 PAT）。
// status: 0=全部 1=启用 2=禁用（服务端语义）；传 0 时不带该参数。
func (c *Client) ListChannels(ctx context.Context, page, pageSize, status int) (*PageResult[ChannelSummary], error) {
	q := url.Values{}
	q.Set("p", itoa(page, 1))
	if pageSize > 0 {
		q.Set("page_size", itoa(pageSize, 20))
	}
	if status > 0 {
		q.Set("status", strconv.Itoa(status))
	}
	var raw paged[channelRaw]
	if err := c.GetJSON(ctx, RouteChannels, q, &raw); err != nil {
		return nil, err
	}
	out := &PageResult[ChannelSummary]{
		Page: raw.Page, PageSize: raw.PageSize, Total: raw.Total,
		Items: make([]ChannelSummary, 0, len(raw.Items)),
	}
	for _, r := range raw.Items {
		out.Items = append(out.Items, r.toSummary())
	}
	return out, nil
}

// GetChannel 拉取单渠道详情（需管理员 PAT）。
func (c *Client) GetChannel(ctx context.Context, id int) (*ChannelSummary, error) {
	var raw channelRaw
	if err := c.GetJSON(ctx, fmt.Sprintf(RouteChannelDetail, id), nil, &raw); err != nil {
		return nil, err
	}
	s := raw.toSummary()
	return &s, nil
}

func itoa(v, def int) string {
	if v <= 0 {
		v = def
	}
	return strconv.Itoa(v)
}
