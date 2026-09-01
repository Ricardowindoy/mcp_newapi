package channels

// channel_test.go Summary 的 model_mapping 透传测试（mock 上游）。
// 上游契约实测（2026-09-01 生产网关）：model_mapping 为 JSON 字符串，可为 null/空串。

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"mcp_newapi/internal/newapi"
)

const mappingFixture = `{"deepseek-v4-flash-0731":"deepseek-v4-flash"}`

func TestGetReturnsModelMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"success":true,"message":"","data":{"id":9,"name":"ch","type":1,"status":1,
			"models":"m1","group":"default","priority":0,"weight":0,"response_time":0,"used_quota":0,"key":"sk-x",
			"model_mapping":%q}}`, mappingFixture)
	}))
	t.Cleanup(srv.Close)
	c := newapi.NewClient(srv.URL, "test-pat", 0)

	ch, err := Get(context.Background(), c, 9)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ch.ModelMapping != mappingFixture {
		t.Errorf("model_mapping 应原样透传，got %q", ch.ModelMapping)
	}
}

func TestListModelMappingNullAndEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"","data":{"page":1,"page_size":20,"total":2,"items":[
			{"id":1,"name":"a","type":1,"status":1,"models":"m","group":"default","key":"sk-abcd1234efgh7890","model_mapping":null},
			{"id":2,"name":"b","type":1,"status":1,"models":"m","group":"default","key":"sk-abcd1234efgh7890","model_mapping":` + `"` + `"}]}}`))
	}))
	t.Cleanup(srv.Close)
	c := newapi.NewClient(srv.URL, "test-pat", 0)

	res, err := List(context.Background(), c, 1, 20, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// null → 空串且 omitempty 不出字段
	b1, _ := json.Marshal(res.Items[0])
	if res.Items[0].ModelMapping != "" {
		t.Errorf("null 应归一为空串，got %q", res.Items[0].ModelMapping)
	}
	if string(b1) == "" || jsonContains(t, b1, "model_mapping") {
		t.Errorf("空映射不应输出 model_mapping 字段: %s", b1)
	}
	// 显式空串同样不输出
	b2, _ := json.Marshal(res.Items[1])
	if jsonContains(t, b2, "model_mapping") {
		t.Errorf("空串映射不应输出 model_mapping 字段: %s", b2)
	}
	// 有值渠道透传
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"success":true,"message":"","data":{"page":1,"page_size":20,"total":1,"items":[
			{"id":3,"name":"c","type":1,"status":1,"models":"m","group":"default","key":"sk-abcd1234efgh7890","model_mapping":%q}]}}`, mappingFixture)
	}))
	t.Cleanup(srv2.Close)
	c2 := newapi.NewClient(srv2.URL, "test-pat", 0)
	res2, err := List(context.Background(), c2, 1, 20, 0)
	if err != nil {
		t.Fatalf("List(2): %v", err)
	}
	if res2.Items[0].ModelMapping != mappingFixture {
		t.Errorf("列表项 model_mapping 应透传，got %q", res2.Items[0].ModelMapping)
	}
}

func jsonContains(t *testing.T, data []byte, key string) bool {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", data, err)
	}
	_, ok := m[key]
	return ok
}
