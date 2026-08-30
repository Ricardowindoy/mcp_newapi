package newapi

// channel_ops.go 渠道域·运维操作（需 ChannelOperate 权限的管理员 PAT）。
// 上游契约：controller/channel-test.go、controller/channel.go（见 /api/channel 路由组）。

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// TestResult 是渠道测试结果（业务失败也是有效结果，不算错误）。
type TestResult struct {
	Success   bool    `json:"success"`
	Message   string  `json:"message,omitempty"`
	TimeSec   float64 `json:"time_seconds"`
	ErrorCode string  `json:"error_code,omitempty"`
}

// TestChannel 对单渠道发一次测试请求。model 为空时用渠道默认测试模型。
// 注意：该端点把 time/error_code 放在顶层而非 data，走 DoTopLevel。
func (c *Client) TestChannel(ctx context.Context, id int, model string) (*TestResult, error) {
	q := url.Values{}
	if model != "" {
		q.Set("model", model)
	}
	m, err := c.DoTopLevel(ctx, "GET", fmt.Sprintf(RouteChannelTest, id), q, nil)
	if err != nil {
		return nil, err
	}
	tr := &TestResult{}
	if v, ok := m["success"].(bool); ok {
		tr.Success = v
	}
	if v, ok := m["message"].(string); ok {
		tr.Message = v
	}
	if v, ok := m["time"].(float64); ok {
		tr.TimeSec = v
	}
	if v, ok := m["error_code"].(string); ok {
		tr.ErrorCode = v
	}
	return tr, nil
}

// TestAllChannels 触发全量渠道测试（异步系统任务），返回任务信息。
func (c *Client) TestAllChannels(ctx context.Context) (map[string]any, error) {
	data, err := c.Do(ctx, "GET", RouteChannelTestAll, nil, nil)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// UpdateChannelBalance 刷新单渠道余额（部分渠道类型不支持，返回业务错误）。
func (c *Client) UpdateChannelBalance(ctx context.Context, id int) error {
	_, err := c.Do(ctx, "GET", fmt.Sprintf(RouteChannelBal, id), nil, nil)
	return err
}

// SetChannelStatus 启用/禁用渠道（记录为 manual operation），返回是否有实际变更。
// 上游契约：POST /api/channel/:id/status，body {status:1|2}，data=changed(bool)。
func (c *Client) SetChannelStatus(ctx context.Context, id int, enabled bool) (bool, error) {
	status := 2
	if enabled {
		status = 1
	}
	data, err := c.Do(ctx, "POST", fmt.Sprintf(RouteChannelStatus, id), nil, map[string]any{"status": status})
	if err != nil {
		return false, err
	}
	return string(data) == "true", nil
}
