package defscheduler

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestScheduler 测试调度器
func TestScheduler(t *testing.T) {
	t.Run("New Scheduler", func(t *testing.T) {
		s := NewScheduler("test-scheduler").(*scheduler)
		s.Start()
		defer s.Close()

		assert.NotNil(t, s)
		assert.Equal(t, "test-scheduler", s.name)
		assert.Eventually(t, func() bool {
			return s.state.Load() == running
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("Execute Task", func(t *testing.T) {
		s := NewScheduler("test")
		s.Start()
		defer s.Close()

		var executed atomic.Bool
		task := func() {
			executed.Store(true)
		}

		s.Execute(task)

		// 等待任务执行
		assert.Eventually(t, func() bool {
			return executed.Load()
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("Execute Nil Task", func(t *testing.T) {
		s := NewScheduler("test")
		s.Start()
		defer s.Close()

		// 应该不会panic
		s.Execute(nil)
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("Execute Panic Task", func(t *testing.T) {
		s := NewScheduler("test")
		s.Start()
		defer s.Close()

		var executed atomic.Bool
		panicTask := func() {
			executed.Store(true)
			panic("test panic")
		}

		// 应该不会导致调度器崩溃
		s.Execute(panicTask)

		assert.Eventually(t, func() bool {
			return executed.Load()
		}, time.Second, 10*time.Millisecond)

		// 调度器应该仍然可以处理新任务
		var afterPanicExecuted atomic.Bool
		s.Execute(func() {
			afterPanicExecuted.Store(true)
		})

		assert.Eventually(t, func() bool {
			return afterPanicExecuted.Load()
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("Close Scheduler", func(t *testing.T) {
		s := NewScheduler("test").(*scheduler)
		s.Close()
		assert.Equal(t, created, s.state.Load())

		s.Start()
		assert.Equal(t, running, s.state.Load())

		s.Close()
		assert.Equal(t, closed, s.state.Load())

		// 重复关闭应该安全
		s.Close()
		assert.Equal(t, closed, s.state.Load())
	})
}

// TestScheduler_TimerMethods 测试调度器的定时器方法
func TestScheduler_TimerMethods(t *testing.T) {
	s := NewScheduler("test")
	s.Start()
	defer s.Close()

	t.Run("NewTimer", func(t *testing.T) {
		var execCount atomic.Int32
		timer := s.NewTimer(50*time.Millisecond, func() {
			execCount.Add(1)
		})

		assert.NotNil(t, timer)

		// 等待执行几次
		assert.Eventually(t, func() bool {
			return execCount.Load() >= 2
		}, time.Second, 10*time.Millisecond)

		timer.Stop()
		currentCount := execCount.Load()

		// 等待一段时间，确保停止后不再执行
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, currentCount, execCount.Load())
	})

	t.Run("NewCountTimer", func(t *testing.T) {
		var execCount atomic.Int32
		timer := s.NewCountTimer(20*time.Millisecond, 3, func() {
			execCount.Add(1)
		})

		assert.NotNil(t, timer)

		// 等待执行完成
		assert.Eventually(t, func() bool {
			return execCount.Load() == 3
		}, time.Second, 10*time.Millisecond)

		// 再等待一段时间，确保不会执行第4次
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(3), execCount.Load())
	})

	t.Run("NewAfterTimer", func(t *testing.T) {
		var execCount atomic.Int32
		timer := s.NewAfterTimer(50*time.Millisecond, func() {
			execCount.Add(1)
		})

		assert.NotNil(t, timer)

		// 等待执行
		assert.Eventually(t, func() bool {
			return execCount.Load() == 1
		}, time.Second, 10*time.Millisecond)

		// 再等待一段时间，确保只执行一次
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(1), execCount.Load())
	})

	t.Run("NewCondTimer", func(t *testing.T) {
		var execCount atomic.Int32
		cond := &testCondition{}
		timer := s.NewCondTimer(cond, func() {
			execCount.Add(1)
		})

		assert.NotNil(t, timer)

		// 条件不满足时不执行
		cond.shouldTrigger.Store(false)
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(0), execCount.Load())

		// 条件满足时执行
		cond.shouldTrigger.Store(true)
		assert.Eventually(t, func() bool {
			return execCount.Load() >= 1
		}, time.Second, 10*time.Millisecond)

		timer.Stop()
	})

	t.Run("NewCondCountTimer", func(t *testing.T) {
		var execCount atomic.Int32
		cond := &testCondition{}
		cond.shouldTrigger.Store(true)

		timer := s.NewCondCountTimer(cond, 2, func() {
			execCount.Add(1)
		})

		assert.NotNil(t, timer)

		// 等待执行完成
		assert.Eventually(t, func() bool {
			return execCount.Load() == 2
		}, time.Second, 10*time.Millisecond)

		// 再等待一段时间，确保不会执行第3次
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(2), execCount.Load())
	})
}

// TestScheduler_Integration 集成测试
func TestScheduler_Integration(t *testing.T) {
	t.Run("Mixed Tasks and Timers", func(t *testing.T) {
		s := NewScheduler("integration-test")
		s.Start()
		defer s.Close()

		var taskCount atomic.Int32
		var timerCount atomic.Int32

		// 添加普通任务
		for i := 0; i < 10; i++ {
			s.Execute(func() {
				taskCount.Add(1)
			})
		}

		// 添加定时器
		timer := s.NewCountTimer(20*time.Millisecond, 5, func() {
			timerCount.Add(1)
		})

		// 等待所有任务完成
		assert.Eventually(t, func() bool {
			return taskCount.Load() == 10 && timerCount.Load() == 5
		}, 2*time.Second, 10*time.Millisecond)

		assert.NotNil(t, timer)
	})

	t.Run("High Concurrency Stress Test", func(t *testing.T) {
		s := NewScheduler("stress-test")
		s.Start()
		defer s.Close()

		const numGoroutines = 100
		const numTasksPerGoroutine = 50

		var totalExecuted atomic.Int32
		var wg sync.WaitGroup

		// 并发提交任务
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < numTasksPerGoroutine; j++ {
					s.Execute(func() {
						totalExecuted.Add(1)
					})
				}
			}()
		}

		// 同时创建定时器
		var timerExecuted atomic.Int32
		timers := make([]*Timer, 20)
		for i := 0; i < 20; i++ {
			timers[i] = s.NewCountTimer(10*time.Millisecond, 1, func() {
				timerExecuted.Add(1)
			})
		}

		wg.Wait()

		// 等待所有任务和定时器执行完成
		assert.Eventually(t, func() bool {
			return totalExecuted.Load() == numGoroutines*numTasksPerGoroutine &&
				timerExecuted.Load() == 20
		}, 5*time.Second, 50*time.Millisecond)

		for _, timer := range timers {
			assert.NotNil(t, timer)
		}
	})
}

// TestScheduler_EdgeCases 边界测试
func TestScheduler_EdgeCases(t *testing.T) {
	t.Run("Scheduler start twice", func(t *testing.T) {
		s := NewScheduler("edge-test").(*scheduler)
		s.Start()
		defer s.Close()

		time.Sleep(50 * time.Millisecond)
		num := runtime.NumGoroutine()

		s.Start()
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, num, runtime.NumGoroutine())
	})

	t.Run("Scheduler execute after closed", func(t *testing.T) {
		s := NewScheduler("edge-test")
		s.Start()

		var taskCount atomic.Int32
		s.Execute(func() {
			taskCount.Add(1)
		})
		assert.Eventually(t, func() bool {
			return taskCount.Load() == 1
		}, 5*time.Second, 1*time.Millisecond)

		s.Close()

		// 再次调度任务，应该不会panic, 并且不会执行
		s.Execute(func() {
			taskCount.Add(1)
		})
		time.Sleep(5 * time.Millisecond)
		assert.Equal(t, int32(1), taskCount.Load())
	})

	t.Run("Timer With Very Short Interval", func(t *testing.T) {
		s := NewScheduler("edge-test")
		s.Start()
		defer s.Close()

		var execCount atomic.Int32
		timer := s.NewCountTimer(time.Nanosecond, 10, func() {
			execCount.Add(1)
		})

		assert.Eventually(t, func() bool {
			return execCount.Load() == 10
		}, time.Second, 10*time.Millisecond)

		timer.Stop() // 即使已经执行完也应该安全
	})

	t.Run("Timer With Very Long Interval", func(t *testing.T) {
		s := NewScheduler("edge-test")
		s.Start()
		defer s.Close()

		var execCount atomic.Int32
		timer := s.NewTimer(time.Hour, func() {
			execCount.Add(1)
		})

		// 短时间内不应该执行
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(0), execCount.Load())

		timer.Stop()
	})

	t.Run("Stop Timer Immediately After Creation", func(t *testing.T) {
		s := NewScheduler("edge-test")
		s.Start()
		defer s.Close()

		var execCount atomic.Int32
		timer := s.NewTimer(10*time.Millisecond, func() {
			execCount.Add(1)
		})

		timer.Stop() // 立即停止

		// 等待一段时间，确保不执行
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(0), execCount.Load())
	})

	t.Run("Multiple Schedulers", func(t *testing.T) {
		s1 := NewScheduler("test1")
		s2 := NewScheduler("test2")
		s1.Start()
		s2.Start()
		defer s1.Close()
		defer s2.Close()

		var count1, count2 atomic.Int32

		s1.Execute(func() { count1.Add(1) })
		s2.Execute(func() { count2.Add(1) })

		assert.Eventually(t, func() bool {
			return count1.Load() == 1 && count2.Load() == 1
		}, time.Second, 10*time.Millisecond)
	})
}

// TestTimerPanicHandling 测试定时器panic处理
func TestTimerPanicHandling(t *testing.T) {
	s := NewScheduler("panic-test")
	s.Start()
	defer s.Close()

	var normalExecuted atomic.Bool

	// 创建会panic的定时器
	panicTimer := s.NewCountTimer(20*time.Millisecond, 1, func() {
		panic("timer panic")
	})

	// 创建正常的定时器
	normalTimer := s.NewCountTimer(30*time.Millisecond, 1, func() {
		normalExecuted.Store(true)
	})

	// 等待正常定时器执行，验证panic没有影响调度器
	assert.Eventually(t, func() bool {
		return normalExecuted.Load()
	}, time.Second, 10*time.Millisecond)

	assert.NotNil(t, panicTimer)
	assert.NotNil(t, normalTimer)
}

// BenchmarkScheduler 性能测试
func BenchmarkScheduler(b *testing.B) {
	s := NewScheduler("benchmark")
	s.Start()
	defer s.Close()

	b.Run("Execute Tasks", func(b *testing.B) {
		var counter atomic.Int64

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				s.Execute(func() {
					counter.Add(1)
				})
			}
		})
	})

	b.Run("Create Timers", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			timer := s.NewAfterTimer(time.Hour, func() {})
			timer.Stop()
		}
	})
}

type testCondition struct {
	shouldTrigger atomic.Bool
}

// 实现TimerCondition接口用于测试
func (tc *testCondition) Check(now time.Time) bool {
	return tc.shouldTrigger.Load()
}
