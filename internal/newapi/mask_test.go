package newapi

import "testing"

func TestMaskKey(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},                                          // 空
		{"sk-abc", "sk***bc"},                             // 短值（≤10）保头尾2
		{"sk-abcdefghij1234567890", "sk-a***7890"},        // 常规：头尾4
		{"vnnF**********RcPV", "vnnF**********RcPV"},      // 已掩码原样
		{"  sk-spaces-123456  ", "sk-s***3456"},           // 去首尾空白
	}
	for _, c := range cases {
		if got := MaskKey(c.in); got != c.want {
			t.Errorf("MaskKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
