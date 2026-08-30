package newapi

// page.go 列表分页的共享类型与查询参数小工具。
// 各域子包用它组装分页响应；上游统一分页壳 {page,page_size,total,items}。

import "strconv"

// Paged 是上游列表端点的统一分页壳（原始 DTO 用）。
type Paged[T any] struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
	Items    []T `json:"items"`
}

// PageResult 是对外（MCP 层）的分页结果。
type PageResult[T any] struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
	Items    []T `json:"items"`
}

// Itoa 页码类参数：v<=0 时用 def。
func Itoa(v, def int) string {
	if v <= 0 {
		v = def
	}
	return strconv.Itoa(v)
}

// Itoa64 时间戳类参数直转。
func Itoa64(v int64) string {
	return strconv.FormatInt(v, 10)
}
