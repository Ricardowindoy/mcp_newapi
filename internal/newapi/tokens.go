package newapi

// tokens.go 令牌管理域（读写一体的独立模块，普通 PAT 即可用）。
// 上游契约：controller/token.go（GET/POST/PUT/DELETE /api/token/*）。
// 创建后完整 key 只在面板可见，本模块只透出掩码。

import (
	"context"
	"fmt"
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

func (t tokenRaw) toSummary() TokenSummary {
	return TokenSummary{
		ID: t.ID, Name: t.Name, Key: MaskKey(t.Key), Status: t.Status,
		UsedQuota: t.UsedQuota, RemainQuota: t.RemainQuota,
		UnlimitedQuota: t.UnlimitedQuota, ExpiredTime: t.ExpiredTime,
		CreatedTime: t.CreatedTime, AccessedTime: t.AccessedTime,
	}
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
		out.Items = append(out.Items, t.toSummary())
	}
	return out, nil
}

// TokenCreateReq 是创建令牌的参数。
type TokenCreateReq struct {
	Name               string `json:"name"`
	RemainQuota        int64  `json:"remain_quota"` // quota 单位；Unlimited 时忽略
	UnlimitedQuota     bool   `json:"unlimited_quota"`
	ExpiredTime        int64  `json:"expired_time"` // -1=永不过期
	ModelLimitsEnabled bool   `json:"model_limits_enabled"`
	ModelLimits        string `json:"model_limits"` // 逗号分隔模型名
	AllowIps           string `json:"allow_ips"`
	Group              string `json:"group"` // 空=default
}

// CreateToken 创建令牌。上游 AddToken 不回传 key/id，
// 这里再按名称搜索拿回 id 与掩码 key（完整 key 请在面板查看）。
func (c *Client) CreateToken(ctx context.Context, req TokenCreateReq) (*TokenSummary, error) {
	if _, err := c.Do(ctx, "POST", RouteTokens, nil, req); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("keyword", req.Name)
	q.Set("p", "1")
	q.Set("page_size", "10")
	var raw paged[tokenRaw]
	if err := c.GetJSON(ctx, RouteTokenSearch, q, &raw); err != nil {
		return nil, fmt.Errorf("令牌已创建，但查询失败: %w", err)
	}
	for _, t := range raw.Items {
		if t.Name == req.Name {
			s := t.toSummary()
			return &s, nil
		}
	}
	return &TokenSummary{Name: req.Name}, nil
}

// DeleteToken 删除当前用户的令牌（不可恢复）。
func (c *Client) DeleteToken(ctx context.Context, id int) error {
	_, err := c.Do(ctx, "DELETE", fmt.Sprintf(RouteTokenID, id), nil, nil)
	return err
}
