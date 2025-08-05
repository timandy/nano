package defscheduler

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTimer_BasicFunctionality 测试Timer基本功能
func TestTimer_BasicFunctionality(t *testing.T) {
	t.Run("Timer ID", func(t *testing.T) {
		timer := newTimer(123, time.Second, func() {})
		assert.Equal(t, int64(123), timer.ID())
	})

	t.Run("Timer Stop", func(t *testing.T) {
		timer := newTimer(1, time.Second, func() {})
		assert.False(t, timer.isStopped())

		timer.Stop()
		assert.True(t, timer.isStopped())

		// 重复停止应该安全
		timer.Stop()
		assert.True(t, timer.isStopped())
	})

	t.Run("Timer Countdown", func(t *testing.T) {
		timer := newCountTimer(1, time.Second, 3, func() {})
		assert.False(t, timer.isStopped())

		timer.countdown()
		assert.False(t, timer.isStopped())

		timer.countdown()
		assert.False(t, timer.isStopped())

		timer.countdown()
		assert.True(t, timer.isStopped())
	})

	t.Run("Infinite Timer Countdown", func(t *testing.T) {
		timer := newTimer(1, time.Second, func() {})
		for i := 0; i < 1000; i++ {
			timer.countdown()
		}
		assert.False(t, timer.isStopped())
	})
}

// TestTimer_Execution 测试Timer执行逻辑
func TestTimer_Execution(t *testing.T) {
	t.Run("Interval Timer Execution", func(t *testing.T) {
		var execCount atomic.Int32
		s := NewScheduler("test").(*scheduler)
		defer s.Close()

		timer := newTimer(1, 50*time.Millisecond, func() {
			execCount.Add(1)
		})

		now := time.Now()
		ts := now.UnixNano()

		// 刚创建时不应该执行
		timer.exec(s, now, ts)
		assert.Equal(t, int32(0), execCount.Load())

		// 间隔时间后应该执行
		futureTime := now.Add(100 * time.Millisecond)
		futureTs := futureTime.UnixNano()
		timer.exec(s, futureTime, futureTs)
		time.Sleep(10 * time.Millisecond) // 等待任务执行
		assert.Equal(t, int32(1), execCount.Load())
	})

	t.Run("Stopped Timer Should Not Execute", func(t *testing.T) {
		var execCount atomic.Int32
		s := NewScheduler("test").(*scheduler)
		defer s.Close()

		timer := newTimer(1, 10*time.Millisecond, func() {
			execCount.Add(1)
		})
		timer.Stop()

		futureTime := time.Now().Add(100 * time.Millisecond)
		futureTs := futureTime.UnixNano()
		timer.exec(s, futureTime, futureTs)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, int32(0), execCount.Load())
	})
}

// TestTimer_CreateTimer 测试createTimer函数的所有分支
func TestTimer_CreateTimer(t *testing.T) {
	t.Run("Create Timer With Count", func(t *testing.T) {
		timer := createTimer(1, func() {}, time.Second, nil, 5)
		assert.Equal(t, int64(1), timer.id)
		assert.Equal(t, time.Second, timer.interval)
		assert.Nil(t, timer.condition)
		assert.Equal(t, int64(5), timer.counter.Load())
		assert.Equal(t, int64(time.Second), timer.elapse)
	})

	t.Run("Create Timer With Condition", func(t *testing.T) {
		cond := &testCondition{}
		timer := createTimer(2, func() {}, time.Hour, cond, infinite)
		assert.Equal(t, int64(2), timer.id)
		assert.Equal(t, time.Hour, timer.interval)
		assert.Equal(t, cond, timer.condition)
		assert.Equal(t, infinite, timer.counter.Load())
		assert.Equal(t, int64(time.Hour), timer.elapse)
	})

	t.Run("Create Timer With Zero Count", func(t *testing.T) {
		timer := createTimer(3, func() {}, time.Second, nil, 0)
		assert.Equal(t, int64(0), timer.counter.Load())
		assert.True(t, timer.isStopped())
	})
}

// TestTimer_ConditionTimer 测试条件定时器
func TestTimer_ConditionTimer(t *testing.T) {
	cond := &testCondition{}

	t.Run("Condition Timer Basic", func(t *testing.T) {
		var execCount atomic.Int32
		s := NewScheduler("test").(*scheduler)
		defer s.Close()

		timer := newCondTimer(1, cond, func() {
			execCount.Add(1)
		})

		now := time.Now()
		ts := now.UnixNano()

		// 条件不满足时不执行
		cond.shouldTrigger.Store(false)
		timer.exec(s, now, ts)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, int32(0), execCount.Load())

		// 条件满足时执行
		cond.shouldTrigger.Store(true)
		timer.exec(s, now, ts)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, int32(1), execCount.Load())
	})

	t.Run("Condition Count Timer", func(t *testing.T) {
		var execCount atomic.Int32
		s := NewScheduler("test").(*scheduler)
		defer s.Close()

		timer := newCondCountTimer(1, cond, 2, func() {
			execCount.Add(1)
		})

		now := time.Now()
		ts := now.UnixNano()

		cond.shouldTrigger.Store(true)

		// 第一次执行
		timer.exec(s, now, ts)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, int32(1), execCount.Load())
		assert.False(t, timer.isStopped())

		// 第二次执行
		timer.exec(s, now, ts)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, int32(2), execCount.Load())
		assert.True(t, timer.isStopped())

		// 第三次不应该执行
		timer.exec(s, now, ts)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, int32(2), execCount.Load())
	})
}
