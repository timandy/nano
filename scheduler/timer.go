package scheduler

import (
	"math"
	"sync/atomic"
	"time"
)

// infinite 表示定时器永久运行, 用在 Timer.counter 字段
const infinite int64 = math.MaxInt64

// TimerFunc 定时器执行函数类型
type TimerFunc func()

// TimerCondition 定时器执行条件接口
type TimerCondition interface {
	Check(now time.Time) bool
}

//====

// Timer 表示一个定时任务
type Timer struct {
	id        int64          // 定时器 ID
	fn        TimerFunc      // 执行的函数
	createAt  int64          // 定时器创建时间
	interval  time.Duration  // 执行间隔
	condition TimerCondition // 定时器执行条件
	elapse    int64          // 运行时变量, 总的运行时间
	closed    atomic.Bool    // 运行时变量, 定时器是否已关闭
	counter   atomic.Int64   // 运行时变量, 计数器
}

// ID 返回当前定时器的 ID
func (t *Timer) ID() int64 {
	return t.id
}

// Stop 关闭定时器
func (t *Timer) Stop() {
	if !t.closed.CompareAndSwap(false, true) {
		return
	}
	t.counter.Store(0)
}

// isStopped 检查定时器是否已停止
func (t *Timer) isStopped() bool {
	return t.closed.Load() || t.counter.Load() <= 0
}

// countdown 计数器减一
func (t *Timer) countdown() {
	if t.counter.Load() != infinite {
		t.counter.Add(-1)
	}
}

// exec 执行定时器任务
func (t *Timer) exec(s *scheduler, now time.Time, ts int64) {
	//已关闭
	if t.isStopped() {
		return
	}

	// 条件定时器
	if t.condition != nil {
		if t.condition.Check(now) {
			s.runTimerTask(t.id, t.fn)
			t.countdown() // 执行后计数器减一
		}
		return
	}

	// 执行作业
	if ts >= t.createAt+t.elapse {
		t.elapse += int64(t.interval) // 更新总的流逝时间
		s.runTimerTask(t.id, t.fn)
		t.countdown() // 执行后计数器减一
	}
}

// newTimer 创建一个新的定时器
func createTimer(id int64, fn TimerFunc, interval time.Duration, condition TimerCondition, count int64) *Timer {
	timer := &Timer{
		id:        id,
		fn:        fn,
		createAt:  time.Now().UnixNano(),
		interval:  interval,
		condition: condition,
		elapse:    int64(interval), // 第一次执行将在 interval 间隔之后
	}
	if count > 0 {
		timer.counter.Add(count) // 设置计数器, 计数器为 0 时, 定时器会自动停止
	}
	return timer
}

// newTimer 构造函数
func newTimer(id int64, interval time.Duration, fn TimerFunc) *Timer {
	return createTimer(id, fn, interval, nil, infinite)
}

// newCondTimer 构造函数
func newCountTimer(id int64, interval time.Duration, count int, fn TimerFunc) *Timer {
	return createTimer(id, fn, interval, nil, int64(count))
}

// newAfterTimer 构造函数
func newAfterTimer(id int64, interval time.Duration, fn TimerFunc) *Timer {
	return createTimer(id, fn, interval, nil, 1)
}

// newAfterCondTimer 构造函数
func newCondTimer(id int64, condition TimerCondition, fn TimerFunc) *Timer {
	return createTimer(id, fn, math.MaxInt64, condition, infinite)
}

// newCondCountTimer 构造函数
func newCondCountTimer(id int64, condition TimerCondition, count int, fn TimerFunc) *Timer {
	return createTimer(id, fn, math.MaxInt64, condition, int64(count))
}
