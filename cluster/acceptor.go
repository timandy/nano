package cluster

import (
	"context"
	"net"
	"sync/atomic"

	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/session"
	"github.com/lonng/nano/test/mock"
)

var _ session.NetworkEntity = (*acceptor)(nil)

// acceptor 集群模式中, 工作节点的网络对象
type acceptor struct {
	sid        int64
	node       *Node
	gateClient clusterpb.GateClient
	session    *session.Session
	lastMid    uint64
	rpcHandler rpcHandler
	gateAddr   string
	state      atomic.Int32 // current acceptor state
}

// newAcceptor 构造函数
func newAcceptor(sid int64, node *Node, gateClient clusterpb.GateClient, rpcHandler rpcHandler, gateAddr string) *acceptor {
	a := &acceptor{
		sid:        sid,
		node:       node,
		gateClient: gateClient,
		rpcHandler: rpcHandler,
		gateAddr:   gateAddr,
	}
	a.state.Store(statusWorking)
	return a
}

// RemoteAddr 返回一个假的地址
func (a *acceptor) RemoteAddr() net.Addr {
	return mock.NetAddr{}
}

// LastMid 上次消息 ID
func (a *acceptor) LastMid() uint64 {
	return a.lastMid
}

// RPC 调用集群内的服务
func (a *acceptor) RPC(route string, v any) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}
	// TODO: buffer
	data, err := env.Marshal(v)
	if err != nil {
		return err
	}
	msg := &message.Message{
		Type:  message.Notify,
		Route: route,
		Data:  data,
	}
	a.rpcHandler(a.session, msg, true)
	return nil
}

// Push 调用 Gate, 推送数据给客户端
func (a *acceptor) Push(route string, v any) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}
	// TODO: buffer
	data, err := env.Marshal(v)
	if err != nil {
		return err
	}
	request := &clusterpb.PushMessage{
		SessionId: a.sid,
		Route:     route,
		Data:      data,
	}
	_, err = a.gateClient.HandlePush(context.Background(), request)
	return err
}

// Response 调用 Gate, 返回响应数据给客户端
func (a *acceptor) Response(v any) error {
	return a.ResponseMid(a.lastMid, v)
}

// ResponseMid 调用 Gate, 返回响应数据给客户端
func (a *acceptor) ResponseMid(mid uint64, v any) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}
	// TODO: buffer
	data, err := env.Marshal(v)
	if err != nil {
		return err
	}
	request := &clusterpb.ResponseMessage{
		SessionId: a.sid,
		Id:        mid,
		Data:      data,
	}
	_, err = a.gateClient.HandleResponse(context.Background(), request)
	return err
}

// Kick 推送消息给客户端并关闭连接
func (a *acceptor) Kick(v any) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}

	// 通知 Gate 关闭连接, 不管 kick 消息发送成功与否
	//goland:noinspection GoUnhandledErrorResult
	defer a.Close()

	// TODO: buffer
	data, err := env.Marshal(v)
	if err != nil {
		return err
	}
	request := &clusterpb.KickMessage{
		SessionId: a.sid,
		Data:      data,
	}
	_, err = a.gateClient.HandleKick(context.Background(), request)
	return err
}

// Close 集群模式下, Worker 节点关闭会话, 通知 Gate 也关闭(主动关闭)
func (a *acceptor) Close() error {
	if !a.state.CompareAndSwap(statusWorking, statusClosed) {
		return ErrCloseClosedSession
	}
	// TODO: buffer
	// 先删除
	s, found := a.node.delSession(a.sid)
	// 通知 Gate 关闭连接
	request := &clusterpb.CloseSessionRequest{
		SessionId: a.sid,
	}
	_, err := a.gateClient.CloseSession(context.Background(), request)
	// 触发事件
	if found {
		s.Execute(func() { session.Event.FireSessionClosed(s) }) //异步执行关闭事件
	}
	return err
}

// status 获取当前状态
func (a *acceptor) status() int32 {
	return a.state.Load()
}
