package mcp

// registry.go 是本项目所有对外提供的服务（MCP 工具）的唯一汇总表。
// 表驱动注册：新增/调整工具只改本文件的表 + 对应 handler 实现，server.go 不动。
//
// 表项四要素：Name / Tier / 声明（描述+参数） / Handler 工厂。
// handler 实现按档位放在 tools_read.go / tools_ops.go / tools_admin.go。

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp_newapi/internal/newapi"
)

// tier 工具档位（与 config 的 writemode 对应）。
type tier int

const (
	tierRead  tier = 0 // 始终注册
	tierOps   tier = 1 // writemode >= ops
	tierAdmin tier = 2 // writemode == admin
)

// toolDef 是一个对外工具的完整声明。
type toolDef struct {
	Name    string
	Tier    tier
	Options []mcp.ToolOption           // 描述 + 参数声明
	Handler func(*newapi.Client) toolHandler // handler 工厂
}

// toolRegistry —— 对外服务汇总表（唯一索引）。
var toolRegistry = []toolDef{
	// ============ read 档（8） ============
	{
		Name: "newapi_status", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("获取 new-api 网关状态：版本、启动时间、注册开关，并探测 relay (/v1/models) 活性。无需鉴权。"),
		},
		Handler: statusHandler,
	},
	{
		Name: "newapi_list_models", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("获取网关全站可用模型列表（按分组返回）。无需鉴权。"),
		},
		Handler: modelsHandler,
	},
	{
		Name: "newapi_list_channels", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("分页获取渠道列表（需管理员 PAT）：id、名称、类型、状态、余额、模型、分组、响应延迟。key 已掩码。"),
			mcp.WithNumber("page", mcp.Description("页码，从 1 开始"), mcp.DefaultNumber(1)),
			mcp.WithNumber("page_size", mcp.Description("每页条数"), mcp.DefaultNumber(20)),
			mcp.WithNumber("status", mcp.Description("状态过滤：0=全部 1=启用 2=禁用"), mcp.DefaultNumber(0)),
		},
		Handler: channelsHandler,
	},
	{
		Name: "newapi_get_channel", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("获取单个渠道详情（需管理员 PAT）。key 已掩码。"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
		},
		Handler: channelDetailHandler,
	},
	{
		Name: "newapi_list_tokens", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("获取当前用户的令牌（sk- key）列表：名称、状态、用量、剩余额度。key 已掩码。"),
			mcp.WithNumber("page", mcp.Description("页码，从 1 开始"), mcp.DefaultNumber(1)),
			mcp.WithNumber("page_size", mcp.Description("每页条数"), mcp.DefaultNumber(20)),
		},
		Handler: tokensHandler,
	},
	{
		Name: "newapi_logs", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("检索网关日志（消费/错误等）。type: 2=消费 5=错误；不传=全部。"),
			mcp.WithNumber("page", mcp.Description("页码，从 1 开始"), mcp.DefaultNumber(1)),
			mcp.WithNumber("page_size", mcp.Description("每页条数"), mcp.DefaultNumber(20)),
			mcp.WithNumber("type", mcp.Description("日志类型：0=全部 2=消费 5=错误")),
			mcp.WithNumber("start_timestamp", mcp.Description("起始 Unix 时间戳（秒）")),
			mcp.WithNumber("end_timestamp", mcp.Description("结束 Unix 时间戳（秒）")),
			mcp.WithString("model_name", mcp.Description("按模型过滤")),
			mcp.WithString("token_name", mcp.Description("按令牌名过滤")),
			mcp.WithNumber("channel", mcp.Description("按渠道 ID 过滤")),
		},
		Handler: logsHandler,
	},
	{
		Name: "newapi_usage_summary", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("近 N 天用量汇总：按模型聚合的调用次数/tokens/消费额（quota 及美元，500000 quota=$1）。"),
			mcp.WithNumber("days", mcp.Description("统计天数，默认 7，最大 365"), mcp.DefaultNumber(7)),
		},
		Handler: usageSummaryHandler,
	},
	{
		Name: "newapi_pricing", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("获取模型倍率定价（注意：实例可能禁用此端点，报错即禁用）。"),
			mcp.WithString("model", mcp.Description("按模型名过滤")),
		},
		Handler: pricingHandler,
	},

	// ============ ops 档（6） ============
	{
		Name: "newapi_test_channel", Tier: tierOps,
		Options: []mcp.ToolOption{
			mcp.WithDescription("对单个渠道发一次测试请求，返回是否成功、耗时（秒）与错误信息。测试失败是有效结果（success:false），不代表调用出错。"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
			mcp.WithString("model", mcp.Description("测试用的模型名；缺省用渠道默认测试模型")),
		},
		Handler: testChannelHandler,
	},
	{
		Name: "newapi_test_all_channels", Tier: tierOps,
		Options: []mcp.ToolOption{
			mcp.WithDescription("触发全量渠道测试（异步系统任务），返回 task_id。结果可在面板任务中心或稍后用 newapi_list_channels 观察响应时间/状态。"),
		},
		Handler: testAllHandler,
	},
	{
		Name: "newapi_update_channel_balance", Tier: tierOps,
		Options: []mcp.ToolOption{
			mcp.WithDescription("刷新单个渠道的余额（部分渠道类型不支持，会返回业务错误）。"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
		},
		Handler: updateBalanceHandler,
	},
	{
		Name: "newapi_set_channel_status", Tier: tierOps,
		Options: []mcp.ToolOption{
			mcp.WithDescription("启用或禁用渠道（记录为 manual operation）。返回是否有实际变更。"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
			mcp.WithBoolean("enabled", mcp.Required(), mcp.Description("true=启用 false=禁用")),
		},
		Handler: setChannelStatusHandler,
	},
	{
		Name: "newapi_create_token", Tier: tierOps,
		Options: []mcp.ToolOption{
			mcp.WithDescription("创建一个 API 令牌（sk- key）。返回 id 与掩码 key；完整 key 请在面板查看。"),
			mcp.WithString("name", mcp.Required(), mcp.Description("令牌名称（≤50 字符）")),
			mcp.WithBoolean("unlimited_quota", mcp.Description("无限额度，默认 false")),
			mcp.WithNumber("remain_quota", mcp.Description("额度（quota 单位，500000=$1）；unlimited 时忽略")),
			mcp.WithNumber("expired_time", mcp.Description("过期 Unix 时间戳（秒）；-1 或缺省=永不过期")),
			mcp.WithString("model_limits", mcp.Description("模型限制，逗号分隔模型名")),
			mcp.WithString("group", mcp.Description("分组，缺省 default")),
		},
		Handler: createTokenHandler,
	},
	{
		Name: "newapi_delete_token", Tier: tierOps,
		Options: []mcp.ToolOption{
			mcp.WithDescription("删除当前用户的 API 令牌（不可恢复）。必须显式传 confirm=true。"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("令牌 ID")),
			mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("必须为 true 才执行删除")),
		},
		Handler: deleteTokenHandler,
	},

	// ============ admin 档（3） ============
	{
		Name: "newapi_create_channel", Tier: tierAdmin,
		Options: []mcp.ToolOption{
			mcp.WithDescription("创建渠道（admin）。需提供名称/类型/key/模型列表；创建后按名称回查返回渠道 id。key 只在此请求中传输。"),
			mcp.WithString("name", mcp.Required(), mcp.Description("渠道名称")),
			mcp.WithNumber("type", mcp.Required(), mcp.Description("渠道类型：1=OpenAI 兼容 等")),
			mcp.WithString("key", mcp.Required(), mcp.Description("上游 API key（仅在创建请求中传输）")),
			mcp.WithString("models", mcp.Required(), mcp.Description("模型列表，逗号分隔")),
			mcp.WithString("base_url", mcp.Description("上游 base URL（OpenAI 兼容网关填此项）")),
			mcp.WithString("group", mcp.Description("分组，逗号分隔，默认 default")),
			mcp.WithString("model_mapping", mcp.Description("模型重定向 JSON，如 {\"alias\":\"real\"}")),
			mcp.WithNumber("priority", mcp.Description("优先级")),
			mcp.WithNumber("weight", mcp.Description("权重")),
			mcp.WithString("test_model", mcp.Description("测试模型名")),
		},
		Handler: createChannelHandler,
	},
	{
		Name: "newapi_update_channel", Tier: tierAdmin,
		Options: []mcp.ToolOption{
			mcp.WithDescription("更新渠道（admin，PATCH 语义：只传要改的字段）。注意：不能改 status（用 newapi_set_channel_status）；key 留空则不修改。"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
			mcp.WithString("name", mcp.Description("新名称")),
			mcp.WithString("key", mcp.Description("新 key（留空不修改；多 key 渠道慎用）")),
			mcp.WithString("models", mcp.Description("模型列表，逗号分隔")),
			mcp.WithString("base_url", mcp.Description("上游 base URL")),
			mcp.WithString("group", mcp.Description("分组")),
			mcp.WithString("model_mapping", mcp.Description("模型重定向 JSON")),
			mcp.WithNumber("priority", mcp.Description("优先级")),
			mcp.WithNumber("weight", mcp.Description("权重")),
			mcp.WithString("test_model", mcp.Description("测试模型名")),
			mcp.WithNumber("type", mcp.Description("渠道类型")),
		},
		Handler: updateChannelHandler,
	},
	{
		Name: "newapi_delete_channel", Tier: tierAdmin,
		Options: []mcp.ToolOption{
			mcp.WithDescription("删除渠道（admin，不可恢复）。必须显式传 confirm=true。"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
			mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("必须为 true 才执行删除")),
		},
		Handler: deleteChannelHandler,
	},
}

// registerTools 表驱动注册：按 writemode 决定注册到哪一档（低档不含高档工具）。
func registerTools(s *server.MCPServer, client *newapi.Client, writemode string) {
	max := tierRead
	switch writemode {
	case "admin":
		max = tierAdmin
	case "ops":
		max = tierOps
	}
	registered := 0
	for i := range toolRegistry {
		def := &toolRegistry[i]
		if def.Tier > max {
			continue
		}
		s.AddTool(mcp.NewTool(def.Name, def.Options...), def.Handler(client))
		registered++
	}
}
