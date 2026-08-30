package options

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mcp_newapi/internal/newapi"
)

func TestParseRanges(t *testing.T) {
	rs, err := ParseRanges("500,502,429,402")
	if err != nil {
		t.Fatal(err)
	}
	want := []CodeRange{{402, 402}, {429, 429}, {500, 500}, {502, 502}}
	if len(rs) != len(want) {
		t.Fatalf("rs = %+v, want %+v", rs, want)
	}
	for i := range want {
		if rs[i] != want[i] {
			t.Fatalf("rs[%d] = %+v, want %+v", i, rs[i], want[i])
		}
	}
	// 相邻合并：402,403 → 402-403
	rs, _ = ParseRanges("402,403,100")
	if len(rs) != 2 || rs[0].Start != 100 || rs[1].Start != 402 || rs[1].End != 403 {
		t.Fatalf("merge wrong: %+v", rs)
	}
	// 全角逗号 + 空段
	rs, _ = ParseRanges("401，403 ， ,402")
	if len(rs) != 1 || rs[0].Start != 401 || rs[0].End != 403 {
		t.Fatalf("normalize wrong: %+v", rs)
	}
	// 非法 token 整体报错
	if _, err := ParseRanges("402,abc"); err == nil {
		t.Fatal("want error for invalid token")
	}
}

func TestSerializeAndMatch(t *testing.T) {
	rs := []CodeRange{{402, 402}, {500, 502}}
	if got := SerializeRanges(rs); got != "402,500-502" {
		t.Fatalf("serialize = %q", got)
	}
	if !CoveredBy(rs, 402) || !CoveredBy(rs, 501) || CoveredBy(rs, 403) {
		t.Fatal("covered semantics wrong")
	}
}

func TestAddAndRemove(t *testing.T) {
	// add：402 已被 401-403 覆盖 → already；405 追加 → 401-403,405
	added, err := StatusCodesOp("401-403", "add", []string{"402", "405"})
	if err != nil {
		t.Fatal(err)
	}
	if len(added.Already) != 1 || added.Already[0] != "402" || len(added.Added) != 1 || added.Added[0] != "405" {
		t.Fatalf("add result = %+v", added)
	}
	if added.Codes != "401-403,405" {
		t.Fatalf("codes = %q", added.Codes)
	}
	// remove：从 401-403 中挖掉 402 → 401,403；405 不在 → not_found
	removed, err := StatusCodesOp("401-403,405", "remove", []string{"402", "405"})
	if err != nil {
		t.Fatal(err)
	}
	if len(removed.Removed) != 2 || len(removed.NotFound) != 0 {
		t.Fatalf("remove result = %+v", removed)
	}
	if removed.Codes != "401,403" {
		t.Fatalf("codes = %q", removed.Codes)
	}
	// set：全量重写为规范形（500/502 不相邻不合并）
	set, err := StatusCodesOp("", "set", []string{"502", "500", "429"})
	if err != nil {
		t.Fatal(err)
	}
	if set.Codes != "429,500,502" {
		t.Fatalf("set codes = %q", set.Codes)
	}
}

// StatusCodesOp 是纯代数入口（不走网络），测试用。
func StatusCodesOp(existing, action string, tokens []string) (*CodesResult, error) {
	tokRs, err := ParseTokenList(tokens)
	if err != nil {
		return nil, err
	}
	rs, err := ParseRanges(existing)
	if err != nil {
		return nil, err
	}
	res := &CodesResult{Action: action, Raw: existing}
	switch action {
	case "set":
		res.Ranges = mergeRanges(append([]CodeRange{}, tokRs...))
	case "add":
		for _, tr := range tokRs {
			token := SerializeRanges([]CodeRange{tr})
			if coveredAll(rs, tr) {
				res.Already = append(res.Already, token)
			} else {
				res.Added = append(res.Added, token)
			}
		}
		res.Ranges = mergeRanges(append(append([]CodeRange{}, rs...), tokRs...))
	case "remove":
		for _, tr := range tokRs {
			token := SerializeRanges([]CodeRange{tr})
			if !overlapsAny(rs, tr) {
				res.NotFound = append(res.NotFound, token)
				continue
			}
			res.Removed = append(res.Removed, token)
			rs = subtract(rs, tr)
		}
		res.Ranges = rs
	}
	res.Codes = SerializeRanges(res.Ranges)
	return res, nil
}

// TestStatusCodesListHTTPTP 走 httptest 验证 list 的网络路径。
func TestStatusCodesListHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": []map[string]string{
				{"key": "AutomaticDisableStatusCodes", "value": "500,502,429,402"},
			},
		})
	}))
	defer srv.Close()
	res, err := StatusCodesList(context.Background(), newapi.NewClient(srv.URL, "t", 0), "disable")
	if err != nil {
		t.Fatal(err)
	}
	if res.Codes != "402,429,500,502" {
		t.Fatalf("codes = %q", res.Codes)
	}
	if res.Raw != "500,502,429,402" {
		t.Fatalf("raw = %q", res.Raw)
	}
}
