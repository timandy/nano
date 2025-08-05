package schedulerapi

import (
	"time"
)

// Task 定义一个任务类型
type Task func()

// ExecutorState 调度器状态常量
type ExecutorState = int32

const (
	// ExecutorStateCreated 执行器已创建, 但未启动
	ExecutorStateCreated ExecutorState = 0
	// ExecutorStateRunning 执行器正在运行
	ExecutorStateRunning ExecutorState = 1
	// ExecutorStateClosed 执行器已关闭
	ExecutorStateClosed ExecutorState = 2
)

// Executor 执行器接口
type Executor interface {
	// Start 启动执行器
	Start()

	// Close 关闭执行器, 停止所有任务
	Close()

	// State 返回执行器的当前状态
	State() ExecutorState

	// Execute 提交一个任务到执行器
	Execute(task Task) bool
}

// Scheduler 调度器接口
type Scheduler interface {
	// Start 启动调度器
	Start()

	// Close 关闭调度器, 停止所有定时器和任务
	Close()

	// State 返回调度器的当前状态
	State() ExecutorState

	// Execute 提交一个任务到调度器
	Execute(task Task) bool

	//====

	// NewTimer 创建一个永久运行的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
	NewTimer(interval time.Duration, fn TimerFunc) Timer

	// NewCountTimer 创建一个执行 count 次的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
	NewCountTimer(interval time.Duration, count int, fn TimerFunc) Timer

	// NewAfterTimer 创建一个执行 1 次的定时器, 等待 duration 后执行 fn. 调用 Stop 方法可以停止定时器.
	NewAfterTimer(duration time.Duration, fn TimerFunc) Timer

	// NewCondTimer 创建一个无次数限制的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
	NewCondTimer(condition TimerCondition, fn TimerFunc) Timer

	// NewCondCountTimer 创建一个执行 count 次的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
	NewCondCountTimer(condition TimerCondition, count int, fn TimerFunc) Timer

	//====

	// NewTicker 创建一个永久运行的 Ticker, 每隔 interval 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
	NewTicker(interval time.Duration) *Ticker

	// NewCountTicker 创建一个执行 count 次的 Ticker, 每隔 interval 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
	NewCountTicker(interval time.Duration, count int) *Ticker

	// NewAfterTicker 创建一个执行 1 次的 Ticker, 等待 duration 后往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
	NewAfterTicker(duration time.Duration) *Ticker

	// NewCondTicker 创建一个无次数限制的条件 Ticker, 当 condition 满足时, 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
	NewCondTicker(condition TimerCondition) *Ticker

	// NewCondCountTicker 创建一个执行 count 次的条件 Ticker, 当 condition 满足时, 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
	NewCondCountTicker(condition TimerCondition, count int) *Ticker
}
