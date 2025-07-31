package buff

import "bytes"

// Replace 跳过 offset 在 packet 中查找 old 字节切片，并替换为 new 字节切片，只替换第一个找到的关键字。
// 不会修改 packet 原始切片内容，即便未替换也返回 packet 的副本。
func Replace(packet, old, new []byte, offset int) []byte {
	if offset >= len(packet) {
		// 无替换区域，直接返回副本
		return append([]byte(nil), packet...)
	}

	// 在 packet[offset:] 中查找 old
	idx := bytes.Index(packet[offset:], old)
	if idx < 0 {
		// 没找到，直接返回副本
		return append([]byte(nil), packet...)
	}
	start := offset + idx

	// 分配新切片
	newLen := len(packet) + len(new) - len(old)
	buf := make([]byte, newLen)
	// 往新切片填充数据
	w := 0
	// 拷贝前段 [0:start]
	w += copy(buf[w:], packet[:start])
	// 拷贝 new
	w += copy(buf[w:], new)
	// 拷贝后段 [start+len(old):]
	w += copy(buf[w:], packet[start+len(old):])
	return buf[:w]
}
