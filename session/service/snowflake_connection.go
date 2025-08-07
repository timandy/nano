package service

import (
	"github.com/lonng/nano/internal/snowflake"
)

// snowflakeConnection 基于雪花算法的连接服务
type snowflakeConnection struct {
	flake *snowflake.SnowFlake
}

// SessionID 返回新的会话ID
func (c *snowflakeConnection) SessionID() int64 {
	return c.flake.NextId()
}

// newSnowflakeConnection 构造函数
func newSnowflakeConnection() *snowflakeConnection {
	return &snowflakeConnection{flake: snowflake.NewSnowFlake()}
}
