package defscheduler

import (
	"sync/atomic"
	"time"

	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/scheduler/schedulerapi"
	"github.com/timandy/routine"
)

// scheduler 调度器
type scheduler struct {
	name    string                 // 调度器名称
	tick    time.Duration          // 最小时间粒度
	state   atomic.Int32           // 调度器状态
	chDie   chan struct{}          // 关闭信号通道
	chTasks chan schedulerapi.Task // 任务队列
	tm      timerManager           // 管理所有的定时器
}

// NewScheduler 构造一个新的调度器, 需要调用 Start() 方法来启动调度器.
func NewScheduler(name string, tick time.Duration) schedulerapi.Scheduler {
	if tick <= 0 {
		panic("tick must > 0")
	}
	s := &scheduler{
		name:    name,
		tick:    tick,
		chDie:   make(chan struct{}),
		chTasks: make(chan schedulerapi.Task, 1<<8),
		tm: timerManager{
			timers: make(map[int64]*timer),
		},
	}
	return s
}

// runTask 执行一个任务, 捕获 panic
func (s *scheduler) runTask(task schedulerapi.Task) {
	if task == nil {
		return
	}
	defer func() {
		if err := recover(); err != nil {
			log.Error("Nano scheduler [%v] execute task error.", s.name, routine.NewRuntimeError(err))
		}
	}()
	task()
}

// runTimerTask 执行一个定时器任务, 捕获 panic
func (s *scheduler) runTimerTask(id int64, fnTimer schedulerapi.TimerFunc, fnTicker schedulerapi.TickerFunc, now time.Time) {
	defer func() {
		if err := recover(); err != nil {
			log.Error("Nano scheduler [%v] execute timer-%v error.", s.name, id, routine.NewRuntimeError(err))
		}
	}()
	if fnTimer != nil {
		fnTimer()
		return
	}
	if fnTicker != nil {
		fnTicker(now)
	}
}

// run 调度器的主循环
func (s *scheduler) run() {
	if env.Debug {
		log.Info("Nano scheduler [%v] staring.", s.name)
	}

	ticker := time.NewTicker(s.tick)
	defer func() {
		ticker.Stop()
		s.tm.close()
		close(s.chTasks)
		if env.Debug {
			log.Info("Nano scheduler [%v] closed.", s.name)
		}
	}()

	for {
		select {
		case <-ticker.C:
			s.tm.cron(s)

		case task := <-s.chTasks:
			s.runTask(task)

		case <-s.chDie:
			return
		}
	}
}

// Start 启动调度器
func (s *scheduler) Start() {
	if !s.state.CompareAndSwap(schedulerapi.ExecutorStateCreated, schedulerapi.ExecutorStateRunning) {
		return
	}

	// 子协程启动循环
	go s.run()
}

// Close 关闭调度器, 停止所有定时器和任务
func (s *scheduler) Close() {
	if !s.state.CompareAndSwap(schedulerapi.ExecutorStateRunning, schedulerapi.ExecutorStateClosed) {
		return
	}
	close(s.chDie)
}

// State 返回调度器的当前状态
func (s *scheduler) State() schedulerapi.ExecutorState {
	return s.state.Load()
}

// Execute 提交一个任务到调度器
func (s *scheduler) Execute(task schedulerapi.Task) bool {
	if s.state.Load() == schedulerapi.ExecutorStateClosed {
		if env.Debug {
			log.Info("Nano scheduler [%v] already closed, new tasks are not accepted.", s.name)
		}
		return false
	}
	s.chTasks <- task
	return true
}

//====

// NewTimer 创建一个永久运行的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
func (s *scheduler) NewTimer(interval time.Duration, fn schedulerapi.TimerFunc) schedulerapi.Timer {
	return s.tm.newTimer(interval, fn)
}

// NewCountTimer 创建一个执行 count 次的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
func (s *scheduler) NewCountTimer(interval time.Duration, count int, fn schedulerapi.TimerFunc) schedulerapi.Timer {
	return s.tm.newCountTimer(interval, count, fn)
}

// NewAfterTimer 创建一个执行 1 次的定时器, 等待 duration 后执行 fn. 调用 Stop 方法可以停止定时器.
func (s *scheduler) NewAfterTimer(duration time.Duration, fn schedulerapi.TimerFunc) schedulerapi.Timer {
	return s.tm.newAfterTimer(duration, fn)
}

// NewCondTimer 创建一个无次数限制的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
func (s *scheduler) NewCondTimer(condition schedulerapi.TimerCondition, fn schedulerapi.TimerFunc) schedulerapi.Timer {
	return s.tm.newCondTimer(condition, fn)
}

// NewCondCountTimer 创建一个执行 count 次的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
func (s *scheduler) NewCondCountTimer(condition schedulerapi.TimerCondition, count int, fn schedulerapi.TimerFunc) schedulerapi.Timer {
	return s.tm.newCondCountTimer(condition, count, fn)
}

//====

// NewTicker 创建一个永久运行的 Ticker, 每隔 interval 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (s *scheduler) NewTicker(interval time.Duration) *schedulerapi.Ticker {
	return s.tm.newTicker(interval)
}

// NewCountTicker 创建一个执行 count 次的 Ticker, 每隔 interval 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (s *scheduler) NewCountTicker(interval time.Duration, count int) *schedulerapi.Ticker {
	return s.tm.newCountTicker(interval, count)
}

// NewAfterTicker 创建一个执行 1 次的 Ticker, 等待 duration 后往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (s *scheduler) NewAfterTicker(duration time.Duration) *schedulerapi.Ticker {
	return s.tm.newAfterTicker(duration)
}

// NewCondTicker 创建一个无次数限制的条件 Ticker, 当 condition 满足时, 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (s *scheduler) NewCondTicker(condition schedulerapi.TimerCondition) *schedulerapi.Ticker {
	return s.tm.newCondTicker(condition)
}

// NewCondCountTicker 创建一个执行 count 次的条件 Ticker, 当 condition 满足时, 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (s *scheduler) NewCondCountTicker(condition schedulerapi.TimerCondition, count int) *schedulerapi.Ticker {
	return s.tm.newCondCountTicker(condition, count)
}
