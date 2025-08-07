package service

// Connections  默认连接服务
var Connections Connection = newSnowflakeConnection()

// Connection 连接服务接口
type Connection interface {
	// Increment 增加连接数
	Increment()

	// Decrement 减少连接数
	Decrement()

	// Count 返回当前连接数
	Count() int64

	// Reset 重置连接数
	Reset()

	// SessionID 返回新的会话ID
	SessionID() int64
}
