package options

// options.go 系统设置域（option）。
// 上游契约：controller/option.go（/api/option 路由组，AdminAuth）：
//   List:   GET /api/option/ → data: [{key,value}...]；上游已过滤敏感键
//           （后缀 Token/Secret/Key/secret/api_key）与 theme.frontend，并追加合成键 CompletionRatioMeta。
//   Update: PUT /api/option/ body {key, value}（value 任意 JSON 类型，上游统一转 string 落库）；
//           上游不校验 key 白名单（未知 key 也会写入），调用方必须先 List 确认键名。

import (
	"context"
	"encoding/json"
	"sort"

	"mcp_newapi/internal/newapi"
)

// Entry 是单个系统设置的键值对（裁剪 DTO）。
type Entry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// List 拉取全部系统设置（上游已脱敏），按 key 排序保证输出稳定。
func List(ctx context.Context, c *newapi.Client) ([]Entry, error) {
	data, err := c.Do(ctx, "GET", newapi.RouteOptions, nil, nil)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
	return entries, nil
}

// Update 写入一个 option。value 按字符串传输（上游统一转 string）。
func Update(ctx context.Context, c *newapi.Client, key, value string) error {
	body := map[string]any{"key": key, "value": value}
	_, err := c.Do(ctx, "PUT", newapi.RouteOptions, nil, body)
	return err
}
