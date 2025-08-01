package schedule

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lonng/nano/internal/env"
	"github.com/stretchr/testify/assert"
)

func init() {
	// 设置测试环境的定时器精度
	env.TimerPrecision = 10 * time.Millisecond
	env.Debug = true
	Start()
}

func restore() {
	Replace(NewScheduler("default")) // 恢复
	Start()
}

// 确保默认调度器已初始化
func TestDefaultScheduler_IsInitialized(t *testing.T) {
	assert.NotNil(t, global, "默认调度器应该已初始化")
}

// Replace 正常替换
func TestReplace_Normal(t *testing.T) {
	orig := global
	defer restore()
	mock := NewScheduler("mock")
	Replace(mock)
	assert.Same(t, mock, global, "Replace 后 global 应该指向 mock")
	assert.Equal(t, running, mock.(*scheduler).state.Load())
	assert.Equal(t, closed, orig.(*scheduler).state.Load(), "原调度器状态应为 closed")
}

// Replace 传 nil 不替换
func TestReplace_Nil(t *testing.T) {
	orig := global
	Replace(nil)
	assert.Same(t, orig, global, "传 nil 时不应替换")
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
	Execute(func() { v.Add(1) })
	assert.Eventually(t, func() bool {
		return v.Load() == 1
	}, 500*time.Millisecond, 5*time.Millisecond)
}

// Execute 调度器关闭后不再接收任务
func TestExecute_AfterClose(t *testing.T) {
	defer restore() // 恢复

	Close()

	var v atomic.Int32
	Execute(func() { v.Add(1) })
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
	assert.True(t, timer.isStopped())
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
	assert.True(t, timer.isStopped())
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
	assert.True(t, timer.isStopped())
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
	assert.True(t, timer.isStopped())
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
	assert.True(t, timer.isStopped())
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

// 并发创建大量定时器
func TestConcurrentCreateTimer(t *testing.T) {
	const N = 500
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			timer := NewCountTimer(time.Millisecond, 1, func() {})
			time.Sleep(10 * time.Millisecond)
			timer.Stop()
		}()
	}
	wg.Wait()
	// 确保无泄漏
	assert.Eventually(t, func() bool {
		global.(*scheduler).tm.mu.Lock()
		defer global.(*scheduler).tm.mu.Unlock()
		return len(global.(*scheduler).tm.pendingTimers) == 0
	}, 200*time.Millisecond, 10*time.Millisecond, "pendingTimers 应该被清空")
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
			Execute(func() { sum.Add(1) })
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
	assert.PanicsWithValue(t, "nano/timer: non-positive interval for NewTimer", func() {
		NewTimer(0, func() {})
	})
}

func TestNewAfterTimer_InvalidDuration(t *testing.T) {
	assert.PanicsWithValue(t, "nano/timer: non-positive duration for NewAfterTimer", func() {
		NewAfterTimer(-1, func() {})
	})
}

func TestNewCountTimer_InvalidInterval(t *testing.T) {
	assert.PanicsWithValue(t, "nano/timer: non-positive interval for NewCountTimer", func() {
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
	assert.False(t, tm.isStopped())
	tm.Stop()
	assert.True(t, tm.isStopped())
}
