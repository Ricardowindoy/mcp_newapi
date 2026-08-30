// Package tokens 令牌管理域（读写一体，普通 PAT 即可用）。
// 上游契约：controller/token.go（GET/POST/PUT/DELETE /api/token/*）。
// 创建后完整 key 只在面板可见，本模块只透出掩码。
package tokens

import (
	"context"
	"fmt"
	"net/url"

	"mcp_newapi/internal/newapi"
)

// Summary 是用户令牌的裁剪 DTO（key 服务端本就掩码，防御性再掩码）。
type Summary struct {
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

type raw struct {
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

func (t raw) toSummary() Summary {
	return Summary{
		ID: t.ID, Name: t.Name, Key: newapi.MaskKey(t.Key), Status: t.Status,
		UsedQuota: t.UsedQuota, RemainQuota: t.RemainQuota,
		UnlimitedQuota: t.UnlimitedQuota, ExpiredTime: t.ExpiredTime,
		CreatedTime: t.CreatedTime, AccessedTime: t.AccessedTime,
	}
}

// List 拉取当前 PAT 用户的令牌列表。
func List(ctx context.Context, c *newapi.Client, page, pageSize int) (*newapi.PageResult[Summary], error) {
	q := url.Values{}
	q.Set("p", newapi.Itoa(page, 1))
	if pageSize > 0 {
		q.Set("page_size", fmt.Sprintf("%d", pageSize))
	}
	var p newapi.Paged[raw]
	if err := c.GetJSON(ctx, newapi.RouteTokens, q, &p); err != nil {
		return nil, err
	}
	out := &newapi.PageResult[Summary]{
		Page: p.Page, PageSize: p.PageSize, Total: p.Total,
		Items: make([]Summary, 0, len(p.Items)),
	}
	for _, t := range p.Items {
		out.Items = append(out.Items, t.toSummary())
	}
	return out, nil
}

// CreateReq 是创建令牌的参数。
type CreateReq struct {
	Name               string `json:"name"`
	RemainQuota        int64  `json:"remain_quota"` // quota 单位；Unlimited 时忽略
	UnlimitedQuota     bool   `json:"unlimited_quota"`
	ExpiredTime        int64  `json:"expired_time"` // -1=永不过期
	ModelLimitsEnabled bool   `json:"model_limits_enabled"`
	ModelLimits        string `json:"model_limits"` // 逗号分隔模型名
	AllowIps           string `json:"allow_ips"`
	Group              string `json:"group"` // 空=default
}

// Create 创建令牌。上游 AddToken 不回传 key/id，
// 这里再按名称搜索拿回 id 与掩码 key（完整 key 请在面板查看）。
func Create(ctx context.Context, c *newapi.Client, req CreateReq) (*Summary, error) {
	if _, err := c.Do(ctx, "POST", newapi.RouteTokens, nil, req); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("keyword", req.Name)
	q.Set("p", "1")
	q.Set("page_size", "10")
	var p newapi.Paged[raw]
	if err := c.GetJSON(ctx, newapi.RouteTokenSearch, q, &p); err != nil {
		return nil, fmt.Errorf("令牌已创建，但查询失败: %w", err)
	}
	for _, t := range p.Items {
		if t.Name == req.Name {
			s := t.toSummary()
			return &s, nil
		}
	}
	return &Summary{Name: req.Name}, nil
}

// Delete 删除当前用户的令牌（不可恢复）。
func Delete(ctx context.Context, c *newapi.Client, id int) error {
	_, err := c.Do(ctx, "DELETE", fmt.Sprintf(newapi.RouteTokenID, id), nil, nil)
	return err
}
