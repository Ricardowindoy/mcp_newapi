package newapi

import (
	"context"
	"net/url"
	"sort"
)

// LogEntry 是消费/错误日志的裁剪 DTO。
type LogEntry struct {
	ID              int     `json:"id"`
	CreatedAt       int64   `json:"created_at"`
	Type            int     `json:"type"` // 2=消费 5=错误 等
	Username        string  `json:"username"`
	TokenName       string  `json:"token_name"`
	ModelName       string  `json:"model_name"`
	Quota           float64 `json:"quota"`
	PromptTokens    int     `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	UseTime         int     `json:"use_time"` // 秒
	IsStream        bool    `json:"is_stream"`
	Channel         int     `json:"channel"`
	ChannelName     string  `json:"channel_name"`
	Group           string  `json:"group"`
	Content         string  `json:"content,omitempty"` // 错误日志的正文
}

// LogStat 是 log/stat 的汇总。
type LogStat struct {
	Quota float64 `json:"quota"`
	RPM   int     `json:"rpm"`
	TPM   int     `json:"tpm"`
}

// LogQuery 是日志检索参数（零值字段不发送）。
type LogQuery struct {
	Type           int // 0=全部
	StartTimestamp int64
	EndTimestamp   int64
	ModelName      string
	TokenName      string
	Channel        int
	Page, PageSize int
}

// Logs 检索日志。
func (c *Client) Logs(ctx context.Context, qy LogQuery) (*PageResult[LogEntry], error) {
	q := url.Values{}
	q.Set("p", itoa(qy.Page, 1))
	if qy.PageSize > 0 {
		q.Set("page_size", itoa(qy.PageSize, 20))
	}
	if qy.Type > 0 {
		q.Set("type", itoa(qy.Type, 0))
	}
	if qy.StartTimestamp > 0 {
		q.Set("start_timestamp", itoa64(qy.StartTimestamp))
	}
	if qy.EndTimestamp > 0 {
		q.Set("end_timestamp", itoa64(qy.EndTimestamp))
	}
	if qy.ModelName != "" {
		q.Set("model_name", qy.ModelName)
	}
	if qy.TokenName != "" {
		q.Set("token_name", qy.TokenName)
	}
	if qy.Channel > 0 {
		q.Set("channel", itoa(qy.Channel, 0))
	}
	var raw paged[LogEntry]
	if err := c.GetJSON(ctx, RouteLogs, q, &raw); err != nil {
		return nil, err
	}
	return &PageResult[LogEntry]{
		Page: raw.Page, PageSize: raw.PageSize, Total: raw.Total, Items: raw.Items,
	}, nil
}

// LogStatWindow 拉取时间窗内的总量统计。
func (c *Client) LogStatWindow(ctx context.Context, start, end int64) (*LogStat, error) {
	q := url.Values{}
	q.Set("type", "0")
	q.Set("start_timestamp", itoa64(start))
	q.Set("end_timestamp", itoa64(end))
	var st LogStat
	if err := c.GetJSON(ctx, RouteLogStat, q, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// dashboardPoint 是 /api/data/ 的分桶条目。
type dashboardPoint struct {
	ModelName string  `json:"model_name"`
	TokenUsed int     `json:"token_used"`
	Count     int     `json:"count"`
	Quota     float64 `json:"quota"`
}

// ModelUsage 是按模型聚合的用量。
type ModelUsage struct {
	Model      string  `json:"model"`
	Calls      int     `json:"calls"`
	Tokens     int     `json:"tokens"`
	Quota      float64 `json:"quota"`
	QuotaUSD   float64 `json:"quota_usd"`
}

// DashboardData 拉取 /api/data/ 并按模型聚合。
func (c *Client) DashboardData(ctx context.Context, start, end int64) ([]ModelUsage, error) {
	q := url.Values{}
	q.Set("start_timestamp", itoa64(start))
	q.Set("end_timestamp", itoa64(end))
	q.Set("default_time", "custom")
	var pts []dashboardPoint
	if err := c.GetJSON(ctx, RouteData, q, &pts); err != nil {
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
		m.QuotaUSD = m.Quota / QuotaPerUnit
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Quota > out[j].Quota })
	return out, nil
}

// QuotaPerUnit 是 new-api 的 quota→美元换算（common.QuotaPerUnit 默认值）。
const QuotaPerUnit = 500000.0

func itoa64(v int64) string {
	return itoa(int(v), 0)
}
