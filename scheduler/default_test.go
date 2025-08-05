package scheduler

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/scheduler/defscheduler"
	"github.com/lonng/nano/scheduler/schedulerapi"
	"github.com/stretchr/testify/assert"
)

func init() {
	// 设置测试环境的定时器精度
	env.TimerPrecision = 10 * time.Millisecond
	env.Debug = true
	Start()
}

func restore() {
	Replace(defscheduler.NewScheduler("default", time.Millisecond)) // 恢复
	Start()
}

// 确保默认调度器已初始化
func TestDefaultScheduler_IsInitialized(t *testing.T) {
	assert.NotNil(t, Default, "默认调度器应该已初始化")
	assert.Equal(t, schedulerapi.ExecutorStateRunning, State())
}

// Replace 正常替换
func TestReplace_Normal(t *testing.T) {
	orig := Default
	defer restore()
	mock := defscheduler.NewScheduler("mock", time.Millisecond)
	Replace(mock)
	assert.Equal(t, schedulerapi.ExecutorStateRunning, State())
	assert.Same(t, mock, Default, "Replace 后 Default 应该指向 mock")
	assert.Equal(t, schedulerapi.ExecutorStateRunning, mock.State())
	assert.Equal(t, schedulerapi.ExecutorStateClosed, orig.State(), "原调度器状态应为 closed")
}

// Replace 传 nil 不替换
func TestReplace_Nil(t *testing.T) {
	orig := Default
	Replace(nil)
	assert.Same(t, orig, Default, "传 nil 时不应替换")
}

// Start 重复调用不会 panic
func TestStart_Idempotent(t *testing.T) {
	assert.NotPanics(t, func() {
		Start()
		Start()
	})
}

// Close 重复调用不会 panic
func TestClose_Idempotent(t *testing.T) {
	defer restore() // 恢复

	assert.NotPanics(t, func() {
		Close()
		Close()
	})
}

// Execute 提交任务
func TestExecute_Basic(t *testing.T) {
	var v atomic.Int32
	assert.True(t, Execute(func() { v.Add(1) }))
	assert.Eventually(t, func() bool {
		return v.Load() == 1
	}, 500*time.Millisecond, 5*time.Millisecond)
}

// Execute 调度器关闭后不再接收任务
func TestExecute_AfterClose(t *testing.T) {
	defer restore() // 恢复

	Close()

	var v atomic.Int32
	assert.False(t, Execute(func() { v.Add(1) }))
	time.Sleep(20 * time.Millisecond)
	assert.EqualValues(t, 0, v.Load())
}

// NewTimer 永久运行
func TestNewTimer_Infinite(t *testing.T) {
	var cnt atomic.Int32
	timer := NewTimer(10*time.Millisecond, func() { cnt.Add(1) })
	assert.NotNil(t, timer)
	time.Sleep(35 * time.Millisecond)
	assert.GreaterOrEqual(t, cnt.Load(), int32(2))
	timer.Stop()
	assert.True(t, timer.Stopped())
	runCount := cnt.Load()
	time.Sleep(50 * time.Millisecond)
	assert.EqualValues(t, runCount, cnt.Load())
}

// NewCountTimer 固定次数
func TestNewCountTimer_FixedCount(t *testing.T) {
	var cnt atomic.Int32
	timer := NewCountTimer(10*time.Millisecond, 3, func() { cnt.Add(1) })
	assert.NotNil(t, timer)
	time.Sleep(50 * time.Millisecond)
	assert.EqualValues(t, 3, cnt.Load())
	// 计数器到 0，自动停止
	assert.True(t, timer.Stopped())
	runCount := cnt.Load()
	time.Sleep(50 * time.Millisecond)
	assert.EqualValues(t, runCount, cnt.Load())
}

// NewAfterTimer 只执行一次
func TestNewAfterTimer_Once(t *testing.T) {
	var cnt atomic.Int32
	timer := NewAfterTimer(10*time.Millisecond, func() { cnt.Add(1) })
	assert.NotNil(t, timer)
	time.Sleep(50 * time.Millisecond)
	assert.EqualValues(t, 1, cnt.Load())
	assert.True(t, timer.Stopped())
	runCount := cnt.Load()
	time.Sleep(50 * time.Millisecond)
	assert.EqualValues(t, runCount, cnt.Load())
}

// NewCondTimer 条件永远满足
func TestNewCondTimer_AlwaysTrue(t *testing.T) {
	var cnt atomic.Int32
	cond := &testCondition{}
	cond.shouldTrigger.Store(true)
	timer := NewCondTimer(cond, func() { cnt.Add(1) })
	assert.NotNil(t, timer)
	time.Sleep(50 * time.Millisecond)
	assert.Greater(t, cnt.Load(), int32(1))
	timer.Stop()
	assert.True(t, timer.Stopped())
	runCount := cnt.Load()
	time.Sleep(50 * time.Millisecond)
	assert.EqualValues(t, runCount, cnt.Load())
}

// NewCondCountTimer 条件满足且次数限制
func TestNewCondCountTimer_ConditionWithCount(t *testing.T) {
	var cnt atomic.Int32
	cond := &testCondition{}
	cond.shouldTrigger.Store(true)
	timer := NewCondCountTimer(cond, 2, func() { cnt.Add(1) })
	assert.NotNil(t, timer)
	time.Sleep(50 * time.Millisecond)
	assert.EqualValues(t, 2, cnt.Load())
	assert.True(t, timer.Stopped())
	runCount := cnt.Load()
	time.Sleep(50 * time.Millisecond)
	assert.EqualValues(t, runCount, cnt.Load())
}

// Timer Stop 多次调用安全
func TestTimer_StopIdempotent(t *testing.T) {
	timer := NewAfterTimer(30*time.Millisecond, func() {})
	assert.NotPanics(t, func() {
		timer.Stop()
		timer.Stop()
	})
}

// 并发提交任务
func TestConcurrentExecute(t *testing.T) {
	const N = 1000
	var sum atomic.Int32
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			assert.True(t, Execute(func() { sum.Add(1) }))
		}()
	}
	wg.Wait()
	time.Sleep(50 * time.Millisecond)
	assert.Eventually(t, func() bool {
		return sum.Load() == N
	}, 50*time.Millisecond, 5*time.Millisecond, "所有任务应该被执行")
}

// 边界：零或负间隔/时长 panic
func TestNewTimer_InvalidInterval(t *testing.T) {
	assert.PanicsWithValue(t, "nano/timer: non-positive interval", func() {
		NewTimer(0, func() {})
	})
}

func TestNewAfterTimer_InvalidDuration(t *testing.T) {
	assert.PanicsWithValue(t, "nano/timer: non-positive duration", func() {
		NewAfterTimer(-1, func() {})
	})
}

func TestNewCountTimer_InvalidInterval(t *testing.T) {
	assert.PanicsWithValue(t, "nano/timer: non-positive interval", func() {
		NewCountTimer(0, 1, func() {})
	})
}

// 边界：nil 回调/条件 panic
func TestNewTimer_NilFunc(t *testing.T) {
	assert.PanicsWithValue(t, "nano/timer: nil timer function", func() {
		NewTimer(time.Second, nil)
	})
}

func TestNewCondTimer_NilCondition(t *testing.T) {
	assert.PanicsWithValue(t, "nano/timer: nil condition", func() {
		NewCondTimer(nil, func() {})
	})
}

// 条件定时器条件不满足时不执行
func TestNewCondTimer_NeverTrue(t *testing.T) {
	var cnt int32
	cond := &testCondition{}
	tm := NewCondTimer(cond, func() { atomic.AddInt32(&cnt, 1) })
	time.Sleep(50 * time.Millisecond)
	assert.Zero(t, atomic.LoadInt32(&cnt))
	assert.False(t, tm.Stopped())
	tm.Stop()
	assert.True(t, tm.Stopped())
}

// 边界：零或负间隔/时长 panic
func TestNewTicker_InvalidInterval(t *testing.T) {
	assert.PanicsWithValue(t, "nano/timer: non-positive interval", func() {
		NewTicker(0)
	})
}

func TestNewCountTicker_InvalidInterval(t *testing.T) {
	assert.PanicsWithValue(t, "nano/timer: non-positive interval", func() {
		NewCountTicker(0, 1)
	})
}

func TestNewAfterTicker_InvalidDuration(t *testing.T) {
	assert.PanicsWithValue(t, "nano/timer: non-positive duration", func() {
		NewAfterTicker(-1)
	})
}

func TestNewCondTicker_NilCondition(t *testing.T) {
	assert.PanicsWithValue(t, "nano/timer: nil condition", func() {
		NewCondTicker(nil)
	})
}

func TestNewCondCountTicker_NilCondition(t *testing.T) {
	assert.PanicsWithValue(t, "nano/timer: nil condition", func() {
		NewCondCountTicker(nil, 1)
	})
}

// 条件定时器条件不满足时不执行
func TestNewCondTicker_NeverTrue(t *testing.T) {
	var cnt int32
	cond := &testCondition{}
	tm := NewCondTicker(cond)
	go func() {
		for {
			select {
			case <-tm.C:
				atomic.AddInt32(&cnt, 1)
			}
		}
	}()

	time.Sleep(50 * time.Millisecond)
	assert.Zero(t, atomic.LoadInt32(&cnt))
	assert.False(t, tm.Stopped())
	tm.Stop()
	assert.True(t, tm.Stopped())
}

type testCondition struct {
	shouldTrigger atomic.Bool
}

// 实现TimerCondition接口用于测试
func (tc *testCondition) Check(now time.Time) bool {
	return tc.shouldTrigger.Load()
}
