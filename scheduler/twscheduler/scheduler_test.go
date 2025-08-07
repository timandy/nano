package twscheduler

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/scheduler/schedulerapi"
	"github.com/stretchr/testify/assert"
)

func init() {
	env.Debug = true
}

// TestScheduler 测试时间轮调度器
func TestScheduler(t *testing.T) {
	t.Run("New Scheduler", func(t *testing.T) {
		s := NewScheduler("test-scheduler", time.Millisecond, 16).(*scheduler)
		s.Start()
		s.Start()
		defer s.Close()

		assert.NotNil(t, s)
		assert.Equal(t, "test-scheduler", s.name)
		assert.Equal(t, time.Millisecond, s.tick)
		assert.Eventually(t, func() bool {
			return s.State() == schedulerapi.ExecutorStateRunning
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("New Scheduler with non-power-of-2 slotNum", func(t *testing.T) {
		s := NewScheduler("test", time.Millisecond, 10).(*scheduler)
		defer s.Close()

		// slotNum应该被调整为16（大于10的最小2的幂）
		assert.Equal(t, int64(15), s.tm.slotMask)
		assert.Len(t, s.tm.slots, 16)
	})

	t.Run("Execute Task", func(t *testing.T) {
		s := NewScheduler("test", time.Millisecond, 16)
		s.Start()
		defer s.Close()

		var executed atomic.Bool
		task := func() {
			executed.Store(true)
		}

		assert.True(t, s.Execute(task))

		// 等待任务执行
		assert.Eventually(t, func() bool {
			return executed.Load()
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("Execute Nil Task", func(t *testing.T) {
		s := NewScheduler("test", time.Millisecond, 16)
		s.Start()
		defer s.Close()

		// 应该不会panic
		assert.True(t, s.Execute(nil))
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("Execute Panic Task", func(t *testing.T) {
		s := NewScheduler("test", time.Millisecond, 16)
		s.Start()
		defer s.Close()

		var executed atomic.Bool
		panicTask := func() {
			executed.Store(true)
			panic("test panic")
		}

		// 应该不会导致调度器崩溃
		assert.True(t, s.Execute(panicTask))

		assert.Eventually(t, func() bool {
			return executed.Load()
		}, time.Second, 10*time.Millisecond)

		// 调度器应该仍然可以处理新任务
		var afterPanicExecuted atomic.Bool
		assert.True(t, s.Execute(func() {
			afterPanicExecuted.Store(true)
		}))

		assert.Eventually(t, func() bool {
			return afterPanicExecuted.Load()
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("Close Scheduler", func(t *testing.T) {
		s := NewScheduler("test", time.Millisecond, 16).(*scheduler)
		s.Close()
		assert.Equal(t, schedulerapi.ExecutorStateCreated, s.state.Load())

		s.Start()
		assert.Equal(t, schedulerapi.ExecutorStateRunning, s.state.Load())

		s.Close()
		assert.Equal(t, schedulerapi.ExecutorStateClosed, s.state.Load())

		// 重复关闭应该安全
		s.Close()
		assert.Equal(t, schedulerapi.ExecutorStateClosed, s.state.Load())
	})

	t.Run("Execute after close", func(t *testing.T) {
		s := NewScheduler("test", time.Millisecond, 16)
		s.Start()
		s.Close()

		// 关闭后提交任务应该被忽略
		var executed atomic.Bool
		assert.False(t, s.Execute(func() {
			executed.Store(true)
		}))

		// 等待一段时间确保任务不会被执行
		time.Sleep(100 * time.Millisecond)
		assert.False(t, executed.Load())
	})
}

// TestScheduler_TimerMethods 测试调度器的定时器方法
func TestScheduler_TimerMethods(t *testing.T) {
	s := NewScheduler("test", time.Millisecond, 16)
	s.Start()
	defer s.Close()

	t.Run("NewTimer", func(t *testing.T) {
		var execCount atomic.Int32
		tmr := s.NewTimer(50*time.Millisecond, func() {
			execCount.Add(1)
		})

		assert.NotNil(t, tmr)

		// 等待执行几次
		assert.Eventually(t, func() bool {
			return execCount.Load() >= 2
		}, time.Second, 10*time.Millisecond)

		tmr.Stop()
		currentCount := execCount.Load()

		// 等待一段时间，确保停止后不再执行
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, currentCount, execCount.Load())
	})

	t.Run("NewCountTimer", func(t *testing.T) {
		var execCount atomic.Int32
		tmr := s.NewCountTimer(20*time.Millisecond, 3, func() {
			execCount.Add(1)
		})

		assert.NotNil(t, tmr)

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
		tmr := s.NewAfterTimer(50*time.Millisecond, func() {
			execCount.Add(1)
		})

		assert.NotNil(t, tmr)

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
		cond := &testCondition{shouldTrigger: atomic.Bool{}}
		tmr := s.NewCondTimer(cond, func() {
			execCount.Add(1)
		})

		assert.NotNil(t, tmr)

		// 条件不满足时不执行
		cond.shouldTrigger.Store(false)
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(0), execCount.Load())

		// 条件满足时执行
		cond.shouldTrigger.Store(true)
		assert.Eventually(t, func() bool {
			return execCount.Load() >= 1
		}, time.Second, 10*time.Millisecond)

		tmr.Stop()
	})

	t.Run("NewCondCountTimer", func(t *testing.T) {
		var execCount atomic.Int32
		cond := &testCondition{shouldTrigger: atomic.Bool{}}
		cond.shouldTrigger.Store(true)

		tmr := s.NewCondCountTimer(cond, 2, func() {
			execCount.Add(1)
		})

		assert.NotNil(t, tmr)

		// 等待执行完成
		assert.Eventually(t, func() bool {
			return execCount.Load() == 2
		}, time.Second, 10*time.Millisecond)

		// 再等待一段时间，确保不会执行第3次
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(2), execCount.Load())
	})

	t.Run("Timer panic recovery", func(t *testing.T) {
		var execCount atomic.Int32
		tmr := s.NewTimer(10*time.Millisecond, func() {
			execCount.Add(1)
			panic("timer panic")
		})

		assert.NotNil(t, tmr)

		// 等待执行，调度器应该捕获panic并继续运行
		assert.Eventually(t, func() bool {
			return execCount.Load() >= 1
		}, time.Second, 10*time.Millisecond)

		// 调度器应该仍然可以处理新任务
		var newExec atomic.Bool
		assert.True(t, s.Execute(func() {
			newExec.Store(true)
		}))
		assert.Eventually(t, func() bool {
			return newExec.Load()
		}, time.Second, 10*time.Millisecond)

		tmr.Stop()
	})
}

// TestScheduler_TickerMethods 测试调度器的Ticker方法
func TestScheduler_TickerMethods(t *testing.T) {
	s := NewScheduler("test", time.Millisecond, 16)
	s.Start()
	defer s.Close()

	t.Run("NewTicker", func(t *testing.T) {
		ticker := s.NewTicker(50 * time.Millisecond)
		assert.NotNil(t, ticker)
		assert.NotNil(t, ticker.C)

		// 接收几次tick
		var received atomic.Int32
		go func() {
			for range ticker.C {
				received.Add(1)
			}
		}()

		assert.Eventually(t, func() bool {
			return received.Load() >= 2
		}, time.Second, 10*time.Millisecond)

		ticker.Stop()

		// 验证channel被关闭
		assert.Eventually(t, func() bool {
			select {
			case _, ok := <-ticker.C:
				return !ok // 如果 channel 已关闭，则 ok == false
			default:
				return false // 没有关闭，但也没有值可读
			}
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("NewCountTicker", func(t *testing.T) {
		ticker := s.NewCountTicker(20*time.Millisecond, 3)
		assert.NotNil(t, ticker)
		assert.NotNil(t, ticker.C)

		var received atomic.Int32
		go func() {
			for range ticker.C {
				received.Add(1)
			}
		}()

		// 等待接收3次
		assert.Eventually(t, func() bool {
			return received.Load() == 3
		}, time.Second, 10*time.Millisecond)

		// 验证channel被关闭
		assert.Eventually(t, func() bool {
			select {
			case _, ok := <-ticker.C:
				return !ok // 如果 channel 已关闭，则 ok == false
			default:
				return false // 没有关闭，但也没有值可读
			}
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("NewAfterTicker", func(t *testing.T) {
		ticker := s.NewAfterTicker(50 * time.Millisecond)
		assert.NotNil(t, ticker)
		assert.NotNil(t, ticker.C)

		var received atomic.Int32
		go func() {
			for range ticker.C {
				received.Add(1)
			}
		}()

		// 等待接收1次
		assert.Eventually(t, func() bool {
			return received.Load() == 1
		}, time.Second, 10*time.Millisecond)

		// 验证channel被关闭
		assert.Eventually(t, func() bool {
			select {
			case _, ok := <-ticker.C:
				return !ok // 如果 channel 已关闭，则 ok == false
			default:
				return false // 没有关闭，但也没有值可读
			}
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("NewCondTicker", func(t *testing.T) {
		cond := &testCondition{shouldTrigger: atomic.Bool{}}
		ticker := s.NewCondTicker(cond)
		assert.NotNil(t, ticker)
		assert.NotNil(t, ticker.C)

		var received atomic.Int32
		go func() {
			for range ticker.C {
				received.Add(1)
			}
		}()

		// 条件不满足时不接收
		cond.shouldTrigger.Store(false)
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(0), received.Load())

		// 条件满足时接收
		cond.shouldTrigger.Store(true)
		assert.Eventually(t, func() bool {
			return received.Load() >= 1
		}, time.Second, 10*time.Millisecond)

		ticker.Stop()

		// 验证channel被关闭
		assert.Eventually(t, func() bool {
			select {
			case _, ok := <-ticker.C:
				return !ok // 如果 channel 已关闭，则 ok == false
			default:
				return false // 没有关闭，但也没有值可读
			}
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("NewCondCountTicker", func(t *testing.T) {
		cond := &testCondition{shouldTrigger: atomic.Bool{}}
		cond.shouldTrigger.Store(true)
		ticker := s.NewCondCountTicker(cond, 2)
		assert.NotNil(t, ticker)
		assert.NotNil(t, ticker.C)

		var received atomic.Int32
		go func() {
			for range ticker.C {
				received.Add(1)
			}
		}()

		// 等待接收2次
		assert.Eventually(t, func() bool {
			return received.Load() == 2
		}, time.Second, 10*time.Millisecond)

		// 验证channel被关闭
		assert.Eventually(t, func() bool {
			select {
			case _, ok := <-ticker.C:
				return !ok // 如果 channel 已关闭，则 ok == false
			default:
				return false // 没有关闭，但也没有值可读
			}
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("NewTicker_ConsumeSlow", func(t *testing.T) {
		ticker := s.NewTicker(5 * time.Millisecond)
		assert.NotNil(t, ticker)
		assert.NotNil(t, ticker.C)

		// 接收几次tick
		var received atomic.Int32
		go func() {
			for {
				select {
				case <-ticker.C:
					received.Add(1)
					time.Sleep(2 * time.Second)
				}
			}
		}()

		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, int32(1), received.Load())

		ticker.Stop()

		// 验证channel被关闭
		assert.Eventually(t, func() bool {
			select {
			case _, ok := <-ticker.C:
				return !ok // 如果 channel 已关闭，则 ok == false
			default:
				return false // 没有关闭，但也没有值可读
			}
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("NewCountTicker_ConsumeSlow", func(t *testing.T) {
		ticker := s.NewCountTicker(5*time.Millisecond, 3)
		assert.NotNil(t, ticker)
		assert.NotNil(t, ticker.C)

		var received atomic.Int32
		go func() {
			for {
				select {
				case <-ticker.C:
					received.Add(1)
					time.Sleep(2 * time.Second)
				}
			}
		}()

		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, int32(1), received.Load())

		// 验证channel被关闭
		assert.Eventually(t, func() bool {
			select {
			case _, ok := <-ticker.C:
				return !ok // 如果 channel 已关闭，则 ok == false
			default:
				return false // 没有关闭，但也没有值可读
			}
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("NewAfterTicker_ConsumeSlow", func(t *testing.T) {
		ticker := s.NewAfterTicker(50 * time.Millisecond)
		assert.NotNil(t, ticker)
		assert.NotNil(t, ticker.C)

		var received atomic.Int32
		go func() {
			for {
				select {
				case <-ticker.C:
					received.Add(1)
					time.Sleep(2 * time.Second)
				}
			}
		}()

		// 等待接收1次
		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, int32(1), received.Load())

		// 验证channel被关闭
		assert.Eventually(t, func() bool {
			select {
			case _, ok := <-ticker.C:
				return !ok // 如果 channel 已关闭，则 ok == false
			default:
				return false // 没有关闭，但也没有值可读
			}
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("NewCondTicker_ConsumeSlow", func(t *testing.T) {
		cond := &testCondition{shouldTrigger: atomic.Bool{}}
		ticker := s.NewCondTicker(cond)
		assert.NotNil(t, ticker)
		assert.NotNil(t, ticker.C)

		var received atomic.Int32
		go func() {
			for {
				select {
				case <-ticker.C:
					received.Add(1)
					time.Sleep(2 * time.Second)
				}
			}
		}()

		// 条件不满足时不接收
		cond.shouldTrigger.Store(false)
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(0), received.Load())

		// 条件满足时接收
		cond.shouldTrigger.Store(true)
		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, int32(1), received.Load())

		ticker.Stop()

		// 验证channel被关闭
		assert.Eventually(t, func() bool {
			select {
			case _, ok := <-ticker.C:
				return !ok // 如果 channel 已关闭，则 ok == false
			default:
				return false // 没有关闭，但也没有值可读
			}
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("NewCondCountTicker_ConsumeSlow", func(t *testing.T) {
		cond := &testCondition{shouldTrigger: atomic.Bool{}}
		cond.shouldTrigger.Store(true)
		ticker := s.NewCondCountTicker(cond, 2)
		assert.NotNil(t, ticker)
		assert.NotNil(t, ticker.C)

		var received atomic.Int32
		go func() {
			for {
				select {
				case <-ticker.C:
					received.Add(1)
					time.Sleep(2 * time.Second)
				}
			}
		}()

		// 等待接收2次
		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, int32(1), received.Load())

		// 验证channel被关闭
		assert.Eventually(t, func() bool {
			select {
			case _, ok := <-ticker.C:
				return !ok // 如果 channel 已关闭，则 ok == false
			default:
				return false // 没有关闭，但也没有值可读
			}
		}, 2*time.Second, 10*time.Millisecond)
	})
}

// TestScheduler_ConcurrentOperations 测试并发操作
func TestScheduler_ConcurrentOperations(t *testing.T) {
	s := NewScheduler("test", time.Millisecond, 16)
	s.Start()
	defer s.Close()

	t.Run("Concurrent task execution", func(t *testing.T) {
		const numGoroutines = 100
		const numTasksPerGoroutine = 10

		var counter atomic.Int32
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < numTasksPerGoroutine; j++ {
					assert.True(t, s.Execute(func() {
						counter.Add(1)
					}))
				}
			}()
		}

		wg.Wait()

		// 等待所有任务执行完成
		assert.Eventually(t, func() bool {
			return counter.Load() == numGoroutines*numTasksPerGoroutine
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("Concurrent timer operations", func(t *testing.T) {
		const numGoroutines = 50
		const numTimersPerGoroutine = 5

		var counter atomic.Int32
		var timersMu sync.Mutex
		var timers []schedulerapi.Timer
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < numTimersPerGoroutine; j++ {
					tmr := s.NewTimer(10*time.Millisecond, func() {
						counter.Add(1)
					})
					timersMu.Lock()
					timers = append(timers, tmr)
					timersMu.Unlock()
				}
			}()
		}

		wg.Wait()

		// 等待定时器执行
		time.Sleep(200 * time.Millisecond)

		// 清理所有定时器
		timersMu.Lock()
		for _, tmr := range timers {
			tmr.Stop()
		}
		timersMu.Unlock()

		assert.Greater(t, counter.Load(), int32(0))
	})
}

// TestScheduler_ErrorCases 测试错误情况
func TestScheduler_ErrorCases(t *testing.T) {
	t.Run("NewScheduler with zero tick", func(t *testing.T) {
		assert.Panics(t, func() {
			NewScheduler("test", 0, 16)
		})
	})

	t.Run("NewScheduler with zero slotNum", func(t *testing.T) {
		assert.Panics(t, func() {
			NewScheduler("test", time.Millisecond, 0)
		})
	})

	t.Run("NewScheduler with negative slotNum", func(t *testing.T) {
		assert.Panics(t, func() {
			NewScheduler("test", time.Millisecond, -1)
		})
	})

	t.Run("Timer creation with invalid parameters", func(t *testing.T) {
		s := NewScheduler("test", time.Millisecond, 16)
		s.Start()
		defer s.Close()

		assert.Panics(t, func() {
			s.NewTimer(0, func() {})
		})
		assert.Panics(t, func() {
			s.NewTimer(-time.Second, func() {})
		})
		assert.Panics(t, func() {
			s.NewTimer(time.Second, nil)
		})
		assert.Panics(t, func() {
			s.NewCountTimer(0, 5, func() {})
		})
		assert.Panics(t, func() {
			s.NewAfterTimer(0, func() {})
		})
		assert.Panics(t, func() {
			s.NewCondTimer(nil, func() {})
		})
		assert.Panics(t, func() {
			s.NewCondCountTimer(nil, 5, func() {})
		})
	})

	t.Run("Ticker creation with invalid parameters", func(t *testing.T) {
		s := NewScheduler("test", time.Millisecond, 16)
		s.Start()
		defer s.Close()

		assert.Panics(t, func() {
			s.NewTicker(0)
		})
		assert.Panics(t, func() {
			s.NewCountTicker(0, 5)
		})
		assert.Panics(t, func() {
			s.NewAfterTicker(0)
		})
		assert.Panics(t, func() {
			s.NewCondTicker(nil)
		})
		assert.Panics(t, func() {
			s.NewCondCountTicker(nil, 5)
		})
	})
}

// TestScheduler_Integration 测试集成场景
func TestScheduler_Integration(t *testing.T) {
	s := NewScheduler("integration-test", time.Millisecond, 64)
	s.Start()
	defer s.Close()

	var taskCounter atomic.Int32
	var timerCounter atomic.Int32
	var timerCountCounter atomic.Int32
	var tickerCounter atomic.Int32

	// 添加普通任务
	assert.True(t, s.Execute(func() {
		taskCounter.Add(1)
	}))

	// 添加定时器
	tmr := s.NewTimer(20*time.Millisecond, func() {
		timerCounter.Add(1)
	})
	defer tmr.Stop()

	// 添加计数器定时器
	countTimer := s.NewCountTimer(30*time.Millisecond, 2, func() {
		timerCountCounter.Add(10)
	})
	defer countTimer.Stop()

	// 添加Ticker
	ticker := s.NewCountTicker(25*time.Millisecond, 3)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			tickerCounter.Add(100)
		}
	}()

	// 等待所有操作完成
	assert.Eventually(t, func() bool {
		return taskCounter.Load() == 1 &&
			timerCounter.Load() >= 1 &&
			timerCountCounter.Load() == 20 &&
			tickerCounter.Load() == 300
	}, 2*time.Second, 10*time.Millisecond)
}

// testCondition 测试用的条件
type testCondition struct {
	shouldTrigger atomic.Bool
}

func (c *testCondition) Check(now time.Time) bool {
	return c.shouldTrigger.Load()
}

func (c *testCondition) String() string {
	return "testCondition"
}
