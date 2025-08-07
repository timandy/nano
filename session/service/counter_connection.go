package service

import "sync/atomic"

// counterConnection 基于计数器的连接服务
type counterConnection struct {
	count atomic.Int64
	sid   atomic.Int64
}

// Increment 增加连接数
func (c *counterConnection) Increment() {
	c.count.Add(1)
}

// Decrement 减少连接数
func (c *counterConnection) Decrement() {
	c.count.Add(-1)
}

// Count 返回当前连接数
func (c *counterConnection) Count() int64 {
	return c.count.Load()
}

// Reset 重置连接计数器
func (c *counterConnection) Reset() {
	c.count.Store(0)
}

// SessionID 返回新的会话ID
func (c *counterConnection) SessionID() int64 {
	return c.sid.Add(1)
}

// newCounterConnection 构造函数
func newCounterConnection() *counterConnection {
	return &counterConnection{}
}
