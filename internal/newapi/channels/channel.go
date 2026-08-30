// Package channels 渠道域：读（channel.go）/ 运维（ops.go）/ 管理（admin.go）。
// 上游契约：controller/channel*.go（/api/channel 路由组，AdminAuth）。
package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"mcp_newapi/internal/newapi"
)

// Summary 是渠道列表/详情的裁剪 DTO。
// key 已在构造时掩码；完整 key 永不出本包。
type Summary struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Type         int     `json:"type"`
	Status       int     `json:"status"` // 1=启用 2=手动禁用 3=自动禁用
	StatusReason string  `json:"status_reason,omitempty"`
	Balance      float64 `json:"balance"`
	BaseURL      string  `json:"base_url,omitempty"`
	Models       string  `json:"models"`
	Group        string  `json:"group"`
	Priority     int     `json:"priority"`
	Weight       int     `json:"weight"`
	TestModel    string  `json:"test_model,omitempty"`
	ResponseTime int     `json:"response_time"` // ms，0=未测
	UsedQuota    float64 `json:"used_quota"`
	Key          string  `json:"key"` // 掩码后
}

type raw struct {
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

func (r raw) toSummary() Summary {
	s := Summary{
		ID: r.ID, Name: r.Name, Type: r.Type, Status: r.Status,
		Balance: r.Balance, BaseURL: r.BaseURL, Models: r.Models,
		Group: r.Group, Priority: r.Priority, Weight: r.Weight,
		TestModel: r.TestModel, ResponseTime: r.ResponseTime,
		UsedQuota: r.UsedQuota, Key: newapi.MaskKey(r.Key),
	}
	// status_reason 藏在 other_info JSON 里
	if r.OtherInfo != "" {
		var oi struct {
			StatusReason string `json:"status_reason"`
		}
		if json.Unmarshal([]byte(r.OtherInfo), &oi) == nil && oi.StatusReason != "" {
			s.StatusReason = oi.StatusReason
		}
	}
	return s
}

// List 拉取渠道列表（需管理员 PAT）。
// status: 0=全部 1=启用 2=禁用（服务端语义）；传 0 时不带该参数。
func List(ctx context.Context, c *newapi.Client, page, pageSize, status int) (*newapi.PageResult[Summary], error) {
	q := url.Values{}
	q.Set("p", newapi.Itoa(page, 1))
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	if status > 0 {
		q.Set("status", strconv.Itoa(status))
	}
	var p newapi.Paged[raw]
	if err := c.GetJSON(ctx, newapi.RouteChannels, q, &p); err != nil {
		return nil, err
	}
	out := &newapi.PageResult[Summary]{
		Page: p.Page, PageSize: p.PageSize, Total: p.Total,
		Items: make([]Summary, 0, len(p.Items)),
	}
	for _, r := range p.Items {
		out.Items = append(out.Items, r.toSummary())
	}
	return out, nil
}

// Get 拉取单渠道详情（需管理员 PAT）。
func Get(ctx context.Context, c *newapi.Client, id int) (*Summary, error) {
	var r raw
	if err := c.GetJSON(ctx, fmt.Sprintf(newapi.RouteChannelDetail, id), nil, &r); err != nil {
		return nil, err
	}
	s := r.toSummary()
	return &s, nil
}
