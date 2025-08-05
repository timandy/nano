package defscheduler

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/lonng/nano/scheduler/schedulerapi"
)

// timerManager 定时器管理器
type timerManager struct {
	incrementCounter atomic.Int64     // timer ID 自增计数器
	timers           map[int64]*timer // 全部计数器, 只能在 scheduler 的协程中读写
	mu               sync.Mutex       // 读写 pendingTimers 的锁
	pendingTimers    []*timer         // 外部创建 timer 时, 先放到这里边, 等待被 stealTimers 偷走
}

// addTimer 可以在任意协程执行, 添加一个定时器到 pendingTimers 中
func (tm *timerManager) addTimer(t *timer) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.pendingTimers = append(tm.pendingTimers, t)
}

// 只能被 scheduler 协程执行, 把 pendingTimers 转移 timers 中
func (tm *timerManager) stealTimers() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, t := range tm.pendingTimers {
		tm.timers[t.id] = t
	}
	tm.pendingTimers = nil
}

// 只能被 scheduler 协程执行, 调度被管理的定时器
func (tm *timerManager) cron(s *scheduler) {
	// 抢夺 timer 到 timers
	tm.stealTimers()

	// 没有定时器, 直接返回
	if len(tm.timers) <= 0 {
		return
	}

	// 执行所有计时器任务
	now := time.Now()
	ts := now.UnixNano()
	var closingTimers []*timer // 用于存储需要关闭的计时器
	for _, t := range tm.timers {
		// 执行定时器作业
		t.exec(s, now, ts)

		// 添加到待删除列表
		if t.Stopped() {
			closingTimers = append(closingTimers, t)
		}
	}

	// 从 timers 中删除要关闭的, 下次不再遍历这些
	for _, t := range closingTimers {
		delete(tm.timers, t.id)
		t.clean()
	}
}

// 只能被 scheduler 协程执行, 清空 timers 和 pendingTimers
func (tm *timerManager) close() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 清空
	tm.timers = make(map[int64]*timer)
	tm.pendingTimers = nil
}

//====

// newTimer 创建一个永久运行的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
func (tm *timerManager) newTimer(interval time.Duration, fn schedulerapi.TimerFunc) *timer {
	if fn == nil {
		panic("nano/timer: nil timer function")
	}
	if interval <= 0 {
		panic("nano/timer: non-positive interval")
	}
	t := newTimer(tm.incrementCounter.Add(1), interval, fn)
	tm.addTimer(t)
	return t
}

// newCountTimer 创建一个执行 count 次的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
func (tm *timerManager) newCountTimer(interval time.Duration, count int, fn schedulerapi.TimerFunc) *timer {
	if fn == nil {
		panic("nano/timer: nil timer function")
	}
	if interval <= 0 {
		panic("nano/timer: non-positive interval")
	}
	t := newCountTimer(tm.incrementCounter.Add(1), interval, count, fn)
	tm.addTimer(t)
	return t
}

// newAfterTimer 创建一个执行 1 次的定时器, 等待 duration 后执行 fn. 调用 Stop 方法可以停止定时器.
func (tm *timerManager) newAfterTimer(duration time.Duration, fn schedulerapi.TimerFunc) *timer {
	if fn == nil {
		panic("nano/timer: nil timer function")
	}
	if duration <= 0 {
		panic("nano/timer: non-positive duration")
	}
	t := newAfterTimer(tm.incrementCounter.Add(1), duration, fn)
	tm.addTimer(t)
	return t
}

// newCondTimer 创建一个无次数限制的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
func (tm *timerManager) newCondTimer(condition schedulerapi.TimerCondition, fn schedulerapi.TimerFunc) *timer {
	if fn == nil {
		panic("nano/timer: nil timer function")
	}
	if condition == nil {
		panic("nano/timer: nil condition")
	}
	t := newCondTimer(tm.incrementCounter.Add(1), condition, fn)
	tm.addTimer(t)
	return t
}

// newCondCountTimer 创建一个执行 count 次的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
func (tm *timerManager) newCondCountTimer(condition schedulerapi.TimerCondition, count int, fn schedulerapi.TimerFunc) *timer {
	if fn == nil {
		panic("nano/timer: nil timer function")
	}
	if condition == nil {
		panic("nano/timer: nil condition")
	}
	t := newCondCountTimer(tm.incrementCounter.Add(1), condition, count, fn)
	tm.addTimer(t)
	return t
}

//====

// NewTicker 创建一个永久运行的 Ticker, 每隔 interval 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (tm *timerManager) newTicker(interval time.Duration) *schedulerapi.Ticker {
	if interval <= 0 {
		panic("nano/timer: non-positive interval")
	}
	t, ticker := newTicker(tm.incrementCounter.Add(1), interval)
	tm.addTimer(t)
	return ticker
}

// NewCountTicker 创建一个执行 count 次的 Ticker, 每隔 interval 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (tm *timerManager) newCountTicker(interval time.Duration, count int) *schedulerapi.Ticker {
	if interval <= 0 {
		panic("nano/timer: non-positive interval")
	}
	t, ticker := newCountTicker(tm.incrementCounter.Add(1), interval, count)
	tm.addTimer(t)
	return ticker
}

// NewAfterTicker 创建一个执行 1 次的 Ticker, 等待 duration 后往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (tm *timerManager) newAfterTicker(duration time.Duration) *schedulerapi.Ticker {
	if duration <= 0 {
		panic("nano/timer: non-positive duration")
	}
	t, ticker := newAfterTicker(tm.incrementCounter.Add(1), duration)
	tm.addTimer(t)
	return ticker
}

// NewCondTicker 创建一个无次数限制的条件 Ticker, 当 condition 满足时, 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (tm *timerManager) newCondTicker(condition schedulerapi.TimerCondition) *schedulerapi.Ticker {
	if condition == nil {
		panic("nano/timer: nil condition")
	}
	t, ticker := newCondTicker(tm.incrementCounter.Add(1), condition)
	tm.addTimer(t)
	return ticker
}

// NewCondCountTicker 创建一个执行 count 次的条件 Ticker, 当 condition 满足时, 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (tm *timerManager) newCondCountTicker(condition schedulerapi.TimerCondition, count int) *schedulerapi.Ticker {
	if condition == nil {
		panic("nano/timer: nil condition")
	}
	t, ticker := newCondCountTicker(tm.incrementCounter.Add(1), condition, count)
	tm.addTimer(t)
	return ticker
}
