package scheduler

import (
	"sync"
	"sync/atomic"
	"time"
)

// timerManager 定时器管理器
type timerManager struct {
	incrementCounter atomic.Int64     // Timer ID 自增计数器
	timers           map[int64]*Timer // 全部计数器, 只能在 scheduler 的协程中读写
	mu               sync.Mutex       // 读写 pendingTimers 的锁
	pendingTimers    []*Timer         // 外部创建 Timer 时, 先放到这里边, 等待被 stealTimers 偷走
}

// addTimer 可以在任意协程执行, 添加一个定时器到 pendingTimers 中
func (tm *timerManager) addTimer(t *Timer) {
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
	if len(tm.timers) < 1 {
		return
	}

	// 执行所有计时器任务
	now := time.Now()
	ts := now.UnixNano()
	var closingTimers []int64 // 用于存储需要关闭的计时器 ID
	for id, t := range tm.timers {
		// 执行定时器作业
		t.exec(s, now, ts)

		// 添加到待删除列表
		if t.isStopped() {
			closingTimers = append(closingTimers, id)
		}
	}

	// 从 timers 中删除要关闭的, 下次不再遍历这些
	for _, id := range closingTimers {
		delete(tm.timers, id)
	}
}

// 只能被 scheduler 协程执行, 清空 timers 和 pendingTimers
func (tm *timerManager) close() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 清空
	tm.timers = make(map[int64]*Timer)
	tm.pendingTimers = nil
}

// newTimer 创建一个永久运行的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
func (tm *timerManager) newTimer(interval time.Duration, fn TimerFunc) *Timer {
	if fn == nil {
		panic("nano/timer: nil timer function")
	}
	if interval <= 0 {
		panic("nano/timer: non-positive interval for NewTimer")
	}
	t := newTimer(tm.incrementCounter.Add(1), interval, fn)
	tm.addTimer(t)
	return t
}

// newCountTimer 创建一个执行 count 次的定时器, 每隔 interval 执行一次 fn. 调用 Stop 方法可以停止定时器.
func (tm *timerManager) newCountTimer(interval time.Duration, count int, fn TimerFunc) *Timer {
	if fn == nil {
		panic("nano/timer: nil timer function")
	}
	if interval <= 0 {
		panic("nano/timer: non-positive interval for NewCountTimer")
	}
	t := newCountTimer(tm.incrementCounter.Add(1), interval, count, fn)
	tm.addTimer(t)
	return t
}

// newAfterTimer 创建一个执行 1 次的定时器, 等待 duration 后执行 fn. 调用 Stop 方法可以停止定时器.
func (tm *timerManager) newAfterTimer(duration time.Duration, fn TimerFunc) *Timer {
	if fn == nil {
		panic("nano/timer: nil timer function")
	}
	if duration <= 0 {
		panic("nano/timer: non-positive duration for NewAfterTimer")
	}
	t := newAfterTimer(tm.incrementCounter.Add(1), duration, fn)
	tm.addTimer(t)
	return t
}

// newCondTimer 创建一个无次数限制的条件定时器, 当 condition 满足时, 执行 fn. 调用 Stop 方法可以停止定时器.
func (tm *timerManager) newCondTimer(condition TimerCondition, fn TimerFunc) *Timer {
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
func (tm *timerManager) newCondCountTimer(condition TimerCondition, count int, fn TimerFunc) *Timer {
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
