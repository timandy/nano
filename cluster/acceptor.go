package cluster

import (
	"context"
	"net"

	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/session"
	"github.com/lonng/nano/test/mock"
)

var _ session.NetworkEntity = (*acceptor)(nil)

type acceptor struct {
	sid        int64
	node       *Node
	gateClient clusterpb.GateClient
	session    *session.Session
	lastMid    uint64
	rpcHandler rpcHandler
	gateAddr   string
}

// 集群模式中, 工作节点的网络对象
func newAcceptor(sid int64, node *Node, gateClient clusterpb.GateClient, rpcHandler rpcHandler, gateAddr string) *acceptor {
	return &acceptor{
		sid:        sid,
		node:       node,
		gateClient: gateClient,
		rpcHandler: rpcHandler,
		gateAddr:   gateAddr,
	}
}

// Push implements the session.NetworkEntity interface
func (a *acceptor) Push(route string, v any) error {
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

// RPC implements the session.NetworkEntity interface
func (a *acceptor) RPC(route string, v any) error {
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

// LastMid implements the session.NetworkEntity interface
func (a *acceptor) LastMid() uint64 {
	return a.lastMid
}

// Response implements the session.NetworkEntity interface
func (a *acceptor) Response(v any) error {
	return a.ResponseMid(a.lastMid, v)
}

// ResponseMid implements the session.NetworkEntity interface
func (a *acceptor) ResponseMid(mid uint64, v any) error {
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

// Close 集群模式下, Worker 节点关闭会话, 通知 Gate 也关闭(主动关闭)
func (a *acceptor) Close() error {
	// TODO: buffer
	// 先删除
	s, found := a.node.delSession(a.sid)
	// 通知 gate
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

// RemoteAddr implements the session.NetworkEntity interface
func (a *acceptor) RemoteAddr() net.Addr {
	return mock.NetAddr{}
}
