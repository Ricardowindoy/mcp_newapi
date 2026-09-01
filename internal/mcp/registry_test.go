package mcp

import (
	"testing"

	"mcp_newapi/internal/newapi"
)

// TestRegistryIntegrity 表本身的完整性：名字唯一、handler 非空、有声明。
func TestRegistryIntegrity(t *testing.T) {
	seen := map[string]bool{}
	for _, def := range toolRegistry {
		if def.Name == "" {
			t.Fatal("存在无名表项")
		}
		if seen[def.Name] {
			t.Errorf("工具重复注册: %s", def.Name)
		}
		seen[def.Name] = true
		if def.Handler == nil {
			t.Errorf("%s: handler 为空", def.Name)
		}
		if len(def.Options) == 0 {
			t.Errorf("%s: 缺少声明（描述/参数）", def.Name)
		}
	}
	if len(toolRegistry) != 25 {
		t.Errorf("工具总数 = %d, want 25", len(toolRegistry))
	}
}

// TestTierFiltering 分档过滤：read 只注册 read 档；admin 全量。
func TestTierFiltering(t *testing.T) {
	client := newapi.NewClient("https://unused.example", "", 0)
	count := func(mode string) int {
		s := NewServer(client, mode, nil)
		return len(s.ListTools())
	}
	if got := count("read"); got != 13 {
		t.Errorf("read 档 = %d, want 13", got)
	}
	if got := count("ops"); got != 19 {
		t.Errorf("ops 档 = %d, want 19", got)
	}
	if got := count("admin"); got != 25 {
		t.Errorf("admin 档 = %d, want 25", got)
	}
}
