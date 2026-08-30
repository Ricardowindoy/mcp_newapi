package newapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient 起一个 mock 上游并返回指向它的 Client。
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "test-pat", 0)
}

func TestDoEnvelope(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-pat" {
			t.Errorf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true, "message": "", "data": map[string]any{"x": 1},
		})
	})
	data, err := c.Do(context.Background(), "GET", "/api/anything", nil, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["x"] != float64(1) {
		t.Errorf("data.x = %v", m["x"])
	}
}

func TestDoBusinessError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false, "message": "无权进行此操作，未登录且未提供 access token",
		})
	})
	_, err := c.Do(context.Background(), "GET", "/api/channel/", nil, nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("want APIError, got %T: %v", err, err)
	}
	if !apiErr.Reachable || apiErr.Message == "" {
		t.Errorf("Reachable=%v Message=%q", apiErr.Reachable, apiErr.Message)
	}
}

func TestDoUnreachable(t *testing.T) {
	// 指向一个立刻关闭的端口：ClientIP 不可达
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	c := NewClient(url, "", 0)
	_, err := c.Do(context.Background(), "GET", "/api/status", nil, nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("want APIError, got %T: %v", err, err)
	}
	if apiErr.Reachable {
		t.Errorf("应标记为不可达")
	}
}

func TestDoTopLevel(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"message":"bad upstream","time":1.25,"error_code":"unknown_error"}`))
	})
	m, err := c.DoTopLevel(context.Background(), "GET", "/api/channel/test/1", nil, nil)
	if err != nil {
		t.Fatalf("DoTopLevel: %v", err)
	}
	if m["success"] != false || m["error_code"] != "unknown_error" || m["time"] != float64(1.25) {
		t.Errorf("top-level 字段解析错误: %v", m)
	}
}

func TestSetChannelStatusChangedParse(t *testing.T) {
	var gotBody map[string]any
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"","data":true}`))
	})
	changed, err := c.SetChannelStatus(context.Background(), 7, false)
	if err != nil {
		t.Fatalf("SetChannelStatus: %v", err)
	}
	if !changed {
		t.Errorf("changed 应为 true")
	}
	if gotBody["status"] != float64(2) {
		t.Errorf("禁用应发 status=2, got %v", gotBody["status"])
	}
}
