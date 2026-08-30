// Package config 是独立的配置模块：TOML 配置文件 + 环境变量覆盖 + 内置默认值。
// 优先级：内置默认值 < TOML 配置文件 < 环境变量。
// token 支持 token_file 间接引用（读文件首行），避免密钥落进配置文件本体。
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// WriteMode 权限档位（与 internal/mcp 的分档注册对应）。
const (
	ModeRead  = "read"
	ModeOps   = "ops"
	ModeAdmin = "admin"
)

// Config 是根配置。
type Config struct {
	NewAPI NewAPIConfig `toml:"newapi"`
	Report ReportConfig `toml:"report"`
}

// ReportConfig 是报表从库配置（密钥经 db_dsn_file 间接引用，避免落配置文件本体）。
type ReportConfig struct {
	DBDSN     string `toml:"db_dsn"`      // 直连 DSN（user:pass@tcp(host:port)/db?charset=utf8mb4）
	DBDSNFile string `toml:"db_dsn_file"` // DSN 文件路径（读首行，0600；推荐）
}

// ReportDSN 解析报表库 DSN：env NEWAPI_REPORT_DB_DSN > db_dsn > db_dsn_file（首行）。空串=未配置。
func (c *Config) ReportDSN() string {
	if c.Report.DBDSN != "" {
		return strings.TrimSpace(c.Report.DBDSN)
	}
	if c.Report.DBDSNFile != "" {
		b, err := os.ReadFile(c.Report.DBDSNFile)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0])
	}
	return ""
}

// NewAPIConfig 是 new-api 网关连接与行为配置。
type NewAPIConfig struct {
	BaseURL      string `toml:"base_url"`       // 网关地址，如 https://newapi.ashou.site
	Token        string `toml:"token"`          // 面板 PAT（建议留空，用 token_file）
	TokenFile    string `toml:"token_file"`     // PAT 文件路径（读首行，0600）
	WriteMode    string `toml:"writemode"`      // read / ops / admin
	TimeoutSec   int    `toml:"timeout_seconds"` // HTTP 超时秒数
}

// Defaults 返回内置默认值。
func Defaults() *Config {
	return &Config{
		NewAPI: NewAPIConfig{
			WriteMode:  ModeRead,
			TimeoutSec: 10,
		},
	}
}

// Load 装配最终配置：默认值 → TOML 文件（path 为空则跳过）→ 环境变量 → token_file → 校验。
func Load(path string) (*Config, error) {
	cfg := Defaults()
	if path != "" {
		if _, err := toml.DecodeFile(path, cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件 %s: %w", path, err)
		}
	}
	cfg.applyEnv()
	if err := cfg.resolveTokenFile(); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// applyEnv 环境变量覆盖（非空才覆盖）。
func (c *Config) applyEnv() {
	if v := os.Getenv("NEWAPI_BASE_URL"); v != "" {
		c.NewAPI.BaseURL = strings.TrimRight(v, "/")
	}
	if v := os.Getenv("NEWAPI_TOKEN"); v != "" {
		c.NewAPI.Token = strings.TrimSpace(v)
	}
	if v := os.Getenv("NEWAPI_WRITEMODE"); v != "" {
		c.NewAPI.WriteMode = v
	}
	if v := os.Getenv("NEWAPI_TIMEOUT"); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			c.NewAPI.TimeoutSec = sec
		}
	}
	if v := os.Getenv("NEWAPI_REPORT_DB_DSN"); v != "" {
		c.Report.DBDSN = strings.TrimSpace(v)
	}
}

// resolveTokenFile 若配置了 token_file 且 token 为空，读文件首行作为 token。
func (c *Config) resolveTokenFile() error {
	if c.NewAPI.Token != "" || c.NewAPI.TokenFile == "" {
		return nil
	}
	b, err := os.ReadFile(c.NewAPI.TokenFile)
	if err != nil {
		return fmt.Errorf("读取 token_file %s: %w", c.NewAPI.TokenFile, err)
	}
	line := strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0])
	if line == "" {
		return fmt.Errorf("token_file %s 首行为空", c.NewAPI.TokenFile)
	}
	c.NewAPI.Token = line
	return nil
}

// Validate 校验最终配置。
func (c *Config) Validate() error {
	if c.NewAPI.BaseURL == "" {
		return fmt.Errorf("base_url 未配置（TOML 或 NEWAPI_BASE_URL）")
	}
	switch c.NewAPI.WriteMode {
	case ModeRead, ModeOps, ModeAdmin:
	default:
		return fmt.Errorf("writemode 非法: %q（应为 read/ops/admin）", c.NewAPI.WriteMode)
	}
	if c.NewAPI.TimeoutSec <= 0 {
		c.NewAPI.TimeoutSec = 10
	}
	return nil
}
