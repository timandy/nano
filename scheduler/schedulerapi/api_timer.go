package schedulerapi

import (
	"math"
	"time"
)

// Infinite 表示定时器永久运行, 用在 Timer.counter 字段
const Infinite int64 = math.MaxInt64

// TimerFunc 定时器执行函数类型
type TimerFunc func()

// CleanFunc 定时器清理函数类型, 用于在定时器停止时执行一些清理工作
type CleanFunc func()

// TimerCondition 定时器执行条件接口
type TimerCondition interface {
	Check(now time.Time) bool
}

// Timer 表示一个定时器
type Timer interface {
	// ID 返回当前定时器的 ID
	ID() int64

	// Stop 关闭定时器
	Stop()

	// Stopped 检查定时器是否已停止
	Stopped() bool
}
