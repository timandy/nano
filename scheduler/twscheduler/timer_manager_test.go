package twscheduler

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lonng/nano/scheduler/schedulerapi"
	"github.com/stretchr/testify/assert"
)

// TestTimerManager 测试时间轮定时器管理器
func TestTimerManager(t *testing.T) {
	t.Run("New TimerManager", func(t *testing.T) {
		tm := &timerManager{
			tick:     int64(time.Millisecond),
			slotMask: 15,
			slots:    make([]*slot, 16),
		}
		for i := 0; i < 16; i++ {
			tm.slots[i] = &slot{}
		}
		assert.Equal(t, int64(time.Millisecond), tm.tick)
		assert.Equal(t, int64(15), tm.slotMask)
		assert.Len(t, tm.slots, 16)
	})

	t.Run("Add Timer to slot", func(t *testing.T) {
		tm := &timerManager{
			tick:     int64(time.Millisecond),
			slotMask: 15,
			slots:    make([]*slot, 16),
		}
		for i := 0; i < 16; i++ {
			tm.slots[i] = &slot{}
		}

		tmr := newTimer(1, 50*time.Millisecond, func() {})
		tm.addTimer(tmr, int64(50*time.Millisecond))

		// 验证定时器被添加到正确的槽位
		expectedSlot := (0 + 50) & 15 // 50/1 = 50, slot = 50 & 15 = 2
		assert.Equal(t, tm.slots[expectedSlot], tmr.slot)
		assert.Equal(t, tmr, tm.slots[expectedSlot].head)
	})

	t.Run("Add Timer with zero delay", func(t *testing.T) {
		tm := &timerManager{
			tick:     int64(time.Millisecond),
			slotMask: 15,
			slots:    make([]*slot, 16),
		}
		for i := 0; i < 16; i++ {
			tm.slots[i] = &slot{}
		}

		tmr := newTimer(1, 50*time.Millisecond, func() {})
		tm.addTimer(tmr, 0) // 零延迟应该被设置为1

		expectedSlot := (0 + 1) & 15
		assert.Equal(t, tm.slots[expectedSlot], tmr.slot)
	})

	t.Run("Next slot advancement", func(t *testing.T) {
		tm := &timerManager{
			tick:     int64(time.Millisecond),
			slotMask: 15,
			slots:    make([]*slot, 16),
		}
		tm.current.Store(5)
		for i := 0; i < 16; i++ {
			tm.slots[i] = &slot{}
		}

		slt := tm.next()
		assert.Equal(t, tm.slots[5], slt)
		assert.Equal(t, int64(6), tm.current.Load())

		// 测试环绕
		tm.current.Store(15)
		slt = tm.next()
		assert.Equal(t, tm.slots[15], slt)
		assert.Equal(t, int64(0), tm.current.Load())
	})

	t.Run("Advance with empty slot", func(t *testing.T) {
		tm := &timerManager{
			tick:     int64(time.Millisecond),
			slotMask: 15,
			slots:    make([]*slot, 16),
		}
		for i := 0; i < 16; i++ {
			tm.slots[i] = &slot{}
		}

		s := NewScheduler("test", time.Millisecond, 16).(*scheduler)
		defer s.Close()

		// 空槽位不应该panic
		tm.advance(s)
		assert.Equal(t, int64(1), tm.current.Load())
	})

	t.Run("Advance with stopped timer", func(t *testing.T) {
		tm := &timerManager{
			tick:     int64(time.Millisecond),
			slotMask: 15,
			slots:    make([]*slot, 16),
		}
		for i := 0; i < 16; i++ {
			tm.slots[i] = &slot{}
		}

		s := NewScheduler("test", time.Millisecond, 16).(*scheduler)
		defer s.Close()

		tmr := newTimer(1, 50*time.Millisecond, func() {})
		tmr.Stop()
		tm.addTimer(tmr, int64(50*time.Millisecond))

		// 执行一轮后, 停止的定时器应该被清理
		for i := 0; i < 16; i++ {
			tm.advance(s)
		}
		assert.Nil(t, tmr.slot)
	})

	t.Run("Advance with condition timer", func(t *testing.T) {
		tm := &timerManager{
			tick:     int64(time.Millisecond),
			slotMask: 15,
			slots:    make([]*slot, 16),
		}
		for i := 0; i < 16; i++ {
			tm.slots[i] = &slot{}
		}

		s := NewScheduler("test", time.Millisecond, 16).(*scheduler)
		defer s.Close()

		cond := &testCondition{shouldTrigger: atomic.Bool{}}
		cond.shouldTrigger.Store(true)

		tmr := newCondTimer(1, cond, func() {})
		tm.addTimer(tmr, 0)

		// 条件定时器应该执行并重新调度
		tm.advance(s)
		assert.NotNil(t, tmr.slot) // 应该被重新调度
	})

	t.Run("Close timer manager", func(t *testing.T) {
		tm := &timerManager{
			tick:     int64(time.Millisecond),
			slotMask: 15,
			slots:    make([]*slot, 16),
		}
		for i := 0; i < 16; i++ {
			tm.slots[i] = &slot{}
		}

		// 添加一些定时器
		tmr := newTimer(1, 50*time.Millisecond, func() {})
		tm.addTimer(tmr, int64(50*time.Millisecond))

		tm.close()

		// 验证所有槽位被清空
		for i := 0; i < 16; i++ {
			assert.Nil(t, tm.slots[i].head)
		}
	})
}

// TestTimerManager_NewTimerMethods 测试定时器管理器的创建方法
func TestTimerManager_NewTimerMethods(t *testing.T) {
	tm := &timerManager{
		tick:     int64(time.Millisecond),
		slotMask: 15,
		slots:    make([]*slot, 16),
	}
	for i := 0; i < 16; i++ {
		tm.slots[i] = &slot{}
	}

	t.Run("newTimer", func(t *testing.T) {
		tmr := tm.newTimer(time.Second, func() {})
		assert.NotNil(t, tmr)
		assert.Equal(t, int64(1), tmr.ID())
		assert.Equal(t, schedulerapi.Infinite, tmr.counter.Load())
	})

	t.Run("newCountTimer", func(t *testing.T) {
		tmr := tm.newCountTimer(time.Second, 5, func() {})
		assert.NotNil(t, tmr)
		assert.Equal(t, int64(2), tmr.ID())
		assert.Equal(t, int64(5), tmr.counter.Load())
	})

	t.Run("newAfterTimer", func(t *testing.T) {
		tmr := tm.newAfterTimer(time.Second, func() {})
		assert.NotNil(t, tmr)
		assert.Equal(t, int64(3), tmr.ID())
		assert.Equal(t, int64(1), tmr.counter.Load())
	})

	t.Run("newCondTimer", func(t *testing.T) {
		cond := &testCondition{shouldTrigger: atomic.Bool{}}
		tmr := tm.newCondTimer(cond, func() {})
		assert.NotNil(t, tmr)
		assert.Equal(t, int64(4), tmr.ID())
		assert.Equal(t, schedulerapi.Infinite, tmr.counter.Load())
		assert.NotNil(t, tmr.condition)
	})

	t.Run("newCondCountTimer", func(t *testing.T) {
		cond := &testCondition{shouldTrigger: atomic.Bool{}}
		tmr := tm.newCondCountTimer(cond, 3, func() {})
		assert.NotNil(t, tmr)
		assert.Equal(t, int64(5), tmr.ID())
		assert.Equal(t, int64(3), tmr.counter.Load())
		assert.NotNil(t, tmr.condition)
	})
}

// TestTimerManager_TickerMethods 测试Ticker创建方法
func TestTimerManager_TickerMethods(t *testing.T) {
	tm := &timerManager{
		tick:     int64(time.Millisecond),
		slotMask: 15,
		slots:    make([]*slot, 16),
	}
	for i := 0; i < 16; i++ {
		tm.slots[i] = &slot{}
	}

	t.Run("newTicker", func(t *testing.T) {
		ticker := tm.newTicker(time.Second)
		assert.NotNil(t, ticker)
		assert.Equal(t, int64(1), ticker.ID())
		assert.False(t, ticker.Stopped())
		ticker.Stop()
		assert.True(t, ticker.Stopped())
	})

	t.Run("newCountTicker", func(t *testing.T) {
		ticker := tm.newCountTicker(time.Second, 5)
		assert.NotNil(t, ticker)
		assert.Equal(t, int64(2), ticker.ID())
		assert.False(t, ticker.Stopped())
		ticker.Stop()
		assert.True(t, ticker.Stopped())
	})

	t.Run("newAfterTicker", func(t *testing.T) {
		ticker := tm.newAfterTicker(time.Second)
		assert.NotNil(t, ticker)
		assert.Equal(t, int64(3), ticker.ID())
		assert.False(t, ticker.Stopped())
		ticker.Stop()
		assert.True(t, ticker.Stopped())
	})

	t.Run("newCondTicker", func(t *testing.T) {
		cond := &testCondition{shouldTrigger: atomic.Bool{}}
		ticker := tm.newCondTicker(cond)
		assert.NotNil(t, ticker)
		assert.Equal(t, int64(4), ticker.ID())
		assert.False(t, ticker.Stopped())
		ticker.Stop()
		assert.True(t, ticker.Stopped())
	})

	t.Run("newCondCountTicker", func(t *testing.T) {
		cond := &testCondition{shouldTrigger: atomic.Bool{}}
		ticker := tm.newCondCountTicker(cond, 3)
		assert.NotNil(t, ticker)
		assert.Equal(t, int64(5), ticker.ID())
		assert.False(t, ticker.Stopped())
		ticker.Stop()
		assert.True(t, ticker.Stopped())
	})
}

// TestTimerManager_ConcurrentAddTimer 测试并发添加定时器
func TestTimerManager_ConcurrentAddTimer(t *testing.T) {
	tm := &timerManager{
		tick:     int64(time.Millisecond),
		slotMask: 15,
		slots:    make([]*slot, 16),
	}
	for i := 0; i < 16; i++ {
		tm.slots[i] = &slot{}
	}

	const numGoroutines = 50
	const numTimersPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(base int) {
			defer wg.Done()
			for j := 0; j < numTimersPerGoroutine; j++ {
				tmr := newTimer(int64(base*numTimersPerGoroutine+j), time.Second, func() {})
				tm.addTimer(tmr, int64(time.Second))
			}
		}(i)
	}

	wg.Wait()

	// 验证所有定时器都被正确添加
	totalTimers := 0
	for i := 0; i < 16; i++ {
		slt := tm.slots[i]
		slt.mu.Lock()
		count := 0
		for tmr := slt.head; tmr != nil; tmr = tmr.next {
			count++
		}
		totalTimers += count
		slt.mu.Unlock()
	}
	assert.Equal(t, numGoroutines*numTimersPerGoroutine, totalTimers)
}

// TestTimerManager_ErrorCases 测试定时器管理器的错误情况
func TestTimerManager_ErrorCases(t *testing.T) {
	tm := &timerManager{
		tick:     int64(time.Millisecond),
		slotMask: 15,
		slots:    make([]*slot, 16),
	}
	for i := 0; i < 16; i++ {
		tm.slots[i] = &slot{}
	}

	t.Run("Nil function panic", func(t *testing.T) {
		assert.Panics(t, func() {
			tm.newTimer(time.Second, nil)
		})
		assert.Panics(t, func() {
			tm.newCountTimer(time.Second, 5, nil)
		})
		assert.Panics(t, func() {
			tm.newAfterTimer(time.Second, nil)
		})
		assert.Panics(t, func() {
			tm.newCondTimer(&testCondition{}, nil)
		})
		assert.Panics(t, func() {
			tm.newCondCountTimer(&testCondition{}, 5, nil)
		})
	})

	t.Run("Non-positive interval panic", func(t *testing.T) {
		assert.Panics(t, func() {
			tm.newTimer(0, func() {})
		})
		assert.Panics(t, func() {
			tm.newTimer(-time.Second, func() {})
		})
		assert.Panics(t, func() {
			tm.newCountTimer(0, 5, func() {})
		})
		assert.Panics(t, func() {
			tm.newAfterTimer(0, func() {})
		})
		assert.Panics(t, func() {
			tm.newAfterTicker(0)
		})
		assert.Panics(t, func() {
			tm.newTicker(0)
		})
		assert.Panics(t, func() {
			tm.newCountTicker(0, 5)
		})
	})

	t.Run("Nil condition panic", func(t *testing.T) {
		assert.Panics(t, func() {
			tm.newCondTimer(nil, func() {})
		})
		assert.Panics(t, func() {
			tm.newCondCountTimer(nil, 5, func() {})
		})
		assert.Panics(t, func() {
			tm.newCondTicker(nil)
		})
		assert.Panics(t, func() {
			tm.newCondCountTicker(nil, 5)
		})
	})
}
