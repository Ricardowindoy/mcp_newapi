// Package newapi 是 new-api 网关 HTTP 管理面的轻量客户端。
// 鉴权：面板 PAT，Authorization: Bearer <pat>（官方已移除 New-Api-User 头要求）。
package newapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client 是 new-api 管理面客户端。
type Client struct {
	BaseURL string // 形如 https://newapi.ashou.site，不带尾斜杠
	Token   string // 面板 PAT，可为空（仅公开端点可用）

	http    *http.Client
	lastErr error
}

// NewClient 创建客户端。timeout<=0 时默认 10s。
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		http:    &http.Client{Timeout: timeout},
	}
}

// envelope 是 new-api 统一响应包装。
type envelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// APIError 表示 new-api 返回 success:false 或非 2xx 的错误。
type APIError struct {
	StatusCode int
	Message    string
	Reachable  bool // false 表示网络层不可达（网关挂了），true 表示网关应答但业务失败
}

func (e *APIError) Error() string {
	if !e.Reachable {
		return fmt.Sprintf("newapi 网关不可达: %v", e.Message)
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("newapi 应答 %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("newapi 业务错误: %s", e.Message)
}

// Do 发起一次管理面请求，解包统一响应，将 data 原样返回。
// method/path 如 GET /api/status；query 可为 nil；body 会被 JSON 编码。
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body any) (json.RawMessage, error) {
	env, _, resp, err := c.doRequest(ctx, method, path, query, body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 || !env.Success {
		msg := env.Message
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return nil, &APIError{Reachable: true, StatusCode: resp.StatusCode, Message: msg}
	}
	return env.Data, nil
}

// DoTopLevel 发起请求并把整个顶层 JSON 应答解析为 map 返回。
// 用于 time/error_code 等业务字段放在顶层（而非 data）的端点（如渠道测试）。
// 注意：业务 success=false 不作为错误返回，由调用方按业务语义处理；
// 仅网络不可达 / 非 JSON 应答 / HTTP>=500 返回错误。
func (c *Client) DoTopLevel(ctx context.Context, method, path string, query url.Values, body any) (map[string]any, error) {
	_, raw, resp, err := c.doRequest(ctx, method, path, query, body)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if jerr := json.Unmarshal(raw, &m); jerr != nil {
		return nil, &APIError{
			Reachable:  true,
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("非 JSON 应答: %s", truncate(string(raw), 200)),
		}
	}
	return m, nil
}

// doRequest 发送请求并返回（envelope, 原始字节, 响应, 错误）。
func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values, body any) (*envelope, []byte, *http.Response, error) {
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("请求体编码失败: %w", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, nil, &APIError{Message: err.Error(), Reachable: false}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, nil, resp, &APIError{Message: err.Error(), Reachable: true, StatusCode: resp.StatusCode}
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		// 非 JSON 应答（如反代错误页）
		return nil, raw, resp, &APIError{
			Reachable:  true,
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("非 JSON 应答（可能被反代拦截或路径不存在）: %s", truncate(string(raw), 200)),
		}
	}
	return &env, raw, resp, nil
}

// GetJSON 便捷方法：GET 并把 data 解到 out。
func (c *Client) GetJSON(ctx context.Context, path string, query url.Values, out any) error {
	data, err := c.Do(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
