package logs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mcp_newapi/internal/newapi"
)

// TestCount 覆盖 Count 的查询参数（page_size=1、type 透传）与 total 解析。
func TestCount(t *testing.T) {
	var gotType, gotPageSize string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotType = q.Get("type")
		gotPageSize = q.Get("page_size")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"page": 1, "page_size": 1, "total": 42,
				"items": []map[string]any{{"id": 1, "type": 2}},
			},
		})
	}))
	defer srv.Close()
	n, err := Count(context.Background(), newapi.NewClient(srv.URL, "t", 0), Query{Type: 2})
	if err != nil {
		t.Fatal(err)
	}
	if n != 42 {
		t.Fatalf("count = %d, want 42", n)
	}
	if gotType != "2" || gotPageSize != "1" {
		t.Fatalf("query params wrong: type=%q page_size=%q", gotType, gotPageSize)
	}
}
