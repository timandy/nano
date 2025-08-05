package scheduler

import (
	"sync"
	"time"

	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/scheduler/defscheduler"
	"github.com/lonng/nano/scheduler/schedulerapi"
	"github.com/lonng/nano/scheduler/twscheduler"
)

// 时间轮常量
const (
	heartbeatSlotNum = 16               // 时间轮, 槽位数
	heartbeatOffset  = 10 * time.Second // 时间轮, 一圈冗余的时间
)

// 默认的全局调度器
var (
	mu        sync.Mutex             //锁
	Default   schedulerapi.Scheduler //默认调度器
	Heartbeat schedulerapi.Scheduler //心跳调度器
)

// Replace 替换默认的调度器
func Replace(s schedulerapi.Scheduler) {
	mu.Lock()
	defer mu.Unlock()

	// 此处可能被绕过
	if s == nil {
		return
	}
	if Default != nil && Default != s {
		Default.Close()
	}
	s.Start()
	Default = s
}

// Start 启动默认的调度器
func Start() {
	mu.Lock()
	defer mu.Unlock()

	// 防止重复启动
	if Default != nil {
		return
	}

	// 初始化默认调度器
	Default = defscheduler.NewScheduler("default", env.TimerPrecision)
	Default.Start()

	// 初始化心跳调度器
	heartbeatTick := (env.HeartbeatInterval + heartbeatOffset) / heartbeatSlotNum
	Heartbeat = twscheduler.NewScheduler("heartbeat", heartbeatTick, heartbeatSlotNum)
	Heartbeat.Start()
}

// Close 关闭, 停止所有定时器和任务
func Close() {
	mu.Lock()
	defer mu.Unlock()

	// 关闭默认调度器
	if Default != nil {
		Default.Close()
	}

	// 关闭心跳调度器
	if Heartbeat != nil {
		Heartbeat.Close()
	}
}

// State 返回调度器的当前状态
func State() schedulerapi.ExecutorState {
	return Default.State()
}

// Execute 提交一个任务到执行器
func Execute(task schedulerapi.Task) bool {
	return Default.Execute(task)
}

//====

// NewTimer 创建一个永久运行的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
func NewTimer(interval time.Duration, fn schedulerapi.TimerFunc) schedulerapi.Timer {
	return Default.NewTimer(interval, fn)
}

// NewCountTimer 创建一个执行 count 次的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
func NewCountTimer(interval time.Duration, count int, fn schedulerapi.TimerFunc) schedulerapi.Timer {
	return Default.NewCountTimer(interval, count, fn)
}

// NewAfterTimer 创建一个执行 1 次的定时器, 等待 duration 后执行 fn. 调用 Stop 方法可以停止定时器.
func NewAfterTimer(duration time.Duration, fn schedulerapi.TimerFunc) schedulerapi.Timer {
	return Default.NewAfterTimer(duration, fn)
}

// NewCondTimer 创建一个无次数限制的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
func NewCondTimer(condition schedulerapi.TimerCondition, fn schedulerapi.TimerFunc) schedulerapi.Timer {
	return Default.NewCondTimer(condition, fn)
}

// NewCondCountTimer 创建一个执行 count 次的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
func NewCondCountTimer(condition schedulerapi.TimerCondition, count int, fn schedulerapi.TimerFunc) schedulerapi.Timer {
	return Default.NewCondCountTimer(condition, count, fn)
}

//====

// NewTicker 创建一个永久运行的 Ticker, 每隔 interval 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func NewTicker(interval time.Duration) *schedulerapi.Ticker {
	return Default.NewTicker(interval)
}

// NewCountTicker 创建一个执行 count 次的 Ticker, 每隔 interval 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func NewCountTicker(interval time.Duration, count int) *schedulerapi.Ticker {
	return Default.NewCountTicker(interval, count)
}

// NewAfterTicker 创建一个执行 1 次的 Ticker, 等待 duration 后往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func NewAfterTicker(duration time.Duration) *schedulerapi.Ticker {
	return Default.NewAfterTicker(duration)
}

// NewCondTicker 创建一个无次数限制的条件 Ticker, 当 condition 满足时, 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func NewCondTicker(condition schedulerapi.TimerCondition) *schedulerapi.Ticker {
	return Default.NewCondTicker(condition)
}

// NewCondCountTicker 创建一个执行 count 次的条件 Ticker, 当 condition 满足时, 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func NewCondCountTicker(condition schedulerapi.TimerCondition, count int) *schedulerapi.Ticker {
	return Default.NewCondCountTicker(condition, count)
}
