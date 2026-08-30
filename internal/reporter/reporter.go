// Package reporter 报表域：直连报表从库（MySQL），聚合基元渠道调用量与消费。
// 与 newapi HTTP 域不同源——数据来自本地从库（newapi.logs + 自定义 model_price_snapshots 表），
// 因此不走 client 传输层，独立为叶子包（凭据经 config 的 db_dsn_file 间接注入，不落仓库）。
package reporter

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL 驱动（sql.Open("mysql", dsn)）
)

// Store 是报表库连接（只读从库）。nil Store 表示未配置，调用方收到「配置不足」错误。
type Store struct{ db *sql.DB }

// Open 打开报表库连接。sql.Open 惰性建连——DSN 格式错误或库不可达都在查询时报错。
// dsn 为空时返回 nil Store（工具保持注册，调用时报配置不足错误）。
func Open(dsn string) *Store {
	if dsn == "" {
		return nil
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil
	}
	return &Store{db: db}
}

// Price 是某模型最近一次价格快照（元 / 1M tokens）。
type Price struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
	Cache  float64 `json:"cache"`
}

// Agg 是一组聚合量（tokens 只计成功请求 type=2，失败不计费）。
type Agg struct {
	Calls    int64   `json:"calls"`
	OK       int64   `json:"ok"`
	Fail     int64   `json:"fail"`
	PTok     float64 `json:"prompt_tokens"`
	CTok     float64 `json:"completion_tokens"`
	CacheTok float64 `json:"cache_tokens"`
	Cost     float64 `json:"cost"`
}

// ChannelAgg 渠道汇总行。
type ChannelAgg struct {
	Channel string `json:"channel"`
	Agg
}

// ModelAgg 按计费模型（映射后 upstream_model_name）汇总行，附三价。
type ModelAgg struct {
	Model string `json:"model"`
	Agg
	Price Price `json:"price"`
}

// DetailAgg 渠道×模型明细行。
type DetailAgg struct {
	Channel string `json:"channel"`
	Model   string `json:"model"`
	Agg
	Price Price `json:"price"`
}

// DailyRow 每日趋势行（消费不在每日列：快照价格是「当前价」，按天算会误导）。
type DailyRow struct {
	Day      string  `json:"day"`
	Calls    int64   `json:"calls"`
	OK       int64   `json:"ok"`
	Fail     int64   `json:"fail"`
	PTok     float64 `json:"prompt_tokens"`
	CTok     float64 `json:"completion_tokens"`
	CacheTok float64 `json:"cache_tokens"`
}

// Report 是消费报表的完整结构化结果。
type Report struct {
	WindowStart   string       `json:"window_start"` // 本地时区 YYYY-MM-DD
	WindowEnd     string       `json:"window_end"`   // 报表生成时刻
	Days          int          `json:"days"`
	NameLike      string       `json:"name_like"`
	Channels      []ChannelAgg `json:"channels"`
	Models        []ModelAgg   `json:"models"`
	Details       []DetailAgg  `json:"details"`
	Daily         []DailyRow   `json:"daily"`
	Total         Agg          `json:"total"`
	MissingPrices []string     `json:"missing_prices,omitempty"` // 有调用但无价格快照的计费模型（按 0 价计）
}

// detailRow 是渠道×模型粒度的原始聚合行（SQL 产出，cost 由 BuildReport 计算）。
type detailRow struct {
	ChannelID   int64
	ChannelName string
	Model       string
	Upstream    string
	Agg
}

// JiyuanReport 聚合指定区间内名称命中 nameLike 的渠道报表。
func (s *Store) JiyuanReport(ctx context.Context, nameLike string, days int) (*Report, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("报表功能未配置：需要在生产配置 [report] 段设置 db_dsn_file（或 db_dsn / 环境变量 NEWAPI_REPORT_DB_DSN）后重启 MCP")
	}
	now := time.Now()
	today0 := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := today0.AddDate(0, 0, -(days - 1))

	details, err := s.queryDetails(ctx, start.Unix(), "%"+nameLike+"%")
	if err != nil {
		return nil, fmt.Errorf("查询日志聚合失败: %w", err)
	}
	prices, err := s.queryPrices(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询价格快照失败: %w", err)
	}
	daily, err := s.queryDaily(ctx, start.Unix(), "%"+nameLike+"%")
	if err != nil {
		return nil, fmt.Errorf("查询每日趋势失败: %w", err)
	}
	return BuildReport(details, prices, daily, start, now, days, nameLike), nil
}

const sqlDetails = `
SELECT c.id, c.name, l.model_name,
       COUNT(*) AS calls,
       SUM(CASE WHEN l.type=2 THEN 1 ELSE 0 END) AS ok,
       SUM(CASE WHEN l.type=5 THEN 1 ELSE 0 END) AS fail,
       SUM(CASE WHEN l.type=2 THEN l.prompt_tokens ELSE 0 END) AS ptok,
       SUM(CASE WHEN l.type=2 THEN l.completion_tokens ELSE 0 END) AS ctok,
       COALESCE(SUM(CASE WHEN l.type=2 THEN JSON_EXTRACT(l.other,'$.cache_tokens') ELSE 0 END),0) AS cachetok,
       COALESCE(JSON_UNQUOTE(JSON_EXTRACT(l.other,'$.upstream_model_name')), l.model_name) AS upstream_model
FROM logs l JOIN channels c ON l.channel_id=c.id
WHERE l.created_at >= ? AND c.name LIKE ?
GROUP BY c.id, c.name, l.model_name, upstream_model`

const sqlPrices = `
SELECT s.model_id, s.input_price, s.output_price, s.cache_price
FROM model_price_snapshots s
JOIN (SELECT model_id, MAX(snapshot_at) AS m FROM model_price_snapshots GROUP BY model_id) t
  ON s.model_id=t.model_id AND s.snapshot_at=t.m`

const sqlDaily = `
SELECT FROM_UNIXTIME(l.created_at,'%m-%d') AS d,
       COUNT(*),
       SUM(CASE WHEN l.type=2 THEN 1 ELSE 0 END),
       SUM(CASE WHEN l.type=5 THEN 1 ELSE 0 END),
       SUM(CASE WHEN l.type=2 THEN l.prompt_tokens ELSE 0 END),
       SUM(CASE WHEN l.type=2 THEN l.completion_tokens ELSE 0 END),
       COALESCE(SUM(CASE WHEN l.type=2 THEN JSON_EXTRACT(l.other,'$.cache_tokens') ELSE 0 END),0)
FROM logs l JOIN channels c ON l.channel_id=c.id
WHERE l.created_at >= ? AND c.name LIKE ?
GROUP BY d ORDER BY d`

func (s *Store) queryDetails(ctx context.Context, startUnix int64, like string) ([]detailRow, error) {
	rows, err := s.db.QueryContext(ctx, sqlDetails, startUnix, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []detailRow
	for rows.Next() {
		var (
			r                    detailRow
			chID                 int64
			calls, ok, fail      int64
			ptok, ctok, cachetok sql.NullFloat64
			upstream             sql.NullString
		)
		if err := rows.Scan(&chID, &r.ChannelName, &r.Model, &calls, &ok, &fail,
			&ptok, &ctok, &cachetok, &upstream); err != nil {
			return nil, err
		}
		r.ChannelID = chID
		r.Calls, r.OK, r.Fail = calls, ok, fail
		r.PTok, r.CTok, r.CacheTok = ptok.Float64, ctok.Float64, cachetok.Float64
		r.Upstream = upstream.String
		if r.Upstream == "" {
			r.Upstream = r.Model
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) queryPrices(ctx context.Context) (map[string]Price, error) {
	rows, err := s.db.QueryContext(ctx, sqlPrices)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	prices := map[string]Price{}
	for rows.Next() {
		var model string
		var pin, pout, pcache sql.NullFloat64
		if err := rows.Scan(&model, &pin, &pout, &pcache); err != nil {
			return nil, err
		}
		prices[model] = Price{Input: pin.Float64, Output: pout.Float64, Cache: pcache.Float64}
	}
	return prices, rows.Err()
}

func (s *Store) queryDaily(ctx context.Context, startUnix int64, like string) ([]DailyRow, error) {
	rows, err := s.db.QueryContext(ctx, sqlDaily, startUnix, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DailyRow
	for rows.Next() {
		var (
			d                    string
			calls, ok, fail      int64
			ptok, ctok, cachetok sql.NullFloat64
		)
		if err := rows.Scan(&d, &calls, &ok, &fail, &ptok, &ctok, &cachetok); err != nil {
			return nil, err
		}
		out = append(out, DailyRow{Day: d, Calls: calls, OK: ok, Fail: fail,
			PTok: ptok.Float64, CTok: ctok.Float64, CacheTok: cachetok.Float64})
	}
	return out, rows.Err()
}

// BuildReport 纯聚合：明细行 × 价格表 → 完整报表（可独立单测，不碰 DB）。
// 消费 = (Prompt−缓存)/1M×输入价 + 缓存/1M×缓存价 + Completion/1M×输出价；
// 价格按计费模型（upstream_model_name，回退 model_name）取最近快照，缺价按 0 并记入 MissingPrices。
func BuildReport(details []detailRow, prices map[string]Price, daily []DailyRow,
	start, end time.Time, days int, nameLike string) *Report {
	rep := &Report{
		WindowStart: start.Format("2006-01-02"),
		WindowEnd:   end.Format("2006-01-02 15:04"),
		Days:        days,
		NameLike:    nameLike,
		Daily:       daily,
	}
	add := func(dst *Agg, r detailRow, cost float64) {
		dst.Calls += r.Calls
		dst.OK += r.OK
		dst.Fail += r.Fail
		dst.PTok += r.PTok
		dst.CTok += r.CTok
		dst.CacheTok += r.CacheTok
		dst.Cost += cost
	}
	chAgg := map[string]*Agg{}
	mdAgg := map[string]*Agg{}
	dtAgg := map[string]*DetailAgg{}
	pricesSeen := map[string]bool{}
	for _, r := range details {
		price, ok := prices[r.Upstream]
		if !ok {
			price, ok = prices[r.Model]
		}
		if !ok {
			price = Price{}
			if !pricesSeen[r.Upstream] {
				rep.MissingPrices = append(rep.MissingPrices, r.Upstream)
				pricesSeen[r.Upstream] = true
			}
		}
		nonCache := r.PTok - r.CacheTok
		if nonCache < 0 {
			nonCache = 0
		}
		cost := nonCache/1e6*price.Input + r.CacheTok/1e6*price.Cache + r.CTok/1e6*price.Output
		r2 := r
		r2.Cost = cost
		if chAgg[r.ChannelName] == nil {
			chAgg[r.ChannelName] = &Agg{}
		}
		add(chAgg[r.ChannelName], r2, cost)
		if mdAgg[r.Upstream] == nil {
			mdAgg[r.Upstream] = &Agg{}
		}
		add(mdAgg[r.Upstream], r2, cost)
		key := r.ChannelName + "|" + r.Model
		if dtAgg[key] == nil {
			dtAgg[key] = &DetailAgg{Channel: r.ChannelName, Model: r.Model, Price: price}
		}
		add(&dtAgg[key].Agg, r2, cost)
		add(&rep.Total, r2, cost)
	}
	for name, a := range chAgg {
		rep.Channels = append(rep.Channels, ChannelAgg{Channel: name, Agg: *a})
	}
	sort.Slice(rep.Channels, func(i, j int) bool { return rep.Channels[i].Calls > rep.Channels[j].Calls })
	for m, a := range mdAgg {
		price := prices[m]
		rep.Models = append(rep.Models, ModelAgg{Model: m, Agg: *a, Price: price})
	}
	sort.Slice(rep.Models, func(i, j int) bool { return rep.Models[i].Calls > rep.Models[j].Calls })
	for _, d := range dtAgg {
		rep.Details = append(rep.Details, *d)
	}
	sort.Slice(rep.Details, func(i, j int) bool { return rep.Details[i].Calls > rep.Details[j].Calls })
	sort.Strings(rep.MissingPrices)
	return rep
}
