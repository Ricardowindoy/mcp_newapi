package options

// statuscodes.go autoban 状态码的增删查改代数。
// 上游契约：setting/operation_setting/status_code_ranges.go——
//   语法：逗号分隔 token，单码（"402"）或闭区间（"400-499"）；全角逗号归一；
//   越界 [100,599] 与 start>end 报错；解析后按 Start 排序并合并相邻（r.Start <= last.End+1）与重叠区间；
//   匹配语义 shouldMatchStatusCodeRanges 依赖有序区间；序列化单码直出、区间 "a-b"。
// 本文件在本地镜像该代数，保证 add/remove 产出上游可直接解析的规范串。

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"mcp_newapi/internal/newapi"
)

// CodeRange 是一个闭区间状态码段。
type CodeRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// target 对应的上游 option 键。
var statusCodeKeys = map[string]string{
	"disable": "AutomaticDisableStatusCodes",
	"retry":   "AutomaticRetryStatusCodes",
}

// StatusCodeKey 返回 target（disable|retry）对应的上游 option 键。
func StatusCodeKey(target string) (string, bool) {
	k, ok := statusCodeKeys[target]
	return k, ok
}

// ParseRanges 解析状态码串（上游 ParseHTTPStatusCodeRanges 的镜像实现）。
func ParseRanges(input string) ([]CodeRange, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}
	input = strings.ReplaceAll(input, "，", ",")
	var rs []CodeRange
	var invalid []string
	for _, seg := range strings.Split(input, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		r, err := parseToken(seg)
		if err != nil {
			invalid = append(invalid, seg)
			continue
		}
		rs = append(rs, r)
	}
	if len(invalid) > 0 {
		return nil, fmt.Errorf("无效状态码规则: %s", strings.Join(invalid, ", "))
	}
	return mergeRanges(rs), nil
}

// ParseTokenList 校验用户给的一组 token（单个非法即整体报错，fail-fast）。
func ParseTokenList(tokens []string) ([]CodeRange, error) {
	var rs []CodeRange
	for _, tok := range tokens {
		for _, seg := range strings.Split(tok, ",") {
			seg = strings.TrimSpace(seg)
			if seg == "" {
				continue
			}
			r, err := parseToken(seg)
			if err != nil {
				return nil, err
			}
			rs = append(rs, r)
		}
	}
	if len(rs) == 0 {
		return nil, fmt.Errorf("codes 不能为空（单码或区间，如 402、400-499）")
	}
	return rs, nil
}

func parseToken(tok string) (CodeRange, error) {
	tok = strings.TrimSpace(tok)
	tok = strings.ReplaceAll(tok, " ", "")
	if tok == "" {
		return CodeRange{}, fmt.Errorf("空 token")
	}
	if strings.Contains(tok, "-") {
		parts := strings.Split(tok, "-")
		if len(parts) != 2 {
			return CodeRange{}, fmt.Errorf("非法区间: %s", tok)
		}
		s, err1 := strconv.Atoi(parts[0])
		e, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return CodeRange{}, fmt.Errorf("非法区间: %s", tok)
		}
		if s > e {
			return CodeRange{}, fmt.Errorf("区间起点大于终点: %s", tok)
		}
		if s < 100 || e > 599 {
			return CodeRange{}, fmt.Errorf("区间越界(100-599): %s", tok)
		}
		return CodeRange{Start: s, End: e}, nil
	}
	code, err := strconv.Atoi(tok)
	if err != nil {
		return CodeRange{}, fmt.Errorf("非法状态码: %s", tok)
	}
	if code < 100 || code > 599 {
		return CodeRange{}, fmt.Errorf("状态码越界(100-599): %s", tok)
	}
	return CodeRange{Start: code, End: code}, nil
}

// mergeRanges 排序 + 合并相邻（差 1 视为相邻）与重叠区间，与上游一致。
func mergeRanges(rs []CodeRange) []CodeRange {
	if len(rs) == 0 {
		return nil
	}
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].Start == rs[j].Start {
			return rs[i].End < rs[j].End
		}
		return rs[i].Start < rs[j].Start
	})
	merged := []CodeRange{rs[0]}
	for _, r := range rs[1:] {
		last := &merged[len(merged)-1]
		if r.Start <= last.End+1 {
			if r.End > last.End {
				last.End = r.End
			}
			continue
		}
		merged = append(merged, r)
	}
	return merged
}

// SerializeRanges 规范序列化（单码直出、区间 "a-b"），与上游 statusCodeRangesToString 一致。
func SerializeRanges(rs []CodeRange) string {
	parts := make([]string, 0, len(rs))
	for _, r := range rs {
		if r.Start == r.End {
			parts = append(parts, strconv.Itoa(r.Start))
			continue
		}
		parts = append(parts, fmt.Sprintf("%d-%d", r.Start, r.End))
	}
	return strings.Join(parts, ",")
}

// CoveredBy 判断 code 是否被有序区间集覆盖（上游匹配语义）。
func CoveredBy(rs []CodeRange, code int) bool {
	if code < 100 || code > 599 {
		return false
	}
	for _, r := range rs {
		if code < r.Start {
			return false
		}
		if code <= r.End {
			return true
		}
	}
	return false
}

func coveredAll(rs []CodeRange, r CodeRange) bool {
	for c := r.Start; c <= r.End; c++ {
		if !CoveredBy(rs, c) {
			return false
		}
	}
	return true
}

func overlapsAny(rs []CodeRange, r CodeRange) bool {
	for _, x := range rs {
		if x.End >= r.Start && x.Start <= r.End {
			return true
		}
	}
	return false
}

// subtract 从有序区间集中挖掉 cut（必要时拆分区间），结果保持有序。
func subtract(rs []CodeRange, cut CodeRange) []CodeRange {
	out := make([]CodeRange, 0, len(rs)+2)
	for _, r := range rs {
		if r.End < cut.Start || r.Start > cut.End {
			out = append(out, r)
			continue
		}
		if r.Start < cut.Start {
			out = append(out, CodeRange{Start: r.Start, End: cut.Start - 1})
		}
		if r.End > cut.End {
			out = append(out, CodeRange{Start: cut.End + 1, End: r.End})
		}
	}
	return out
}

// CodesResult 是一次状态码增删改的结果。
type CodesResult struct {
	Action   string      `json:"action"`
	Target   string      `json:"target"`
	Raw      string      `json:"raw_before"`
	Codes    string      `json:"codes_after"`
	Ranges   []CodeRange `json:"ranges"`
	Added    []string    `json:"added,omitempty"`
	Removed  []string    `json:"removed,omitempty"`
	Already  []string    `json:"already_covered,omitempty"`
	NotFound []string    `json:"not_found,omitempty"`
}

// StatusCodesList 查询当前状态码配置（读上游 option 并本地解析）。
func StatusCodesList(ctx context.Context, c *newapi.Client, target string) (*CodesResult, error) {
	key, ok := StatusCodeKey(target)
	if !ok {
		return nil, fmt.Errorf("target 仅支持 disable|retry")
	}
	raw, err := rawOptionValue(ctx, c, key)
	if err != nil {
		return nil, err
	}
	rs, err := ParseRanges(raw)
	if err != nil {
		return nil, fmt.Errorf("上游现值解析失败（可考虑用 action=set 重写）: %w", err)
	}
	return &CodesResult{Action: "list", Target: target, Raw: raw, Codes: SerializeRanges(rs), Ranges: rs}, nil
}

// StatusCodesModify 按 action（add|remove|set）改写状态码配置并写回上游。
func StatusCodesModify(ctx context.Context, c *newapi.Client, target, action string, tokens []string) (*CodesResult, error) {
	key, ok := StatusCodeKey(target)
	if !ok {
		return nil, fmt.Errorf("target 仅支持 disable|retry")
	}
	tokRs, err := ParseTokenList(tokens)
	if err != nil {
		return nil, err
	}
	raw, err := rawOptionValue(ctx, c, key)
	if err != nil {
		return nil, err
	}
	rs, err := ParseRanges(raw)
	if err != nil {
		return nil, fmt.Errorf("上游现值解析失败，请先用 action=set 全量重写: %w", err)
	}
	res := &CodesResult{Action: action, Target: target, Raw: raw}
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
	default:
		return nil, fmt.Errorf("action 仅支持 list|add|remove|set")
	}
	res.Codes = SerializeRanges(res.Ranges)
	if err := Update(ctx, c, key, res.Codes); err != nil {
		return nil, err
	}
	return res, nil
}

// rawOptionValue 从 option 列表里取单键现值（上游无单键读端点）。
func rawOptionValue(ctx context.Context, c *newapi.Client, key string) (string, error) {
	entries, err := List(ctx, c)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.Key == key {
			return e.Value, nil
		}
	}
	return "", nil // 键不存在视为空配置（上游 option 表未初始化该键）
}
