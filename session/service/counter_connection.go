package service

import "sync/atomic"

// counterConnection 基于计数器的连接服务
type counterConnection struct {
	sid atomic.Int64
}

// SessionID 返回新的会话ID
func (c *counterConnection) SessionID() int64 {
	return c.sid.Add(1)
}

// newCounterConnection 构造函数
func newCounterConnection() *counterConnection {
	return &counterConnection{}
}
