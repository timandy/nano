package scheduler

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTimerManager 测试定时器管理器
func TestTimerManager(t *testing.T) {
	t.Run("Add Timer", func(t *testing.T) {
		tm := &timerManager{
			timers: make(map[int64]*Timer),
		}

		timer := newTimer(1, time.Second, func() {})
		tm.addTimer(timer)

		assert.Len(t, tm.pendingTimers, 1)
		assert.Equal(t, timer, tm.pendingTimers[0])
	})

	t.Run("Steal Timers", func(t *testing.T) {
		tm := &timerManager{
			timers: make(map[int64]*Timer),
		}

		timer1 := newTimer(1, time.Second, func() {})
		timer2 := newTimer(2, time.Second, func() {})

		tm.addTimer(timer1)
		tm.addTimer(timer2)

		assert.Len(t, tm.pendingTimers, 2)
		assert.Len(t, tm.timers, 0)

		tm.stealTimers()

		assert.Len(t, tm.pendingTimers, 0)
		assert.Len(t, tm.timers, 2)
		assert.Equal(t, timer1, tm.timers[1])
		assert.Equal(t, timer2, tm.timers[2])
	})

	t.Run("Cron Execution", func(t *testing.T) {
		s := NewScheduler("test").(*scheduler)
		defer s.Close()

		var execCount atomic.Int32
		tm := &timerManager{
			timers: make(map[int64]*Timer),
		}

		// 创建一个会自动停止的定时器
		timer := newCountTimer(1, time.Nanosecond, 1, func() {
			execCount.Add(1)
		})
		tm.addTimer(timer)

		// 执行cron
		time.Sleep(20 * time.Millisecond)
		tm.cron(s)

		// 验证定时器被执行并从管理器中移除
		assert.Equal(t, int32(1), execCount.Load())
		assert.Len(t, tm.timers, 0)
	})

	t.Run("Concurrent Add Timer", func(t *testing.T) {
		tm := &timerManager{
			timers: make(map[int64]*Timer),
		}

		const numGoroutines = 100
		const numTimersPerGoroutine = 10

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(base int) {
				defer wg.Done()
				for j := 0; j < numTimersPerGoroutine; j++ {
					id := int64(base*numTimersPerGoroutine + j)
					timer := newTimer(id, time.Second, func() {})
					tm.addTimer(timer)
				}
			}(i)
		}

		wg.Wait()

		assert.Len(t, tm.pendingTimers, numGoroutines*numTimersPerGoroutine)

		tm.stealTimers()
		assert.Len(t, tm.timers, numGoroutines*numTimersPerGoroutine)
	})
}

// TestTimerManager_NewTimerMethods 测试定时器管理器的创建方法
func TestTimerManager_NewTimerMethods(t *testing.T) {
	tm := &timerManager{
		timers: make(map[int64]*Timer),
	}

	t.Run("NewTimer", func(t *testing.T) {
		timer := tm.newTimer(time.Second, func() {})
		assert.NotNil(t, timer)
		assert.Equal(t, int64(1), timer.ID())
		assert.Equal(t, infinite, timer.counter.Load())
	})

	t.Run("NewCountTimer", func(t *testing.T) {
		timer := tm.newCountTimer(time.Second, 5, func() {})
		assert.NotNil(t, timer)
		assert.Equal(t, int64(2), timer.ID())
		assert.Equal(t, int64(5), timer.counter.Load())
	})

	t.Run("NewAfterTimer", func(t *testing.T) {
		timer := tm.newAfterTimer(time.Second, func() {})
		assert.NotNil(t, timer)
		assert.Equal(t, int64(3), timer.ID())
		assert.Equal(t, int64(1), timer.counter.Load())
	})

	t.Run("NewCondTimer", func(t *testing.T) {
		cond := &testCondition{}
		timer := tm.newCondTimer(cond, func() {})
		assert.NotNil(t, timer)
		assert.Equal(t, int64(4), timer.ID())
		assert.Equal(t, infinite, timer.counter.Load())
		assert.NotNil(t, timer.condition)
	})

	t.Run("NewCondCountTimer", func(t *testing.T) {
		cond := &testCondition{}
		timer := tm.newCondCountTimer(cond, 3, func() {})
		assert.NotNil(t, timer)
		assert.Equal(t, int64(5), timer.ID())
		assert.Equal(t, int64(3), timer.counter.Load())
		assert.NotNil(t, timer.condition)
	})
}

// TestTimerManager_StealTimersRaceCondition 测试stealTimers的数据竞争
func TestTimerManager_StealTimersRaceCondition(t *testing.T) {
	tm := &timerManager{
		timers: make(map[int64]*Timer),
	}

	const numGoroutines = 50
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // addTimer goroutines + stealTimers goroutines

	// 并发添加定时器
	for i := 0; i < numGoroutines; i++ {
		go func(base int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				id := int64(base*numOperations + j)
				timer := newTimer(id, time.Second, func() {})
				tm.addTimer(timer)
			}
		}(i)
	}

	// 并发执行stealTimers
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				tm.stealTimers()
				time.Sleep(time.Microsecond) // 让出CPU时间
			}
		}()
	}

	wg.Wait()

	// 最后执行一次确保所有定时器都被转移
	tm.stealTimers()

	// 验证没有遗漏的定时器
	assert.Empty(t, tm.pendingTimers)
	assert.Len(t, tm.timers, numGoroutines*numOperations)
}

// TestTimerManager_ErrorCases 测试定时器管理器的错误情况
func TestTimerManager_ErrorCases(t *testing.T) {
	tm := &timerManager{
		timers: make(map[int64]*Timer),
	}

	t.Run("Nil Function Panics", func(t *testing.T) {
		assert.Panics(t, func() {
			tm.newTimer(time.Second, nil)
		})

		assert.Panics(t, func() {
			tm.newCountTimer(time.Second, 1, nil)
		})

		assert.Panics(t, func() {
			tm.newAfterTimer(time.Second, nil)
		})

		assert.Panics(t, func() {
			tm.newCondTimer(&testCondition{}, nil)
		})

		assert.Panics(t, func() {
			tm.newCondCountTimer(&testCondition{}, 1, nil)
		})
	})

	t.Run("Invalid Interval Panics", func(t *testing.T) {
		assert.Panics(t, func() {
			tm.newTimer(0, func() {})
		})

		assert.Panics(t, func() {
			tm.newTimer(-time.Second, func() {})
		})

		assert.Panics(t, func() {
			tm.newCountTimer(0, 1, func() {})
		})

		assert.Panics(t, func() {
			tm.newAfterTimer(-time.Second, func() {})
		})
	})

	t.Run("Nil Condition Panics", func(t *testing.T) {
		assert.Panics(t, func() {
			tm.newCondTimer(nil, func() {})
		})

		assert.Panics(t, func() {
			tm.newCondCountTimer(nil, 1, func() {})
		})
	})
}
