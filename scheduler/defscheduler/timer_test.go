package defscheduler

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/lonng/nano/scheduler/schedulerapi"
	"github.com/stretchr/testify/assert"
)

// TestTimer_BasicFunctionality 测试Timer基本功能
func TestTimer_BasicFunctionality(t *testing.T) {
	t.Run("Timer ID", func(t *testing.T) {
		tmr := newTimer(123, time.Second, func() {})
		assert.Equal(t, int64(123), tmr.ID())
	})

	t.Run("Timer Stop", func(t *testing.T) {
		tmr := newTimer(1, time.Second, func() {})
		assert.False(t, tmr.Stopped())

		tmr.Stop()
		assert.True(t, tmr.Stopped())

		// 重复停止应该安全
		tmr.Stop()
		assert.True(t, tmr.Stopped())
	})

	t.Run("Timer Countdown", func(t *testing.T) {
		tmr := newCountTimer(1, time.Second, 3, func() {})
		assert.False(t, tmr.Stopped())

		tmr.countdown()
		assert.False(t, tmr.Stopped())

		tmr.countdown()
		assert.False(t, tmr.Stopped())

		tmr.countdown()
		assert.True(t, tmr.Stopped())
	})

	t.Run("Infinite Timer Countdown", func(t *testing.T) {
		tmr := newTimer(1, time.Second, func() {})
		for i := 0; i < 1000; i++ {
			tmr.countdown()
		}
		assert.False(t, tmr.Stopped())
	})
}

// TestTimer_Execution 测试Timer执行逻辑
func TestTimer_Execution(t *testing.T) {
	t.Run("Interval Timer Execution", func(t *testing.T) {
		var execCount atomic.Int32
		s := NewScheduler("test", time.Millisecond).(*scheduler)
		defer s.Close()

		tmr := newTimer(1, 50*time.Millisecond, func() {
			execCount.Add(1)
		})

		now := time.Now()
		ts := now.UnixNano()

		// 刚创建时不应该执行
		tmr.exec(s, now, ts)
		assert.Equal(t, int32(0), execCount.Load())

		// 间隔时间后应该执行
		futureTime := now.Add(100 * time.Millisecond)
		futureTs := futureTime.UnixNano()
		tmr.exec(s, futureTime, futureTs)
		time.Sleep(10 * time.Millisecond) // 等待任务执行
		assert.Equal(t, int32(1), execCount.Load())
	})

	t.Run("Stopped Timer Should Not Execute", func(t *testing.T) {
		var execCount atomic.Int32
		s := NewScheduler("test", time.Millisecond).(*scheduler)
		defer s.Close()

		tmr := newTimer(1, 10*time.Millisecond, func() {
			execCount.Add(1)
		})
		tmr.Stop()

		futureTime := time.Now().Add(100 * time.Millisecond)
		futureTs := futureTime.UnixNano()
		tmr.exec(s, futureTime, futureTs)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, int32(0), execCount.Load())
	})
}

// TestTimer_CreateTimer 测试createTimer函数的所有分支
func TestTimer_CreateTimer(t *testing.T) {
	t.Run("Create Timer With Count", func(t *testing.T) {
		tmr := createTimer(1, func() {}, time.Second, nil, 5)
		assert.Equal(t, int64(1), tmr.id)
		assert.Equal(t, int64(time.Second), tmr.interval)
		assert.Nil(t, tmr.condition)
		assert.Equal(t, int64(5), tmr.counter.Load())
		assert.Greater(t, tmr.when-time.Now().UnixNano(), int64(time.Second-2*time.Millisecond))
	})

	t.Run("Create Timer With Condition", func(t *testing.T) {
		cond := &testCondition{}
		tmr := createTimer(2, func() {}, time.Hour, cond, schedulerapi.Infinite)
		assert.Equal(t, int64(2), tmr.id)
		assert.EqualValues(t, int64(time.Hour), tmr.interval)
		assert.Equal(t, cond, tmr.condition)
		assert.Equal(t, schedulerapi.Infinite, tmr.counter.Load())
		assert.Greater(t, tmr.when-time.Now().UnixNano(), int64(time.Hour-time.Second))
	})

	t.Run("Create Timer With Zero Count", func(t *testing.T) {
		tmr := createTimer(3, func() {}, time.Second, nil, 0)
		assert.Equal(t, int64(0), tmr.counter.Load())
		assert.True(t, tmr.Stopped())
	})
}

// TestTimer_ConditionTimer 测试条件定时器
func TestTimer_ConditionTimer(t *testing.T) {
	cond := &testCondition{}

	t.Run("Condition Timer Basic", func(t *testing.T) {
		var execCount atomic.Int32
		s := NewScheduler("test", time.Millisecond).(*scheduler)
		defer s.Close()

		tmr := newCondTimer(1, cond, func() {
			execCount.Add(1)
		})

		now := time.Now()
		ts := now.UnixNano()

		// 条件不满足时不执行
		cond.shouldTrigger.Store(false)
		tmr.exec(s, now, ts)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, int32(0), execCount.Load())

		// 条件满足时执行
		cond.shouldTrigger.Store(true)
		tmr.exec(s, now, ts)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, int32(1), execCount.Load())
	})

	t.Run("Condition Count Timer", func(t *testing.T) {
		var execCount atomic.Int32
		s := NewScheduler("test", time.Millisecond).(*scheduler)
		defer s.Close()

		tmr := newCondCountTimer(1, cond, 2, func() {
			execCount.Add(1)
		})

		now := time.Now()
		ts := now.UnixNano()

		cond.shouldTrigger.Store(true)

		// 第一次执行
		tmr.exec(s, now, ts)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, int32(1), execCount.Load())
		assert.False(t, tmr.Stopped())

		// 第二次执行
		tmr.exec(s, now, ts)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, int32(2), execCount.Load())
		assert.True(t, tmr.Stopped())

		// 第三次不应该执行
		tmr.exec(s, now, ts)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, int32(2), execCount.Load())
	})
}
