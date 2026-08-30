package options

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mcp_newapi/internal/newapi"
)

// TestList 覆盖正常解析 + 按 key 排序。
func TestList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/option/" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": []map[string]string{
				{"key": "AutomaticDisableKeywords", "value": "your credit balance is too low"},
				{"key": "AutomaticDisableChannelEnabled", "value": "true"},
				{"key": "AutomaticDisableStatusCodes", "value": "401,403"},
			},
		})
	}))
	defer srv.Close()
	entries, err := List(context.Background(), newapi.NewClient(srv.URL, "test", 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("len = %d, want 3", len(entries))
	}
	// 排序后 AutomaticDisableChannelEnabled < AutomaticDisableKeywords < AutomaticDisableStatusCodes
	if entries[0].Key != "AutomaticDisableChannelEnabled" || entries[0].Value != "true" {
		t.Fatalf("entries[0] = %+v", entries[0])
	}
	if entries[2].Key != "AutomaticDisableStatusCodes" || entries[2].Value != "401,403" {
		t.Fatalf("entries[2] = %+v", entries[2])
	}
}

// TestUpdateRequestAndBusinessError 覆盖 PUT 请求体形态 + success:false 业务错误。
func TestUpdateRequestAndBusinessError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/option/" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "AutomaticDisableChannelEnabled" || body["value"] != "true" {
			t.Fatalf("body wrong: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "message": "无效的参数"})
	}))
	defer srv.Close()
	err := Update(context.Background(), newapi.NewClient(srv.URL, "test", 0),
		"AutomaticDisableChannelEnabled", "true")
	if err == nil {
		t.Fatal("want business error, got nil")
	}
}
