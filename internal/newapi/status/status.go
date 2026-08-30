// Package status 站点状态域：/api/status 公开端点 + relay 活性探测。
package status

import (
	"context"
	"encoding/json"
	"fmt"

	"mcp_newapi/internal/newapi"
)

// Data 是 GET /api/status 的 data 裁剪（只保留 Agent 关心字段，
// new-api 实际返回字段更多，多余字段经 json.Unmarshal 忽略）。
type Data struct {
	Version         string `json:"version"`
	StartTime       int64  `json:"start_time"`
	EmailVerify     bool   `json:"email_verification"`
	RegisterEnabled bool   `json:"register_enabled"`
}

// Get 拉取站点状态（公开端点，无需 PAT）。
func Get(ctx context.Context, c *newapi.Client) (*Data, error) {
	data, err := c.Do(ctx, "GET", newapi.RouteStatus, nil, nil)
	if err != nil {
		return nil, err
	}
	var d Data
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("status 解析失败: %w", err)
	}
	return &d, nil
}

// RelayProbe 用 /v1/models 探测 relay 活性（无需鉴权也应返回 200/401，
// 返回错误只说明 relay 网络层不可达）。
func RelayProbe(ctx context.Context, c *newapi.Client) error {
	_, err := c.Do(ctx, "GET", "/v1/models", nil, nil)
	// 401/403 说明 relay 活着，只是没带（或带错）key
	if err != nil {
		if apiErr, ok := err.(*newapi.APIError); ok && apiErr.Reachable {
			return nil
		}
		return err
	}
	return nil
}
