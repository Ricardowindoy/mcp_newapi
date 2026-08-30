// Package logs 日志域：消费/错误日志检索、总量统计、dashboard 按模型聚合。
package logs

import (
	"context"
	"net/url"

	"mcp_newapi/internal/newapi"
)

// Entry 是消费/错误日志的裁剪 DTO。
type Entry struct {
	ID               int     `json:"id"`
	CreatedAt        int64   `json:"created_at"`
	Type             int     `json:"type"` // 2=消费 5=错误 等
	Username         string  `json:"username"`
	TokenName        string  `json:"token_name"`
	ModelName        string  `json:"model_name"`
	Quota            float64 `json:"quota"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	UseTime          int     `json:"use_time"` // 秒
	IsStream         bool    `json:"is_stream"`
	Channel          int     `json:"channel"`
	ChannelName      string  `json:"channel_name"`
	Group            string  `json:"group"`
	Content          string  `json:"content,omitempty"` // 错误日志的正文
}

// Stat 是 log/stat 的汇总。
type Stat struct {
	Quota float64 `json:"quota"`
	RPM   int     `json:"rpm"`
	TPM   int     `json:"tpm"`
}

// Query 是日志检索参数（零值字段不发送）。
type Query struct {
	Type           int // 0=全部
	StartTimestamp int64
	EndTimestamp   int64
	ModelName      string
	TokenName      string
	Channel        int
	Page, PageSize int
}

// Search 检索日志。
func Search(ctx context.Context, c *newapi.Client, q Query) (*newapi.PageResult[Entry], error) {
	v := url.Values{}
	v.Set("p", newapi.Itoa(q.Page, 1))
	if q.PageSize > 0 {
		v.Set("page_size", newapi.Itoa(q.PageSize, 20))
	}
	if q.Type > 0 {
		v.Set("type", newapi.Itoa(q.Type, 0))
	}
	if q.StartTimestamp > 0 {
		v.Set("start_timestamp", newapi.Itoa64(q.StartTimestamp))
	}
	if q.EndTimestamp > 0 {
		v.Set("end_timestamp", newapi.Itoa64(q.EndTimestamp))
	}
	if q.ModelName != "" {
		v.Set("model_name", q.ModelName)
	}
	if q.TokenName != "" {
		v.Set("token_name", q.TokenName)
	}
	if q.Channel > 0 {
		v.Set("channel", newapi.Itoa(q.Channel, 0))
	}
	var p newapi.Paged[Entry]
	if err := c.GetJSON(ctx, newapi.RouteLogs, v, &p); err != nil {
		return nil, err
	}
	return &newapi.PageResult[Entry]{
		Page: p.Page, PageSize: p.PageSize, Total: p.Total, Items: p.Items,
	}, nil
}

// Count 统计满足条件的日志条数（只取分页 total，不拉条目）。
func Count(ctx context.Context, c *newapi.Client, q Query) (int, error) {
	q.Page, q.PageSize = 1, 1
	r, err := Search(ctx, c, q)
	if err != nil {
		return 0, err
	}
	return r.Total, nil
}

// StatWindow 拉取时间窗内的总量统计。
func StatWindow(ctx context.Context, c *newapi.Client, start, end int64) (*Stat, error) {
	v := url.Values{}
	v.Set("type", "0")
	v.Set("start_timestamp", newapi.Itoa64(start))
	v.Set("end_timestamp", newapi.Itoa64(end))
	var st Stat
	if err := c.GetJSON(ctx, newapi.RouteLogStat, v, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// point 是 /api/data/ 的分桶条目。
type point struct {
	ModelName string  `json:"model_name"`
	TokenUsed int     `json:"token_used"`
	Count     int     `json:"count"`
	Quota     float64 `json:"quota"`
}

// ModelUsage 是按模型聚合的用量。
type ModelUsage struct {
	Model    string  `json:"model"`
	Calls    int     `json:"calls"`
	Tokens   int     `json:"tokens"`
	Quota    float64 `json:"quota"`
	QuotaUSD float64 `json:"quota_usd"`
}

// AggregateByModel 拉取 /api/data/ 并按模型聚合（按消费额降序）。
func AggregateByModel(ctx context.Context, c *newapi.Client, start, end int64) ([]ModelUsage, error) {
	v := url.Values{}
	v.Set("start_timestamp", newapi.Itoa64(start))
	v.Set("end_timestamp", newapi.Itoa64(end))
	v.Set("default_time", "custom")
	var pts []point
	if err := c.GetJSON(ctx, newapi.RouteData, v, &pts); err != nil {
		return nil, err
	}
	agg := map[string]*ModelUsage{}
	for _, p := range pts {
		m, ok := agg[p.ModelName]
		if !ok {
			m = &ModelUsage{Model: p.ModelName}
			agg[p.ModelName] = m
		}
		m.Calls += p.Count
		m.Tokens += p.TokenUsed
		m.Quota += p.Quota
	}
	out := make([]ModelUsage, 0, len(agg))
	for _, m := range agg {
		m.QuotaUSD = m.Quota / newapi.QuotaPerUnit
		out = append(out, *m)
	}
	sortByQuota(out)
	return out, nil
}
