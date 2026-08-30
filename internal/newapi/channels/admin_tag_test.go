package channels

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mcp_newapi/internal/newapi"
)

// TestEditByTagRequest 覆盖 PUT /api/channel/tag 的路径与请求体。
func TestEditByTagRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/channel/tag" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["tag"] != "基元" || body["priority"] != float64(17) {
			t.Fatalf("body wrong: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": true})
	}))
	defer srv.Close()
	p := int64(17)
	data, err := EditByTag(context.Background(), newapi.NewClient(srv.URL, "t", 0), TagReq{Tag: "基元", Priority: &p})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "true" {
		t.Fatalf("data = %s", data)
	}
}

// TestSetTagStatusRequest 覆盖启停路径与请求体。
func TestSetTagStatusRequest(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["tag"] != "基元" {
			t.Fatalf("body wrong: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": true})
	}))
	defer srv.Close()
	c := newapi.NewClient(srv.URL, "t", 0)
	if _, err := SetTagStatus(context.Background(), c, "基元", false); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/channel/tag/disabled" {
		t.Fatalf("path = %s", gotPath)
	}
	if _, err := SetTagStatus(context.Background(), c, "基元", true); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/channel/tag/enabled" {
		t.Fatalf("path = %s", gotPath)
	}
}
