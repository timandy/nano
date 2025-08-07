package twscheduler

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSlot 测试槽位功能
func TestSlot(t *testing.T) {
	t.Run("New slot", func(t *testing.T) {
		slt := &slot{}
		assert.NotNil(t, slt)
		assert.Nil(t, slt.head)
	})

	t.Run("Link single timer", func(t *testing.T) {
		slt := &slot{}
		tmr := newTimer(1, 0, func() {})

		slt.link(tmr)

		assert.Equal(t, tmr, slt.head)
		assert.Equal(t, slt, tmr.slot)
		assert.Nil(t, tmr.prev)
		assert.Nil(t, tmr.next)
	})

	t.Run("Link multiple timers", func(t *testing.T) {
		slt := &slot{}
		timer1 := newTimer(1, 0, func() {})
		timer2 := newTimer(2, 0, func() {})
		timer3 := newTimer(3, 0, func() {})

		// 依次链接多个定时器
		slt.link(timer1)
		slt.link(timer2)
		slt.link(timer3)

		// 验证链表结构: timer3 -> timer2 -> timer1
		assert.Equal(t, timer3, slt.head)
		assert.Equal(t, slt, timer3.slot)
		assert.Equal(t, timer2, timer3.next)
		assert.Nil(t, timer3.prev)

		assert.Equal(t, slt, timer2.slot)
		assert.Equal(t, timer1, timer2.next)
		assert.Equal(t, timer3, timer2.prev)

		assert.Equal(t, slt, timer1.slot)
		assert.Nil(t, timer1.next)
		assert.Equal(t, timer2, timer1.prev)
	})

	t.Run("Link timer with existing slot", func(t *testing.T) {
		slot1 := &slot{}
		slot2 := &slot{}
		tmr := newTimer(1, 0, func() {})

		// 先链接到slot1
		slot1.link(tmr)
		assert.Equal(t, slot1, tmr.slot)

		// 再链接到slot2（应该覆盖之前的链接）
		tmr.unlink()
		slot2.link(tmr)
		assert.Equal(t, slot2, tmr.slot)
		assert.Equal(t, tmr, slot2.head)
		assert.Nil(t, slot1.head)
	})

	t.Run("Link nil timer", func(t *testing.T) {
		slt := &slot{}
		// 应该panic
		assert.Panics(t, func() {
			slt.link(nil)
		})
		assert.Nil(t, slt.head)
	})
}

// TestSlot_ConcurrentAccess 测试槽位的并发访问
func TestSlot_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent link operations", func(t *testing.T) {
		slt := &slot{}
		const numTimers = 100
		timers := make([]*timer, numTimers)

		for i := 0; i < numTimers; i++ {
			timers[i] = newTimer(int64(i+1), 0, func() {})
		}

		// 并发链接定时器
		var wg sync.WaitGroup
		wg.Add(numTimers)
		for i := 0; i < numTimers; i++ {
			go func(idx int) {
				defer wg.Done()
				slt.mu.Lock()
				defer slt.mu.Unlock()
				slt.link(timers[idx])
			}(i)
		}
		wg.Wait()

		// 验证所有定时器都被链接到槽中
		count := 0
		current := slt.head
		for current != nil {
			count++
			current = current.next
		}
		assert.Equal(t, numTimers, count)
	})

	t.Run("Concurrent link and unlink operations", func(t *testing.T) {
		slt := &slot{}
		const numTimers = 50
		timers := make([]*timer, numTimers)

		for i := 0; i < numTimers; i++ {
			timers[i] = newTimer(int64(i+1), 0, func() {})
		}

		// 并发链接定时器
		var wg sync.WaitGroup
		wg.Add(numTimers)
		for i := 0; i < numTimers; i++ {
			go func(idx int) {
				defer wg.Done()
				slt.mu.Lock()
				defer slt.mu.Unlock()
				slt.link(timers[idx])
			}(i)
		}
		wg.Wait()

		// 并发解除链接
		wg.Add(numTimers)
		for i := 0; i < numTimers; i++ {
			go func(idx int) {
				defer wg.Done()
				slt.mu.Lock()
				defer slt.mu.Unlock()
				timers[idx].unlink()
			}(i)
		}
		wg.Wait()

		// 验证所有定时器都被解除链接
		assert.Nil(t, slt.head)
		for _, tmr := range timers {
			assert.Nil(t, tmr.slot)
			assert.Nil(t, tmr.prev)
			assert.Nil(t, tmr.next)
		}
	})
}

// TestSlot_EdgeCases 测试边界情况
func TestSlot_EdgeCases(t *testing.T) {
	t.Run("Link same timer multiple times", func(t *testing.T) {
		slt := &slot{}
		tmr := newTimer(1, 0, func() {})

		// 多次链接同一个定时器
		for i := 0; i < 5; i++ {
			tmr.unlink()
			slt.link(tmr)
		}

		// 应该只有一个实例在链表中
		count := 0
		current := slt.head
		for current != nil {
			if current == tmr {
				count++
			}
			current = current.next
		}
		assert.Equal(t, 1, count)
	})

	t.Run("Empty slot operations", func(t *testing.T) {
		slt := &slot{}

		// 空槽的各种操作不应该panic
		assert.Nil(t, slt.head)

		// 尝试链接nil应该panic
		assert.Panics(t, func() {
			slt.link(nil)
		})
		assert.Nil(t, slt.head)
	})
}
