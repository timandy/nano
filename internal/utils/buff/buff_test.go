package buff_test

import (
	"bytes"
	"reflect"
	"testing"
	"unsafe"

	"github.com/lonng/nano/internal/utils/buff"
	"github.com/stretchr/testify/assert"
)

var (
	data       = []byte("hello world, hello world, hello world, hello world")
	old        = []byte("world")
	newVal     = []byte("gopher")
	replaceOff = 13 // 从第二个 "world" 开始替换
)

func TestReplace_BasicReplace(t *testing.T) {
	packet := []byte("abc123xyz")
	old := []byte("123")
	newVal := []byte("456")
	result := buff.Replace(packet, old, newVal, 0)
	//
	assert.Equal(t, []byte("abc456xyz"), result)
	assertNotSameSlice(t, result, packet)        // 替换发生，必须不同底层
	assert.Equal(t, []byte("abc123xyz"), packet) // 原始数据不能变
}

func TestReplace_OffsetSkip(t *testing.T) {
	packet := []byte("123abc123")
	old := []byte("123")
	newVal := []byte("456")
	result := buff.Replace(packet, old, newVal, 4)
	//
	assert.Equal(t, []byte("123abc456"), result)
	assertNotSameSlice(t, result, packet)        // 替换发生，必须不同底层
	assert.Equal(t, []byte("123abc123"), packet) // 原始数据不能变
}

func TestReplace_NoMatch(t *testing.T) {
	packet := []byte("abcxyz")
	old := []byte("123")
	newVal := []byte("456")
	result := buff.Replace(packet, old, newVal, 0)
	//
	assert.Equal(t, []byte("abcxyz"), result)
	assertNotSameSlice(t, result, packet)     // 无替换，返回原切片副本
	assert.Equal(t, []byte("abcxyz"), packet) // 内容无变化
}

func TestReplace_OffsetTooLarge(t *testing.T) {
	packet := []byte("abc123")
	old := []byte("123")
	newVal := []byte("456")
	result := buff.Replace(packet, old, newVal, 100)
	///
	assert.Equal(t, []byte("abc123"), result)
	assertNotSameSlice(t, result, packet)     // 无替换，返回原切片副本
	assert.Equal(t, []byte("abc123"), packet) // 内容无变化
}

func TestReplace_EmptyOld(t *testing.T) {
	packet := []byte("abc123")
	old := []byte("")
	newVal := []byte("456")

	result := buff.Replace(packet, old, newVal, 0)
	assert.Equal(t, []byte("456abc123"), result)
	assertNotSameSlice(t, result, packet)     // 替换发生，必须不同底层
	assert.Equal(t, []byte("abc123"), packet) // 内容无变化
}

func BenchmarkBuffReplace(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = buff.Replace(data, old, newVal, replaceOff)
	}
}

func BenchmarkBytesReplace(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// 注意这里无法传 offset，只能从头替换一次
		_ = bytes.Replace(data, old, newVal, 1)
	}
}

// assertNotSameSlice 断言两个 slice 底层内存不同
func assertNotSameSlice(t *testing.T, a, b []byte) {
	if sameSlice(a, b) {
		t.Fatalf("slices share the same underlying array but expected different")
	}
}

// sameSlice 判断两个 byte slice 是否底层内存相同
func sameSlice(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	ah := (*reflect.SliceHeader)(unsafe.Pointer(&a))
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	return ah.Data == bh.Data
}
