package newapi

// routes.go 是 new-api 端点路径的唯一耦合点。
// new-api 迭代较快，端点漂移只改本文件。
const (
	RouteStatus   = "/api/status"   // 公开：站点状态
	RouteModels   = "/api/models"   // 公开：可用模型列表
	RoutePricing  = "/api/pricing"  // 模型倍率定价
	RouteChannels = "/api/channel/" // 渠道列表（管理员）
	// M2/M3 扩展：
	// RouteChannelDetail = "/api/channel/%d"
	// RouteChannelTest   = "/api/channel/test/%d"
	// RouteChannelBal    = "/api/channel/update_balance/%d"
	// RouteTokens        = "/api/token/"
	// RouteLogs          = "/api/log/"
	// RouteLogStat       = "/api/log/stat"
)
