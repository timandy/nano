package service

// Connections  默认连接服务
var Connections Connection = newSnowflakeConnection()

// Connection 连接服务接口
type Connection interface {
	// SessionID 返回新的会话ID
	SessionID() int64
}
