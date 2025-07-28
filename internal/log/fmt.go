package log

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Format 实现了类似 fmt.Sprintf 的格式化，但仅支持常见占位符 %%、%d、%s、%v、%q、%T 等，并具有以下特性：
// 1. 支持将参数依次替换 format 字符串中的占位符；
// 2. 如果参数多于占位符，剩余参数以空格拼接追加在末尾；
// 3. 如果 format 没有占位符但有参数，所有参数以空格拼接追加在末尾；
// 4. 如果最后一个参数为 error 类型，会以 " - 错误信息" 的形式追加在结果末尾；
// 5. 占位符多于参数时，未被替换的占位符原样保留。
// 6. 性能比 fmt.Sprintf 稍高，适合日志记录等场景。
//
// 示例：
//
//	Format("hello %v", "world")                      // "hello world"
//	Format("%v + %v = %v", 1, 2, 3)                  // "1 + 2 = 3"
//	Format("no placeholders", 1, 2)                  // "no placeholders 1 2"
//	Format("error: %v", "fail", errors.New("err"))   // "error: fail - err"
//	Format("%v %v", 1)                               // "1 %v"
//	Format("%% %d %s", 42, "foo")                    // "% 42 foo"
//	Format("%v %v %v", 1)                            // "1 %v %v"
//	Format("only error", errors.New("fail"))         // "only error - fail"
//	Format("no args")                                // "no args"
//	Format("%t", true)                               // "true"
//	Format("%q", "quoted")                           // "\"quoted\""
//	Format("%T", 123)                                // "int"
//	Format("%v", nil)                                // ""
func Format(format string, args ...any) string {
	//提取末尾的 error
	var trailingErr error
	n := len(args)
	if n > 0 {
		if e, ok := args[n-1].(error); ok {
			trailingErr = e
			args = args[:n-1]
			n--
		}
	}

	// 格式化字符串
	builder := strings.Builder{}
	builder.Grow(len(format) + n*8) // 预估平均每个 arg 约 8 个字符
	argIdx := 0
	for len(format) > 0 {
		// 未找到下一个 %
		idx := strings.IndexByte(format, '%')
		if idx < 0 {
			builder.WriteString(format)
			break
		}

		// 写入 % 前面的内容
		if idx > 0 {
			builder.WriteString(format[:idx])
			format = format[idx:]
		}

		// 单独一个 %，原样保留
		if len(format) < 2 {
			builder.WriteByte('%')
			break
		}

		// 读取下一个占位符，format 移除 '%x'
		verb := format[1]
		format = format[2:]

		// 连续两个 %%，转义为一个 %
		if verb == '%' {
			builder.WriteByte('%')
			continue
		}

		// 参数不足，保留原样
		if argIdx >= n {
			builder.WriteByte('%')
			builder.WriteByte(verb)
			continue
		}

		arg := args[argIdx]
		argIdx++
		switch verb {
		case 'd', 'f', 's', 't', 'v':
			builder.WriteString(toString(arg))
		case 'q':
			builder.WriteString(strconv.Quote(toString(arg)))
		case 'T':
			builder.WriteString(toTypeString(arg))
		default:
			builder.WriteByte('%')
			builder.WriteByte(verb)
		}
	}

	// 剩余的参数
	if argIdx < n {
		for _, arg := range args[argIdx:] {
			if builder.Len() > 0 {
				builder.WriteByte(' ')
			}
			builder.WriteString(toString(arg))
		}
	}

	// 末尾 error
	if trailingErr != nil {
		builder.WriteString(" - ")
		builder.WriteString(trailingErr.Error())
	}

	return builder.String()
}

// FormatArgs 格式化参数
func FormatArgs(args ...any) string {
	switch len(args) {
	//没有参数的时候直接返回空字符串
	case 0:
		return ""
	//只有一个参数的时候不处理转义符了, 直接原样返回
	case 1:
		return toString(args[0])
	//多个参数格式化
	default:
		return Format(toString(args[0]), args[1:]...)
	}
}

// toTypeString 获取值的类型字符串
func toTypeString(val any) string {
	if val == nil {
		return "<nil>"
	}
	if typ := reflect.TypeOf(val); typ != nil {
		return typ.String()
	}
	return "<unknown type>"
}

// toString 转换为字符串
func toString(val any) string {
	switch v := val.(type) {
	case int:
		return strconv.Itoa(v)
	case *int:
		if v == nil {
			return "<nil>"
		}
		return strconv.Itoa(*v)
	case string:
		return v
	case *string:
		if v == nil {
			return "<nil>"
		}
		return *v
	default:
		return fmt.Sprint(val)
	}
}
