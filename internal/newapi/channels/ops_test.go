package channels

// ops_test.go SetStatus 的请求/响应语义测试（mock 上游）。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mcp_newapi/internal/newapi"
)

func TestSetStatusRequestAndChanged(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"","data":true}`))
	}))
	t.Cleanup(srv.Close)
	c := newapi.NewClient(srv.URL, "test-pat", 0)

	changed, err := SetStatus(context.Background(), c, 7, false)
	if err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if !changed {
		t.Errorf("changed 应为 true")
	}
	if gotBody["status"] != float64(2) {
		t.Errorf("禁用应发 status=2, got %v", gotBody["status"])
	}
}
