package newapi

import (
	"strings"
)

// MaskKey 掩码密钥：保留头尾各 4 位，中间用 *** 代替。
// 已掩码（含 ***）或过短（≤10）的值原样返回；空值返回空。
func MaskKey(k string) string {
	k = strings.TrimSpace(k)
	if k == "" || strings.Contains(k, "***") {
		return k
	}
	r := []rune(k)
	if len(r) <= 10 {
		return string(r[:2]) + "***" + string(r[len(r)-2:])
	}
	return string(r[:4]) + "***" + string(r[len(r)-4:])
}
