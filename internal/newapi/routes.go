package newapi

// routes.go 是 new-api 端点路径的唯一耦合点。
// new-api 迭代较快，端点漂移只改本文件。
const (
	RouteStatus   = "/api/status"   // 公开：站点状态
	RouteModels   = "/api/models"   // 公开：可用模型列表
	RoutePricing  = "/api/pricing"  // 模型倍率定价（实例可能禁用）
	RouteChannels = "/api/channel/" // 渠道列表（管理员）
	RouteData     = "/api/data/"    // dashboard 按模型聚合数据
	RouteOptions  = "/api/option/"  // 系统设置（管理员）：GET 列表（上游已脱敏）/ PUT 更新 {key,value}

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

	// admin 档（渠道 CRUD）
	// RouteChannels 复用：POST /api/channel/ 创建、PUT /api/channel/ 更新（PATCH 语义，含 tag 字段）
	// RouteChannelDetail 复用：DELETE 删渠道
	RouteChannelTag        = "/api/channel/tag"          // PUT 按 tag 批量编辑（new_tag/priority/weight/models/model_mapping/groups）
	RouteChannelTagEnable  = "/api/channel/tag/enabled"  // POST 按 tag 批量启用
	RouteChannelTagDisable = "/api/channel/tag/disabled" // POST 按 tag 批量禁用

	// admin 档（系统设置）
	// RouteOptions 复用：GET 列表（上游已脱敏）、PUT 更新 {key,value}
)
