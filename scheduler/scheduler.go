package scheduler

import (
	"sync/atomic"
	"time"

	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/timandy/routine"
)

// Task 定义一个任务类型
type Task func()

// Executor 执行器接口
type Executor interface {
	// Start 启动执行器
	Start()

	// Execute 提交一个任务到执行器
	Execute(task Task)

	// Close 关闭执行器, 停止所有任务
	Close()
}

// Scheduler 调度器接口
type Scheduler interface {
	// Start 启动调度器
	Start()

	// Execute 提交一个任务到调度器
	Execute(task Task)

	// Close 关闭调度器, 停止所有定时器和任务
	Close()

	// NewTimer 创建一个永久运行的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
	NewTimer(interval time.Duration, fn TimerFunc) *Timer

	// NewCountTimer 创建一个执行 count 次的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
	NewCountTimer(interval time.Duration, count int, fn TimerFunc) *Timer

	// NewAfterTimer 创建一个执行 1 次的定时器, 等待 duration 后执行 fn. 调用 Stop 方法可以停止定时器.
	NewAfterTimer(duration time.Duration, fn TimerFunc) *Timer

	// NewCondTimer 创建一个无次数限制的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
	NewCondTimer(condition TimerCondition, fn TimerFunc) *Timer

	// NewCondCountTimer 创建一个执行 count 次的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
	NewCondCountTimer(condition TimerCondition, count int, fn TimerFunc) *Timer
}

//====

// 调度器状态常量
const (
	created int32 = 0
	running int32 = 1
	closed  int32 = 2
)

// scheduler 调度器
type scheduler struct {
	name    string        // 调度器名称
	state   atomic.Int32  // 调度器状态
	chDie   chan struct{} // 关闭信号通道
	chTasks chan Task     // 任务队列
	tm      timerManager  // 管理所有的定时器
}

// NewScheduler 构造一个新的调度器并在子协程启动
func NewScheduler(name string) Scheduler {
	return &scheduler{
		name:    name,
		chDie:   make(chan struct{}),
		chTasks: make(chan Task, 1<<8),
		tm: timerManager{
			timers: make(map[int64]*Timer),
		},
	}
}

// runTask 执行一个任务, 捕获 panic
func (s *scheduler) runTask(task Task) {
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
func (s *scheduler) runTimerTask(id int64, task TimerFunc) {
	if task == nil {
		return
	}
	defer func() {
		if err := recover(); err != nil {
			log.Error("Nano scheduler [%v] execute timer-%v error.", s.name, id, routine.NewRuntimeError(err))
		}
	}()
	task()
}

// run 调度器的主循环
func (s *scheduler) run() {
	if env.Debug {
		log.Info("Nano scheduler [%v] staring.", s.name)
	}

	ticker := time.NewTicker(env.TimerPrecision)
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
	if !s.state.CompareAndSwap(created, running) {
		return
	}

	// 子协程启动循环
	go s.run()
}

// Execute 提交一个任务到调度器
func (s *scheduler) Execute(task Task) {
	if s.state.Load() == closed {
		if env.Debug {
			log.Info("Nano scheduler [%v] already closed, new tasks are not accepted.", s.name)
		}
		return
	}
	s.chTasks <- task
}

// Close 关闭调度器, 停止所有定时器和任务
func (s *scheduler) Close() {
	if !s.state.CompareAndSwap(running, closed) {
		return
	}
	close(s.chDie)
}

// NewTimer 创建一个永久运行的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
func (s *scheduler) NewTimer(interval time.Duration, fn TimerFunc) *Timer {
	return s.tm.newTimer(interval, fn)
}

// NewCountTimer 创建一个执行 count 次的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
func (s *scheduler) NewCountTimer(interval time.Duration, count int, fn TimerFunc) *Timer {
	return s.tm.newCountTimer(interval, count, fn)
}

// NewAfterTimer 创建一个执行 1 次的定时器, 等待 duration 后执行 fn. 调用 Stop 方法可以停止定时器.
func (s *scheduler) NewAfterTimer(duration time.Duration, fn TimerFunc) *Timer {
	return s.tm.newAfterTimer(duration, fn)
}

// NewCondTimer 创建一个无次数限制的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
func (s *scheduler) NewCondTimer(condition TimerCondition, fn TimerFunc) *Timer {
	return s.tm.newCondTimer(condition, fn)
}

// NewCondCountTimer 创建一个执行 count 次的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
func (s *scheduler) NewCondCountTimer(condition TimerCondition, count int, fn TimerFunc) *Timer {
	return s.tm.newCondCountTimer(condition, count, fn)
}
