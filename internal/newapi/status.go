package newapi

import (
	"context"
	"encoding/json"
	"fmt"
)

// StatusData 是 GET /api/status 的 data 裁剪（只保留 Agent 关心字段；
// new-api 实际返回字段更多，多余字段经 json.Unmarshal 忽略）。
type StatusData struct {
	Version      string `json:"version"`
	StartTime    int64  `json:"start_time"`
	// 常见运营开关（不同版本字段可能缺失，缺省零值）
	EmailVerify  bool   `json:"email_verification"`
	RegisterEnabled bool `json:"register_enabled"`
	// 原始 data 兜底，避免裁剪丢信息
	Raw json.RawMessage `json:"-"`
}

// Status 拉取站点状态（公开端点，无需 PAT）。
func (c *Client) Status(ctx context.Context) (*StatusData, error) {
	data, err := c.Do(ctx, "GET", RouteStatus, nil, nil)
	if err != nil {
		return nil, err
	}
	var s StatusData
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("status 解析失败: %w", err)
	}
	s.Raw = data
	return &s, nil
}

// RelayProbe 用 /v1/models 探测 relay 活性（无需鉴权也应返回 200/401，
// 返回错误只说明 relay 网络层不可达）。
func (c *Client) RelayProbe(ctx context.Context) error {
	_, err := c.Do(ctx, "GET", "/v1/models", nil, nil)
	// 401/403 说明 relay 活着，只是没带（或带错）key
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.Reachable {
			return nil
		}
		return err
	}
	return nil
}
