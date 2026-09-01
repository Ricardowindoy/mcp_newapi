package mcp

// registry.go 是本项目所有对外提供的服务（MCP 工具）的唯一汇总表。
// 表驱动注册：新增/调整工具只改本文件的表 + 对应 handler 实现，server.go 不动。
//
// 表项四要素：Name / Tier / 声明（描述+参数） / Handler 工厂。
// handler 实现按档位放在 tools_read.go / tools_ops.go / tools_admin.go。

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp_newapi/internal/mcp/handler"
	"mcp_newapi/internal/newapi"
	"mcp_newapi/internal/reporter"
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
	Options []mcp.ToolOption // 描述 + 参数声明
	Handler func(*newapi.Client, *reporter.Store) handler.Handler // handler 工厂（实现在 handler/ 包）
}

// toolRegistry —— 对外服务汇总表（唯一索引）。
var toolRegistry = []toolDef{
	// ============ read 档（13） ============
	{
		Name: "newapi_status", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("获取 new-api 网关状态：版本、启动时间、注册开关，并探测 relay (/v1/models) 活性。无需鉴权。"),
		},
		Handler: handler.StatusHandler,
	},
	{
		Name: "newapi_list_models", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("获取网关全站可用模型列表（按分组返回）。无需鉴权。"),
		},
		Handler: handler.ModelsHandler,
	},
	{
		Name: "newapi_list_channels", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("分页获取渠道列表（需管理员 PAT）：id、名称、类型、状态、余额、模型、分组、响应延迟。key 已掩码。"),
			mcp.WithNumber("page", mcp.Description("页码，从 1 开始"), mcp.DefaultNumber(1)),
			mcp.WithNumber("page_size", mcp.Description("每页条数"), mcp.DefaultNumber(20)),
			mcp.WithNumber("status", mcp.Description("状态过滤：0=全部 1=启用 2=禁用"), mcp.DefaultNumber(0)),
		},
		Handler: handler.ChannelsHandler,
	},
	{
		Name: "newapi_get_channel", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("获取单个渠道详情（需管理员 PAT）。key 已掩码。"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
		},
		Handler: handler.ChannelDetailHandler,
	},
	{
		Name: "newapi_list_tokens", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("获取当前用户的令牌（sk- key）列表：名称、状态、用量、剩余额度。key 已掩码。"),
			mcp.WithNumber("page", mcp.Description("页码，从 1 开始"), mcp.DefaultNumber(1)),
			mcp.WithNumber("page_size", mcp.Description("每页条数"), mcp.DefaultNumber(20)),
		},
		Handler: handler.TokensHandler,
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
		Handler: handler.LogsHandler,
	},
	{
		Name: "newapi_usage_summary", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("近 N 天用量汇总：按模型聚合的调用次数/tokens/消费额（quota 及美元，500000 quota=$1）。"),
			mcp.WithNumber("days", mcp.Description("统计天数，默认 7，最大 365"), mcp.DefaultNumber(7)),
		},
		Handler: handler.UsageSummaryHandler,
	},
	{
		Name: "newapi_pricing", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("获取模型倍率定价（注意：实例可能禁用此端点，报错即禁用）。"),
			mcp.WithString("model", mcp.Description("按模型名过滤")),
		},
		Handler: handler.PricingHandler,
	},
	{
		Name: "newapi_list_options", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("列出 new-api 系统设置 option（需管理员 PAT）：键值对，按 key 排序。上游已过滤敏感键（*Token/*Secret/*Key 等后缀）。用于查 autoban 开关（AutomaticDisableChannelEnabled）、禁用状态码（AutomaticDisableStatusCodes）、禁用关键词（AutomaticDisableKeywords）等运营配置当前值。"),
		},
		Handler: handler.ListOptionsHandler,
	},
	{
		Name: "newapi_success_rate", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("查询请求成功率（基于日志数据：type=2 消费条目 vs type=5 错误条目的 total 计数）。时间窗默认近 24h（hours 1-720，或显式传起止时间戳）；可按渠道/模型/令牌过滤。上游重试会产生多条错误日志，比率为近似值。"),
			mcp.WithNumber("hours", mcp.Description("统计窗口小时数，默认 24（1-720）；传了起止时间戳则忽略"), mcp.DefaultNumber(24)),
			mcp.WithNumber("start_timestamp", mcp.Description("起始 Unix 时间戳（秒），与 end_timestamp 成对使用")),
			mcp.WithNumber("end_timestamp", mcp.Description("结束 Unix 时间戳（秒）")),
			mcp.WithNumber("channel", mcp.Description("按渠道 ID 过滤")),
			mcp.WithString("model_name", mcp.Description("按模型过滤")),
			mcp.WithString("token_name", mcp.Description("按令牌名过滤")),
		},
		Handler: handler.SuccessRateHandler,
	},
	{
		Name: "newapi_autoban_config", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("autoban 配置一次性总览（只读，需管理员 PAT）：全局开关 AutomaticDisableChannelEnabled、禁用/重试状态码、禁用关键词、monitor_setting.* 测试模式与阈值、ChannelDisableThreshold，并普查渠道级 auto_ban（on/off/unset 计数 + 未开启渠道清单——NULL 会被上游视为关，是『ban 不生效』的常见根因）。写入口：全局开关与关键词用 newapi_update_option、状态码用 newapi_autoban_codes、渠道级用 newapi_update_channel(auto_ban)。"),
		},
		Handler: handler.AutobanConfigHandler,
	},
	{
		Name: "newapi_autoban_analysis", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("自动封禁原因分析数据获取（只读，需管理员 PAT）：默认分析所有 status=3 自动禁用或带 status_reason 的渠道（可传 channel 指定任意渠道）；拉时间窗内 type=5 错误日志，精确错误总数 + 近期采样按错误内容/模型聚合，附 status_reason/balance/auto_ban/test_model 与 likely_cause 启发式分类（quota/model/timeout/unreachable）。排查『为什么被自动封禁』用这个。"),
			mcp.WithNumber("channel", mcp.Description("指定渠道 ID（缺省=全部自动禁用/带 status_reason 渠道）")),
			mcp.WithNumber("hours", mcp.Description("错误日志回溯窗口小时数，默认 24（1-720）"), mcp.DefaultNumber(24)),
			mcp.WithNumber("sample", mcp.Description("每渠道错误采样条数，默认 10（1-50）"), mcp.DefaultNumber(10)),
		},
		Handler: handler.AutobanAnalysisHandler,
	},
	{
		Name: "newapi_jiyuan_report", Tier: tierRead,
		Options: []mcp.ToolOption{
			mcp.WithDescription("基元渠道消费报表（默认区间=今天+前3天）：从报表从库聚合 logs 与 model_price_snapshots 每模型最近价格快照。tokens 只计成功请求(type=2)；消费=(Prompt−缓存)/1M×输入价+缓存/1M×缓存价+Completion/1M×输出价；计费模型用 other.upstream_model_name(映射后)回退 model_name；缺价模型按 0 计并列出。返回渠道汇总/按模型/渠道×模型明细/每日趋势/合计。需配置 [report] 报表库（未配置时调用报错）。"),
			mcp.WithString("name_like", mcp.Description("渠道名过滤关键词（LIKE 模糊匹配），缺省 基元")),
			mcp.WithNumber("days", mcp.Description("统计天数（含今天），默认 4，1-30"), mcp.DefaultNumber(4)),
		},
		Handler: handler.JiyuanReportHandler,
	},

	// ============ ops 档（6） ============
	{
		Name: "newapi_test_channel", Tier: tierOps,
		Options: []mcp.ToolOption{
			mcp.WithDescription("对单个渠道发一次测试请求，返回是否成功、耗时（秒）与错误信息。测试失败是有效结果（success:false），不代表调用出错。"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
			mcp.WithString("model", mcp.Description("测试用的模型名；缺省用渠道默认测试模型")),
		},
		Handler: handler.TestChannelHandler,
	},
	{
		Name: "newapi_test_all_channels", Tier: tierOps,
		Options: []mcp.ToolOption{
			mcp.WithDescription("触发全量渠道测试（异步系统任务），返回 task_id。结果可在面板任务中心或稍后用 newapi_list_channels 观察响应时间/状态。"),
		},
		Handler: handler.TestAllHandler,
	},
	{
		Name: "newapi_update_channel_balance", Tier: tierOps,
		Options: []mcp.ToolOption{
			mcp.WithDescription("刷新单个渠道的余额（部分渠道类型不支持，会返回业务错误）。"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
		},
		Handler: handler.UpdateBalanceHandler,
	},
	{
		Name: "newapi_set_channel_status", Tier: tierOps,
		Options: []mcp.ToolOption{
			mcp.WithDescription("启用或禁用渠道（记录为 manual operation）。返回是否有实际变更。"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
			mcp.WithBoolean("enabled", mcp.Required(), mcp.Description("true=启用 false=禁用")),
		},
		Handler: handler.SetChannelStatusHandler,
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
		Handler: handler.CreateTokenHandler,
	},
	{
		Name: "newapi_delete_token", Tier: tierOps,
		Options: []mcp.ToolOption{
			mcp.WithDescription("删除当前用户的 API 令牌（不可恢复）。必须显式传 confirm=true。"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("令牌 ID")),
			mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("必须为 true 才执行删除")),
		},
		Handler: handler.DeleteTokenHandler,
	},

	// ============ admin 档（6） ============
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
		Handler: handler.CreateChannelHandler,
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
			mcp.WithBoolean("auto_ban", mcp.Description("渠道级自动禁用开关；true=该渠道失败可被自动禁用（上游 auto_ban=1），false=关闭。缺省不修改")),
			mcp.WithString("tag", mcp.Description("渠道标签；传空串清除，缺省不修改")),
			mcp.WithNumber("type", mcp.Description("渠道类型")),
		},
		Handler: handler.UpdateChannelHandler,
	},
	{
		Name: "newapi_delete_channel", Tier: tierAdmin,
		Options: []mcp.ToolOption{
			mcp.WithDescription("删除渠道（admin，不可恢复）。必须显式传 confirm=true。"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("渠道 ID")),
			mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("必须为 true 才执行删除")),
		},
		Handler: handler.DeleteChannelHandler,
	},
	{
		Name: "newapi_update_option", Tier: tierAdmin,
		Options: []mcp.ToolOption{
			mcp.WithDescription("修改 new-api 系统设置 option（admin，危险操作：全局生效、key 不做白名单校验，误改影响整个网关）。必须显式传 confirm=true。值均为字符串：布尔传 \"true\"/\"false\"，数字传字符串（如阈值 \"20\"、状态码 \"401,402,429\"）。"),
			mcp.WithString("key", mcp.Required(), mcp.Description("option 键名（先用 newapi_list_options 查现有键，勿凭记忆造键）")),
			mcp.WithString("value", mcp.Required(), mcp.Description("新值（字符串形态）")),
			mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("必须为 true 才执行修改")),
		},
		Handler: handler.UpdateOptionHandler,
	},
	{
		Name: "newapi_autoban_codes", Tier: tierAdmin,
		Options: []mcp.ToolOption{
			mcp.WithDescription("autoban 状态码的增删查改（写 AutomaticDisableStatusCodes / AutomaticRetryStatusCodes，保留其余配置）。action=list 查询现值；add=追加（自动排序合并相邻区间、跳过已覆盖项）；remove=移除（必要时拆分包含它的区间，未覆盖项报 not_found）；set=全量重写为规范形。target=disable(默认)|retry。codes 逗号分隔 token：单码或区间，如 402,400-499（范围 100-599）。变更项需 confirm=true。"),
			mcp.WithString("action", mcp.Description("list|add|remove|set，缺省 list")),
			mcp.WithString("target", mcp.Description("disable(自动禁用，默认)|retry(自动重试)")),
			mcp.WithString("codes", mcp.Description("状态码 token，逗号分隔（add/remove/set 必填）")),
			mcp.WithBoolean("confirm", mcp.Description("变更项必须为 true")),
		},
		Handler: handler.AutobanCodesHandler,
	},
	{
		Name: "newapi_tag_channels", Tier: tierAdmin,
		Options: []mcp.ToolOption{
			mcp.WithDescription("按标签批量操作渠道（admin，影响该 tag 下所有渠道）。action=edit 批量改字段（new_tag/priority/weight/models/model_mapping/group，至少一项）；enable/disable 批量启停。必须显式传 confirm=true。单渠道打标签/清标签用 newapi_update_channel 的 tag 字段。"),
			mcp.WithString("action", mcp.Required(), mcp.Description("edit|enable|disable")),
			mcp.WithString("tag", mcp.Required(), mcp.Description("目标标签")),
			mcp.WithString("new_tag", mcp.Description("edit：重命名标签（不能为空）")),
			mcp.WithNumber("priority", mcp.Description("edit：批量改优先级")),
			mcp.WithNumber("weight", mcp.Description("edit：批量改权重")),
			mcp.WithString("models", mcp.Description("edit：批量改模型列表，逗号分隔")),
			mcp.WithString("model_mapping", mcp.Description("edit：批量改模型重定向 JSON")),
			mcp.WithString("group", mcp.Description("edit：批量改分组，逗号分隔")),
			mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("必须为 true 才执行")),
		},
		Handler: handler.TagChannelsHandler,
	},
}

// registerTools 表驱动注册：按 writemode 决定注册到哪一档（低档不含高档工具）。
// rep 是报表从库连接（entry 从 config 注入；nil=报表未配置，工具仍注册、调用时报配置错误）。
func registerTools(s *server.MCPServer, client *newapi.Client, writemode string, rep *reporter.Store) {
	max := tierRead
	switch writemode {
	case "admin":
		max = tierAdmin
	case "ops":
		max = tierOps
	}
	for i := range toolRegistry {
		def := &toolRegistry[i]
		if def.Tier > max {
			continue
		}
		s.AddTool(mcp.NewTool(def.Name, def.Options...), def.Handler(client, rep))
	}
}
