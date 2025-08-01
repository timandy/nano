package schedule

import (
	"time"
)

// global 默认的全局调度器
var global = NewScheduler("default")

// Replace 替换默认的调度器
func Replace(s Scheduler) {
	// 此处可能被绕过
	if s == nil {
		return
	}
	if global != nil && global != s {
		global.Close()
	}
	s.Start()
	global = s
}

// Start 启动默认的调度器
func Start() {
	global.Start()
}

// Execute 提交一个任务到执行器
func Execute(task Task) {
	global.Execute(task)
}

// Close 关闭, 停止所有定时器和任务
func Close() {
	global.Close()
}

// NewTimer 创建一个永久运行的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
func NewTimer(interval time.Duration, fn TimerFunc) *Timer {
	return global.NewTimer(interval, fn)
}

// NewCountTimer 创建一个执行 count 次的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
func NewCountTimer(interval time.Duration, count int, fn TimerFunc) *Timer {
	return global.NewCountTimer(interval, count, fn)
}

// NewAfterTimer 创建一个执行 1 次的定时器, 等待 duration 后执行 fn. 调用 Stop 方法可以停止定时器.
func NewAfterTimer(duration time.Duration, fn TimerFunc) *Timer {
	return global.NewAfterTimer(duration, fn)
}

// NewCondTimer 创建一个无次数限制的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
func NewCondTimer(condition TimerCondition, fn TimerFunc) *Timer {
	return global.NewCondTimer(condition, fn)
}

// NewCondCountTimer 创建一个执行 count 次的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
func NewCondCountTimer(condition TimerCondition, count int, fn TimerFunc) *Timer {
	return global.NewCondCountTimer(condition, count, fn)
}
