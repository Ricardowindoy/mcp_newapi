package newapi

import (
	"context"
	"net/url"
)

// TokenSummary 是用户令牌的裁剪 DTO（key 服务端本就掩码，防御性再掩码）。
type TokenSummary struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	Key            string  `json:"key"`
	Status         int     `json:"status"` // 1=启用 2=禁用 3=过期 4=耗尽
	UsedQuota      float64 `json:"used_quota"`
	RemainQuota    float64 `json:"remain_quota"`
	UnlimitedQuota bool    `json:"unlimited_quota"`
	ExpiredTime    int64   `json:"expired_time"` // -1=永不过期
	CreatedTime    int64   `json:"created_time"`
	AccessedTime   int64   `json:"accessed_time"`
}

type tokenRaw struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	Key            string  `json:"key"`
	Status         int     `json:"status"`
	UsedQuota      float64 `json:"used_quota"`
	RemainQuota    float64 `json:"remain_quota"`
	UnlimitedQuota bool    `json:"unlimited_quota"`
	ExpiredTime    int64   `json:"expired_time"`
	CreatedTime    int64   `json:"created_time"`
	AccessedTime   int64   `json:"accessed_time"`
}

// ListTokens 拉取当前 PAT 用户的令牌列表。
func (c *Client) ListTokens(ctx context.Context, page, pageSize int) (*PageResult[TokenSummary], error) {
	q := url.Values{}
	q.Set("p", itoa(page, 1))
	if pageSize > 0 {
		q.Set("page_size", itoa(pageSize, 20))
	}
	var raw paged[tokenRaw]
	if err := c.GetJSON(ctx, RouteTokens, q, &raw); err != nil {
		return nil, err
	}
	out := &PageResult[TokenSummary]{
		Page: raw.Page, PageSize: raw.PageSize, Total: raw.Total,
		Items: make([]TokenSummary, 0, len(raw.Items)),
	}
	for _, t := range raw.Items {
		out.Items = append(out.Items, TokenSummary{
			ID: t.ID, Name: t.Name, Key: MaskKey(t.Key), Status: t.Status,
			UsedQuota: t.UsedQuota, RemainQuota: t.RemainQuota,
			UnlimitedQuota: t.UnlimitedQuota, ExpiredTime: t.ExpiredTime,
			CreatedTime: t.CreatedTime, AccessedTime: t.AccessedTime,
		})
	}
	return out, nil
}
