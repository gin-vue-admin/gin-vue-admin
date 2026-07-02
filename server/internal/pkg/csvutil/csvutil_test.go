package csvutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuild(t *testing.T) {
	rows := []map[string]any{
		{"name": "张三", "age": 30, "city": "北京"},
		{"name": "李四", "age": 25, "city": "上海"},
	}
	headers := []string{"name", "age", "city"}
	out := Build(rows, headers)
	expected := "name,age,city\n张三,30,北京\n李四,25,上海\n"
	assert.Equal(t, expected, out)
}

func TestBuild_Empty(t *testing.T) {
	rows := []map[string]any{}
	headers := []string{"name", "age"}
	out := Build(rows, headers)
	expected := "name,age\n"
	assert.Equal(t, expected, out)
}

func TestBuild_Time(t *testing.T) {
	rows := []map[string]any{
		{"name": "x", "time": time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)},
	}
	out := Build(rows, []string{"name", "time"})
	assert.Contains(t, out, "2026-07-02T12:00:00Z")
}

func TestBuild_SpecialChars(t *testing.T) {
	rows := []map[string]any{
		{
			"desc":  "包含,逗号",
			"quote": "他说\"你好\"",
			"newline": "第一行\n第二行",
			"normal": "普通文本",
		},
	}
	headers := []string{"desc", "quote", "newline", "normal"}
	out := Build(rows, headers)
	// 逗号 -> 双引号包裹
	assert.Contains(t, out, `"包含,逗号"`)
	// 引号 -> 双写
	assert.Contains(t, out, `"他说""你好"""`)
	// 换行 -> 双引号包裹
	assert.Contains(t, out, `"第一行`)
	assert.Contains(t, out, `第二行"`)
	// 普通文本无变化
	assert.Contains(t, out, "普通文本")
}

func TestBuild_Nil(t *testing.T) {
	rows := []map[string]any{
		{"name": "张三", "age": nil},
	}
	headers := []string{"name", "age"}
	out := Build(rows, headers)
	expected := "name,age\n张三,\n"
	assert.Equal(t, expected, out)
}

func TestBuild_Number(t *testing.T) {
	rows := []map[string]any{
		{"name": "张三", "score": 98.5, "count": 42},
	}
	headers := []string{"name", "score", "count"}
	out := Build(rows, headers)
	expected := "name,score,count\n张三,98.5,42\n"
	assert.Equal(t, expected, out)
}
