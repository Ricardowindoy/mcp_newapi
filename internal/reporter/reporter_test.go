package reporter

import (
	"testing"
	"time"
)

func TestBuildReportCostAndAggregation(t *testing.T) {
	// 两个渠道、两个计费模型；glm-5.3 无价格快照（缺价按 0）
	details := []detailRow{
		{ChannelID: 1, ChannelName: "基元2_1", Model: "deepseek-v4-flash", Upstream: "deepseek-v4-flash-0731",
			Agg: Agg{Calls: 10, OK: 9, Fail: 1, PTok: 2_000_000, CTok: 1_000_000, CacheTok: 500_000}},
		{ChannelID: 2, ChannelName: "基元3_1", Model: "glm-5.3", Upstream: "glm-5.3",
			Agg: Agg{Calls: 5, OK: 5, Fail: 0, PTok: 1_000_000, CTok: 500_000, CacheTok: 0}},
	}
	prices := map[string]Price{
		"deepseek-v4-flash-0731": {Input: 6, Output: 18, Cache: 1.2},
		// glm-5.3 故意缺价
	}
	start := time.Date(2026, 8, 27, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 0, 4)
	rep := BuildReport(details, prices, nil, start, end, 4, "基元")

	// 消费公式：非缓存输入 × 输入价 + 缓存 × 缓存价 + 输出 × 输出价
	// 0731: (2M-0.5M)/1M*6 + 0.5M/1M*1.2 + 1M/1M*18 = 9 + 0.6 + 18 = 27.6
	if got := rep.Total.Cost; got < 27.5999 || got > 27.6001 {
		t.Fatalf("total cost = %v, want 27.6", got)
	}
	if rep.Total.Calls != 15 || rep.Total.OK != 14 || rep.Total.Fail != 1 {
		t.Fatalf("total agg = %+v", rep.Total)
	}
	// 渠道汇总排序按调用数降序
	if len(rep.Channels) != 2 || rep.Channels[0].Channel != "基元2_1" {
		t.Fatalf("channels = %+v", rep.Channels)
	}
	// 模型汇总按计费模型名
	if len(rep.Models) != 2 || rep.Models[0].Model != "deepseek-v4-flash-0731" {
		t.Fatalf("models = %+v", rep.Models)
	}
	// glm-5.3 缺价 → 记入 MissingPrices 且价格为 0
	if len(rep.MissingPrices) != 1 || rep.MissingPrices[0] != "glm-5.3" {
		t.Fatalf("missing = %+v", rep.MissingPrices)
	}
	for _, m := range rep.Models {
		if m.Model == "glm-5.3" && m.Price.Input != 0 {
			t.Fatalf("missing price should be zero: %+v", m)
		}
	}
	// 缓存超过 Prompt 时非缓存输入不为负
	d2 := []detailRow{{ChannelID: 1, ChannelName: "基元2_1", Model: "m", Upstream: "m",
		Agg: Agg{Calls: 1, OK: 1, PTok: 100, CTok: 0, CacheTok: 500}}}
	rep2 := BuildReport(d2, map[string]Price{"m": {Input: 6}}, nil, start, end, 4, "基元")
	if rep2.Total.Cost != 0 {
		t.Fatalf("negative non-cache should clamp to 0, cost = %v", rep2.Total.Cost)
	}
}
