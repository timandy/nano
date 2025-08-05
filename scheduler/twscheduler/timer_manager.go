package twscheduler

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/lonng/nano/scheduler/schedulerapi"
)

// timerManager 定时器管理器
type timerManager struct {
	incrementCounter atomic.Int64 // timer ID 自增计数器
	tick             int64        // 最小时间粒度(ns)
	slotMask         int64        // 槽位数量掩码, 用于快速计算槽位索引, slotNum(2^n) - 1
	slots            []*slot      // 槽位数组
	current          int64        // 槽位指针
	currentMu        sync.Mutex   // 槽位指针锁
}

// addTimer 根据当前时间, 移动定时器槽位; 调用方, 必须持有定时器所在槽位的锁或定时器不属于任何槽位
func (tm *timerManager) addTimer(t *timer, delay int64) {
	// 取消注册
	origSlt := t.slot
	t.unlink()

	// 计算剩余槽位
	remain := delay / tm.tick
	if remain <= 0 {
		remain = 1
	}

	// 计算槽位下标
	idx := (tm.current + remain) & tm.slotMask
	slt := tm.slots[idx]

	// 锁住目标槽位; 目标槽位不是原槽位时才加锁, 防止死锁
	if slt != origSlt {
		slt.mu.Lock()
		defer slt.mu.Unlock()
	}

	// 添加到目标槽位
	slt.link(t)
}

// next 返回当前槽位, 并推进指针到下一个槽位
func (tm *timerManager) next() *slot {
	tm.currentMu.Lock()
	defer tm.currentMu.Unlock()

	slotIdx := tm.current
	tm.current = (tm.current + 1) & tm.slotMask
	return tm.slots[slotIdx]
}

// 推进指针并执行到期任务
func (tm *timerManager) advance(s *scheduler) {
	// 获取当前槽位, 并推进指针到下一个槽位
	slt := tm.next()

	// 锁定当前槽位
	slt.mu.Lock()
	defer slt.mu.Unlock()

	// 空槽位
	head := slt.head
	if head == nil {
		return
	}

	// 遍历执行
	now := time.Now()
	ts := now.UnixNano()
	var next *timer
	for t := head; t != nil; t = next {
		// 提前保存 next，防止 t 被 unlink 后 next 丢失
		next = t.next

		// 触发
		t.exec(s, now, ts)

		// 已经停止
		if t.Stopped() {
			t.unlink()
			t.clean()
			continue
		}

		// 调整槽位
		tm.addTimer(t, t.when-ts)
	}
}

// 只能被 scheduler 协程执行, 清空 timers 和 pendingTimers
func (tm *timerManager) close() {
	tm.currentMu.Lock()
	defer tm.currentMu.Unlock()

	// 清空
	slotNum := len(tm.slots)
	tm.slots = make([]*slot, slotNum)
	for i := 0; i < slotNum; i++ {
		tm.slots[i] = &slot{}
	}
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
	tm.addTimer(t, int64(interval))
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
	tm.addTimer(t, int64(interval))
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
	tm.addTimer(t, int64(duration))
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
	tm.addTimer(t, 0)
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
	tm.addTimer(t, 0)
	return t
}

//====

// NewTicker 创建一个永久运行的 Ticker, 每隔 interval 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (tm *timerManager) newTicker(interval time.Duration) *schedulerapi.Ticker {
	if interval <= 0 {
		panic("nano/timer: non-positive interval")
	}
	t, ticker := newTicker(tm.incrementCounter.Add(1), interval)
	tm.addTimer(t, int64(interval))
	return ticker
}

// NewCountTicker 创建一个执行 count 次的 Ticker, 每隔 interval 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (tm *timerManager) newCountTicker(interval time.Duration, count int) *schedulerapi.Ticker {
	if interval <= 0 {
		panic("nano/timer: non-positive interval")
	}
	t, ticker := newCountTicker(tm.incrementCounter.Add(1), interval, count)
	tm.addTimer(t, int64(interval))
	return ticker
}

// NewAfterTicker 创建一个执行 1 次的 Ticker, 等待 duration 后往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (tm *timerManager) newAfterTicker(duration time.Duration) *schedulerapi.Ticker {
	if duration <= 0 {
		panic("nano/timer: non-positive duration")
	}
	t, ticker := newAfterTicker(tm.incrementCounter.Add(1), duration)
	tm.addTimer(t, int64(duration))
	return ticker
}

// NewCondTicker 创建一个无次数限制的条件 Ticker, 当 condition 满足时, 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (tm *timerManager) newCondTicker(condition schedulerapi.TimerCondition) *schedulerapi.Ticker {
	if condition == nil {
		panic("nano/timer: nil condition")
	}
	t, ticker := newCondTicker(tm.incrementCounter.Add(1), condition)
	tm.addTimer(t, 0)
	return ticker
}

// NewCondCountTicker 创建一个执行 count 次的条件 Ticker, 当 condition 满足时, 往 C 发送一次当前时间. 调用 Stop 方法可以停止 Ticker.
func (tm *timerManager) newCondCountTicker(condition schedulerapi.TimerCondition, count int) *schedulerapi.Ticker {
	if condition == nil {
		panic("nano/timer: nil condition")
	}
	t, ticker := newCondCountTicker(tm.incrementCounter.Add(1), condition, count)
	tm.addTimer(t, 0)
	return ticker
}
