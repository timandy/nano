package twscheduler

import "sync"

// 槽位
type slot struct {
	mu   sync.Mutex
	head *timer // 槽位头部的定时器链表
}

// link 将定时器链接到槽位的头部
func (s *slot) link(t *timer) {
	t.slot = s
	t.next = s.head
	if s.head != nil {
		s.head.prev = t
	}
	t.prev = nil
	s.head = t
}

// clear 清空槽位, 执行定时器的清理工作
func (s *slot) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var next *timer
	for t := s.head; t != nil; t = next {
		// 提前保存 next，防止 t 被 unlink 后 next 丢失
		next = t.next
		t.unlink()
		t.clean()
	}
}
