package reflection

import (
	"reflect"
	"runtime"
	"unsafe"
)

// Add 指针运算
func Add(p unsafe.Pointer, x uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

// NameOfFunction 获取函数名称
func NameOfFunction(f any) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}
