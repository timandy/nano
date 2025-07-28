package log

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Person struct {
	Name string
	Age  int
}

func TestFormat(t *testing.T) {
	errExample := errors.New("example error")

	tests := []struct {
		name     string
		format   string
		args     []any
		expected string
	}{
		{"单个占位符替换", "hello %v", []any{"world"}, "hello world"},
		{"数字占位符替换", "value: %v", []any{123}, "value: 123"},
		{"多个占位符混合", "%v scored %v points", []any{"Alice", 95}, "Alice scored 95 points"},
		{"多余参数拼接", "%v and %v", []any{"one", "two", "three", 4}, "one and two three 4"},
		{"占位符多于参数", "%v %v %v", []any{"only one"}, "only one %v %v"},
		{"无占位符参数拼接", "static string", []any{"ignored"}, "static string ignored"},
		{"空格式无参数", "", []any{}, ""},
		{"空格式有参数", "", []any{"a", 1}, "a 1"},
		{"末尾 error 参数追加", "failed %v", []any{"operation", errExample}, "failed operation - example error"},
		{"仅 error 参数追加", "error occurred", []any{errExample}, "error occurred - example error"},
		{"多个占位符+末尾 error", "%v %v %v", []any{"a", 2, "c", errExample}, "a 2 c - example error"},
		{"多余参数+末尾 error", "%v", []any{"only", "extra", errExample}, "only extra - example error"},
		{"无参数占位符未替换", "%v %v", []any{}, "%v %v"},
		{"部分占位符被替换", "%v %v %v", []any{"x", "y"}, "x y %v"},
		{"占位符相邻", "%v%v%v", []any{"a", "b", "c"}, "abc"},
		{"数值指针参数", "pointer value: %v", []any{new(int)}, "pointer value: 0"},
		{"结构体指针参数", "person info: %v", []any{&Person{Name: "Tom", Age: 30}}, "person info: &{Tom 30}"},
		{"%d 占位符", "int: %d", []any{42}, "int: 42"},
		{"%s 占位符", "str: %s", []any{"foo"}, "str: foo"},
		{"混合 %d %s %v", "mix: %d %s %v", []any{1, "bar", true}, "mix: 1 bar true"},
		{"单独一个 %，原样保留", "%", []any{"foo"}, "% foo"},
		{"末尾单独一个 %", "%s%%%", []any{"foo"}, "foo%%"},
		{"转义 %%", "escaped %% sign", []any{}, "escaped % sign"},
		{"混合 %% 和占位符", "progress: %d%%", []any{80}, "progress: 80%"},
		{"混合多个 %% 和占位符", "progress: %d%%%%%%", []any{80}, "progress: 80%%%"},
		{"混合多个 %% 和多个占位符", "progress: %d%%%%%%%s", []any{80, "true"}, "progress: 80%%%true"},
		{"连续多个 %%", "%%%%", []any{}, "%%"},
		{"%f 浮点数", "float: %f", []any{3.14}, "float: 3.14"},
		{"%t 布尔值", "bool: %t", []any{false}, "bool: false"},
		{"nil 参数", "nil value: %v", []any{nil}, "nil value: <nil>"},
		{"nil 参数2", "nil value: %v", []any{(*int)(nil)}, "nil value: <nil>"},
		{"参数为切片", "slice: %v", []any{[]int{1, 2, 3}}, "slice: [1 2 3]"},
		{"参数为 map", "map: %v", []any{map[string]int{"a": 1}}, "map: map[a:1]"},
		{"未知占位符", "unknown: %x", []any{123}, "unknown: %x"},
		{"占位符在结尾", "end with %v", []any{"ok"}, "end with ok"},
		{"占位符在开头", "%v is first", []any{"first"}, "first is first"},
		{"连续不同占位符", "%d-%s-%v", []any{1, "a", true}, "1-a-true"},
		{"%q 占位符", "quoted: %q", []any{"text"}, "quoted: \"text\""},
		{"%T 占位符", "type: %T", []any{123}, "type: int"},
		{"%T 占位符指针", "type: %T", []any{new(int)}, "type: *int"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := Format(tt.format, tt.args...)
			assert.Equal(t, tt.expected, out)
		})
	}
}

func TestFormatEdgeCases(t *testing.T) {
	// 测试空字符串、nil参数、非常长字符串等边界情况
	longStr := strings.Repeat("a", 10000)
	longArgs := make([]any, 100)
	for i := 0; i < 100; i++ {
		longArgs[i] = i
	}

	// 空 format 和 nil args
	assert.Equal(t, "<nil>", Format("", nil))

	// 长字符串无参数
	assert.Equal(t, longStr, Format(longStr))

	// 长字符串带参数但无占位符，参数拼接
	out := Format(longStr, 1, 2, 3)
	assert.True(t, len(out) > len(longStr))

	// 长字符串多次占位符替换
	format := strings.Repeat("%v", 100)
	expected := ""
	for i := 0; i < 100; i++ {
		expected += fmt.Sprint(i)
	}
	assert.Equal(t, expected, Format(format, longArgs...))

	// 参数数量多于占位符，多余参数拼接
	expected = expected + " 100 101 102"
	assert.Equal(t, expected, Format(format, append(longArgs, 100, 101, 102)...))

	// format 仅为占位符，参数为 nil
	assert.Equal(t, "<nil>", Format("%v", nil))

	// format 仅为转义符
	assert.Equal(t, "%", Format("%%"))
	assert.Equal(t, "%%", Format("%%%%"))

	// format 仅为未知占位符
	assert.Equal(t, "%x", Format("%x", nil))
	assert.Equal(t, "%y", Format("%y"))

	// format 仅为一个字符
	assert.Equal(t, "a", Format("a"))
	assert.Equal(t, "a 1", Format("a", 1))

	// 参数为 error 且 format 为空
	err := errors.New("err")
	assert.Equal(t, " - err", Format("", err))

	// 参数为 error 且 format 仅为占位符
	assert.Equal(t, "%v - err", Format("%v", err))

	// format 仅为 %v，参数为多个
	assert.Equal(t, "1 2 3", Format("%v", 1, 2, 3))

	// format 仅为 %v%v，参数为 1 个
	assert.Equal(t, "1%v", Format("%v%v", 1))

	// format 仅为 %v%v，参数为 0 个
	assert.Equal(t, "%v%v", Format("%v%v"))

	// format 仅为 %d%s，参数为 0 个
	assert.Equal(t, "%d%s", Format("%d%s"))

	// format 仅为 %d%s，参数为 1 个
	assert.Equal(t, "1%s", Format("%d%s", 1))

	// format 仅为 %d%s，参数为 2 个
	assert.Equal(t, "1foo", Format("%d%s", 1, "foo"))

	// %q 占位符边界测试
	assert.Equal(t, "\"<nil>\"", Format("%q", nil)) // 参数为 nil
	assert.Equal(t, "\"\"", Format("%q", ""))       // 参数为空字符串
	assert.Equal(t, "\"\\n\"", Format("%q", "\n"))  // 参数为换行符
	assert.Equal(t, "\"123\"", Format("%q", 123))   // 参数为数字

	// %T 占位符边界测试
	assert.Equal(t, "<nil>", Format("%T", nil))                             // 参数为 nil
	assert.Equal(t, "int", Format("%T", 123))                               // 参数为基本类型
	assert.Equal(t, "[]int", Format("%T", []int{1, 2, 3}))                  // 参数为切片
	assert.Equal(t, "map[string]int", Format("%T", map[string]int{"a": 1})) // 参数为映射
	assert.Equal(t, "*int", Format("%T", new(int)))                         // 参数为指针
}

func BenchmarkFormat(b *testing.B) {
	format := "User %v has %v unread messages, status: %v"
	args := []any{"Alice", 5, "active"}

	for i := 0; i < b.N; i++ {
		_ = Format(format, args...)
	}
}

func BenchmarkFormatSprintf(b *testing.B) {
	format := "User %v has %v unread messages, status: %v" // 改为 fmt 识别的占位符
	args := []any{"Alice", 5, "active"}

	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf(format, args...)
	}
}

func BenchmarkFormatLargeInput(b *testing.B) {
	format := strings.Repeat("Hello %v, ", 100) + "status: %v"
	args := make([]any, 101)
	for i := 0; i < 100; i++ {
		args[i] = "User"
	}
	args[100] = "active"

	for i := 0; i < b.N; i++ {
		_ = Format(format, args...)
	}
}

func BenchmarkFormatLargeInputSprintf(b *testing.B) {
	format := strings.Repeat("Hello %v, ", 100) + "status: %v" // fmt 占位符
	args := make([]any, 101)
	for i := 0; i < 100; i++ {
		args[i] = "User"
	}
	args[100] = "active"

	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf(format, args...)
	}
}
