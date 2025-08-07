package service

import (
	"sync/atomic"

	"github.com/lonng/nano/internal/snowflake"
)

// snowflakeConnection 基于雪花算法的连接服务
type snowflakeConnection struct {
	count atomic.Int64
	flake *snowflake.SnowFlake
}

// Increment 增加连接数
func (c *snowflakeConnection) Increment() {
	c.count.Add(1)
}

// Decrement 减少连接数
func (c *snowflakeConnection) Decrement() {
	c.count.Add(-1)
}

// Count 返回当前连接数
func (c *snowflakeConnection) Count() int64 {
	return c.count.Load()
}

// Reset 重置连接计数器
func (c *snowflakeConnection) Reset() {
	c.count.Store(0)
}

// SessionID 返回新的会话ID
func (c *snowflakeConnection) SessionID() int64 {
	return c.flake.NextId()
}

// newSnowflakeConnection 构造函数
func newSnowflakeConnection() *snowflakeConnection {
	return &snowflakeConnection{flake: snowflake.NewSnowFlake()}
}
