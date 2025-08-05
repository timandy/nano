package schedulerapi

import "time"

// Ticker 实现 Timer 接口
var _ Timer = (*Ticker)(nil)

// TickerFunc 定时器执行函数类型
type TickerFunc func(now time.Time)

// Ticker 时钟结构, 每隔一定时间往 channel 发送当前时间
type Ticker struct {
	C     <-chan time.Time // The channel on which the ticks are delivered.
	timer Timer
}

// ID 返回当前时钟的 ID
func (t *Ticker) ID() int64 {
	return t.timer.ID()
}

// Stopped 检查时钟是否已停止
func (t *Ticker) Stopped() bool {
	return t.timer.Stopped()
}

// Stop 停止时钟
func (t *Ticker) Stop() {
	t.timer.Stop() // 此时不要关闭 C, timer 关闭时会自动关闭 C
}

// NewTicker 构造函数
func NewTicker(c <-chan time.Time, timer Timer) *Ticker {
	return &Ticker{
		C:     c,
		timer: timer,
	}
}
