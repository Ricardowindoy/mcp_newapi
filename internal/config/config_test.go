package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTOMLOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.toml")
	content := `
[newapi]
base_url = "https://gw.example/"
writemode = "admin"
timeout_seconds = 30
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.NewAPI.BaseURL != "https://gw.example/" { // 尾斜杠由 client 层裁剪，config 存原值
		t.Errorf("base_url = %q", cfg.NewAPI.BaseURL)
	}
	if cfg.NewAPI.WriteMode != ModeAdmin {
		t.Errorf("writemode = %q", cfg.NewAPI.WriteMode)
	}
	if cfg.NewAPI.TimeoutSec != 30 {
		t.Errorf("timeout = %d", cfg.NewAPI.TimeoutSec)
	}
}

func TestEnvOverridesTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.toml")
	_ = os.WriteFile(path, []byte(`[newapi]
base_url = "https://from-toml.example"
writemode = "read"
`), 0o600)
	t.Setenv("NEWAPI_BASE_URL", "https://from-env.example")
	t.Setenv("NEWAPI_WRITEMODE", "ops")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.NewAPI.BaseURL != "https://from-env.example" {
		t.Errorf("env 应覆盖 TOML, got %q", cfg.NewAPI.BaseURL)
	}
	if cfg.NewAPI.WriteMode != ModeOps {
		t.Errorf("writemode = %q", cfg.NewAPI.WriteMode)
	}
}

func TestTokenFile(t *testing.T) {
	dir := t.TempDir()
	patPath := filepath.Join(dir, "pat")
	if err := os.WriteFile(patPath, []byte("  pat-abc-123\nsecond-line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "c.toml")
	_ = os.WriteFile(path, []byte("[newapi]\nbase_url = \"https://gw.example\"\ntoken_file = \""+patPath+"\"\n"), 0o600)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.NewAPI.Token != "pat-abc-123" {
		t.Errorf("token = %q, want 首行", cfg.NewAPI.Token)
	}
}

func TestValidateErrors(t *testing.T) {
	cfg := Defaults() // 无 base_url
	if err := cfg.Validate(); err == nil {
		t.Error("缺 base_url 应报错")
	}
	cfg.NewAPI.BaseURL = "https://x.example"
	cfg.NewAPI.WriteMode = "sudo"
	if err := cfg.Validate(); err == nil {
		t.Error("非法 writemode 应报错")
	}
}

func TestMissingFileIsOK(t *testing.T) {
	// 显式传入不存在的配置文件应报文件错误（--config 传了路径就必须能读）
	t.Setenv("NEWAPI_BASE_URL", "https://env-only.example")
	if _, err := Load("/nonexistent/path/should-not-be-read.toml"); err == nil {
		t.Fatal("不存在的配置文件应报错")
	}
}
