// Package csvutil 提供 CSV 格式构建工具，支持表头、字段转义（逗号/引号/换行）和常见类型转换。
package csvutil

import (
	"fmt"
	"strings"
	"time"
)

// toStr 将任意值转换为 CSV 字段字符串。
//   - string: 原样返回
//   - time.Time: 格式化为 ISO8601 (time.RFC3339)
//   - nil: 返回空字符串
//   - 其他类型: 使用 fmt.Sprintf("%v") 兜底，支持数字等
func toStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case time.Time:
		return x.Format(time.RFC3339)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", x)
	}
}

// needQuote 判断字段值是否需要双引号包裹（含逗号、双引号或换行符时）。
func needQuote(s string) bool {
	return strings.ContainsAny(s, ",\"\n")
}

// quoteField 对字段值进行双引号包裹和转义：内部的 " 双写为 ""。
func quoteField(s string) string {
	s = strings.ReplaceAll(s, `"`, `""`)
	return `"` + s + `"`
}

// Build 根据 rows 和 headers 构建 CSV 格式字符串。
//   - 首行为表头，各列以逗号分隔。
//   - 后续每行对应一条记录，按 headers 顺序取值。
//   - 字段值若包含逗号、双引号或换行符，则用双引号包裹并将内部双引号双写转义。
//   - 若 rows 为空，仅输出表头行。
func Build(rows []map[string]any, headers []string) string {
	var b strings.Builder

	// 写入表头
	for i, h := range headers {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(h)
	}
	b.WriteByte('\n')

	// 写入数据行
	for _, row := range rows {
		for i, h := range headers {
			if i > 0 {
				b.WriteByte(',')
			}
			val := toStr(row[h])
			if needQuote(val) {
				val = quoteField(val)
			}
			b.WriteString(val)
		}
		b.WriteByte('\n')
	}

	return b.String()
}
