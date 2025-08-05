package defscheduler

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/lonng/nano/scheduler/schedulerapi"
)

// timer 表示一个定时任务
type timer struct {
	id        int64                       // 定时器 ID
	fnTimer   schedulerapi.TimerFunc      // 执行的 Timer 函数
	fnTicker  schedulerapi.TickerFunc     // 执行的 Ticker 函数
	fnClean   schedulerapi.CleanFunc      // 停止时的 Clean 函数
	condition schedulerapi.TimerCondition // 定时器执行条件
	when      int64                       // 绝对触发时间(ns)
	interval  int64                       // 周期任务间隔(ns)
	closed    atomic.Bool                 // 运行时变量, 定时器是否已关闭
	counter   atomic.Int64                // 运行时变量, 计数器
}

// ID 返回当前定时器的 ID
func (t *timer) ID() int64 {
	return t.id
}

// Stop 关闭定时器
func (t *timer) Stop() {
	if !t.closed.CompareAndSwap(false, true) {
		return
	}
	t.counter.Store(0)
}

// Stopped 检查定时器是否已停止
func (t *timer) Stopped() bool {
	return t.closed.Load() || t.counter.Load() <= 0
}

// countdown 计数器减 1
func (t *timer) countdown() {
	if t.counter.Load() != schedulerapi.Infinite {
		t.counter.Add(-1)
	}
}

// exec 执行定时器任务
func (t *timer) exec(s *scheduler, now time.Time, ts int64) {
	//已关闭
	if t.Stopped() {
		return
	}

	// 条件定时器
	if t.condition != nil {
		if t.condition.Check(now) {
			s.runTimerTask(t.id, t.fnTimer, t.fnTicker, now)
			t.countdown() // 执行后计数器减一
		}
		return
	}

	// 执行作业
	if ts >= t.when {
		t.when += t.interval // 更新总的流逝时间
		s.runTimerTask(t.id, t.fnTimer, t.fnTicker, now)
		t.countdown() // 执行后计数器减一
	}
}

// clean 执行定时器的清理任务
func (t *timer) clean() {
	if t.fnClean != nil {
		t.fnClean()
	}
}

//====

// newTimer 创建一个新的定时器
func createTimer(id int64, fn schedulerapi.TimerFunc, interval time.Duration, condition schedulerapi.TimerCondition, count int64) *timer {
	t := &timer{
		id:        id,
		fnTimer:   fn,
		condition: condition,
		when:      time.Now().Add(interval).UnixNano(),
		interval:  int64(interval),
	}
	if count > 0 {
		t.counter.Add(count) // 设置计数器, 计数器为 0 时, 定时器会自动停止
	}
	return t
}

// newTimer 构造函数
func newTimer(id int64, interval time.Duration, fn schedulerapi.TimerFunc) *timer {
	return createTimer(id, fn, interval, nil, schedulerapi.Infinite)
}

// newCondTimer 构造函数
func newCountTimer(id int64, interval time.Duration, count int, fn schedulerapi.TimerFunc) *timer {
	return createTimer(id, fn, interval, nil, int64(count))
}

// newAfterTimer 构造函数
func newAfterTimer(id int64, interval time.Duration, fn schedulerapi.TimerFunc) *timer {
	return createTimer(id, fn, interval, nil, 1)
}

// newAfterCondTimer 构造函数
func newCondTimer(id int64, condition schedulerapi.TimerCondition, fn schedulerapi.TimerFunc) *timer {
	return createTimer(id, fn, math.MaxInt64, condition, schedulerapi.Infinite)
}

// newCondCountTimer 构造函数
func newCondCountTimer(id int64, condition schedulerapi.TimerCondition, count int, fn schedulerapi.TimerFunc) *timer {
	return createTimer(id, fn, math.MaxInt64, condition, int64(count))
}

//====

// createTicker 创建一个新的 Ticker
func createTicker(id int64, interval time.Duration, condition schedulerapi.TimerCondition, count int64) (*timer, *schedulerapi.Ticker) {
	c := make(chan time.Time, 1)
	fnTicker := func(now time.Time) {
		select {
		case c <- now:
		default:
		}
	}
	fnClean := func() {
		close(c)
	}
	t := &timer{
		id:        id,
		fnTicker:  fnTicker,
		fnClean:   fnClean,
		condition: condition,
		when:      time.Now().Add(interval).UnixNano(),
		interval:  int64(interval),
	}
	if count > 0 {
		t.counter.Add(count) // 设置计数器, 计数器为 0 时, 定时器会自动停止
	}
	return t, schedulerapi.NewTicker(c, t)
}

// newTicker 构造函数
func newTicker(id int64, interval time.Duration) (*timer, *schedulerapi.Ticker) {
	return createTicker(id, interval, nil, schedulerapi.Infinite)
}

// newCountTicker 构造函数
func newCountTicker(id int64, interval time.Duration, count int) (*timer, *schedulerapi.Ticker) {
	return createTicker(id, interval, nil, int64(count))
}

// newAfterTicker 构造函数
func newAfterTicker(id int64, interval time.Duration) (*timer, *schedulerapi.Ticker) {
	return createTicker(id, interval, nil, 1)
}

// newCondTicker 构造函数
func newCondTicker(id int64, condition schedulerapi.TimerCondition) (*timer, *schedulerapi.Ticker) {
	return createTicker(id, math.MaxInt64, condition, schedulerapi.Infinite)
}

// newCondCountTicker 构造函数
func newCondCountTicker(id int64, condition schedulerapi.TimerCondition, count int) (*timer, *schedulerapi.Ticker) {
	return createTicker(id, math.MaxInt64, condition, int64(count))
}
