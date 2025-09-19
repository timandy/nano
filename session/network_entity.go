package session

import "net"

// NetworkEntity 底层网络对象
type NetworkEntity interface {
	RemoteAddr() net.Addr
	LastMid() uint64
	RPC(route string, v any) error
	Push(route string, v any) error
	Response(v any) error
	ResponseMid(mid uint64, v any) error
	Kick(v any) error
	Close() error
}
