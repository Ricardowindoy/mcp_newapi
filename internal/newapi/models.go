package newapi

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
)

// Models 拉取全站可用模型（data 为 分组→模型数组 的映射，透传并排序）。
func (c *Client) Models(ctx context.Context) (map[string][]string, error) {
	data, err := c.Do(ctx, "GET", RouteModels, nil, nil)
	if err != nil {
		return nil, err
	}
	var m map[string][]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	for g := range m {
		lst := m[g]
		sort.Strings(lst)
		m[g] = lst
	}
	return m, nil
}

// Pricing 拉取模型倍率定价（实例可能禁用，返回原始 data）。
func (c *Client) Pricing(ctx context.Context, model string) (json.RawMessage, error) {
	q := url.Values{}
	if model != "" {
		q.Set("model", model)
	}
	return c.Do(ctx, "GET", RoutePricing, q, nil)
}
