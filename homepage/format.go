package homepage

import (
	"strconv"
	"strings"
)

// roundToAtMost3Decimal 数字字符串保留至多三位小数（去尾零），非数字原样返回
func roundToAtMost3Decimal(s string) string {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	formatted := strconv.FormatFloat(f, 'f', 3, 64)
	if dot := strings.Index(formatted, "."); dot != -1 {
		formatted = strings.TrimRight(formatted, "0")
		formatted = strings.TrimRight(formatted, ".")
	}
	return formatted
}
