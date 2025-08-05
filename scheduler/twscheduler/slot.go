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
