package handler

// read_test.go autoban 工具纯函数测试（不触网）。

import (
	"strings"
	"testing"

	"mcp_newapi/internal/newapi/logs"
)

func TestIsAutobanOptionKey(t *testing.T) {
	yes := []string{
		"AutomaticDisableChannelEnabled", "AutomaticDisableStatusCodes",
		"AutomaticRetryStatusCodes", "AutomaticDisableKeywords", "AutomaticEnableChannelEnabled",
		"ChannelDisableThreshold", "RetryTimes",
		"monitor_setting.channel_test_mode", "monitor_setting.auto_test_channel_enabled",
		"channel_affinity_setting.keep_on_channel_disabled",
	}
	for _, k := range yes {
		if !isAutobanOptionKey(k) {
			t.Errorf("%s 应命中 autoban 键筛选", k)
		}
	}
	no := []string{"QuotaRemindThreshold", "performance_setting.monitor_cpu_threshold", "SystemName", "CompletionRatioMeta", "LogConsumeEnabled"}
	for _, k := range no {
		if isAutobanOptionKey(k) {
			t.Errorf("%s 不应命中 autoban 键筛选", k)
		}
	}
}

func TestErrStats(t *testing.T) {
	entries := []logs.Entry{
		{Content: "余额不足 (402)", ModelName: "m1", CreatedAt: 100},
		{Content: "余额不足 (402)", ModelName: "m1", CreatedAt: 200},
		{Content: "模型已关闭：xxx", ModelName: "m2", CreatedAt: 150},
		{Content: "", ModelName: "", CreatedAt: 50},
	}
	byContent, byModel, lastAt := errStats(entries, 5)
	if lastAt != 200 {
		t.Errorf("last_error_at = %d, want 200", lastAt)
	}
	if len(byContent) != 3 {
		t.Fatalf("by_content 组数 = %d, want 3", len(byContent))
	}
	if byContent[0]["content"] != "余额不足 (402)" || byContent[0]["count"] != 2 {
		t.Errorf("top1 应为「余额不足 (402)」×2, got %v", byContent[0])
	}
	if len(byModel) != 2 || byModel[0]["model"] != "m1" || byModel[0]["count"] != 2 {
		t.Errorf("by_model top1 应为 m1×2, got %v", byModel)
	}
	// topN 截断
	_, bm2, _ := errStats(entries, 1)
	if len(bm2) != 1 {
		t.Errorf("topN=1 应截断到 1 组, got %d", len(bm2))
	}
}

func TestClassifyBanCause(t *testing.T) {
	cases := []struct {
		contents []string
		want     string
	}{
		{[]string{"上游返回 402 Payment Required"}, "quota_exhausted"},
		{[]string{"Your credit balance is too low"}, "quota_exhausted"},
		{[]string{"模型已关闭：deepseek-v4-pro"}, "model_issue"},
		{[]string{"context deadline exceeded"}, "timeout"},
		{[]string{"dial tcp 1.2.3.4:443: connection refused"}, "upstream_unreachable"},
		{[]string{"奇怪的报错"}, "other"},
	}
	for _, c := range cases {
		if got := classifyBanCause(c.contents); got != c.want {
			t.Errorf("classify(%v) = %s, want %s", c.contents, got, c.want)
		}
	}
	// 大小写不敏感
	if got := classifyBanCause([]string{"QUOTA EXCEEDED"}); got != "quota_exhausted" {
		t.Errorf("大写关键词应命中, got %s", got)
	}
	if !strings.Contains("quota_exhausted", "quota") {
		t.Fatal("sanity")
	}
}
