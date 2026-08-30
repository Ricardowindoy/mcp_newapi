package newapi

// routes.go 是 new-api 端点路径的唯一耦合点。
// new-api 迭代较快，端点漂移只改本文件。
const (
	RouteStatus   = "/api/status"   // 公开：站点状态
	RouteModels   = "/api/models"   // 公开：可用模型列表
	RoutePricing  = "/api/pricing"  // 模型倍率定价（实例可能禁用）
	RouteChannels = "/api/channel/" // 渠道列表（管理员）
	RouteData     = "/api/data/"    // dashboard 按模型聚合数据

	RouteChannelDetail = "/api/channel/%d" // 单渠道详情
	RouteTokens        = "/api/token/"     // 令牌列表
	RouteLogs          = "/api/log/"       // 日志列表
	RouteLogStat       = "/api/log/stat"   // 日志总量统计

	// ops 档
	RouteChannelTestAll = "/api/channel/test"           // 触发全量渠道测试（异步系统任务）
	RouteChannelTest    = "/api/channel/test/%d"        // 测试单渠道
	RouteChannelBalAll  = "/api/channel/update_balance" // 全量刷新余额
	RouteChannelBal     = "/api/channel/update_balance/%d"
	RouteChannelStatus  = "/api/channel/%d/status" // 启停渠道 {status:1|2}
	RouteTokenSearch    = "/api/token/search"      // 按关键字搜令牌
	RouteTokenID        = "/api/token/%d"          // 删除令牌
)
