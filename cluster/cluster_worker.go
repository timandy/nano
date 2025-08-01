package cluster

import (
	"context"
	"fmt"

	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
)

var _ clusterpb.WorkerServer = worker{}
var _ clusterpb.WorkerServer = (*worker)(nil)

type worker struct {
	node *Node
}

func newWorker(node *Node) worker {
	return worker{node: node}
}

func (w worker) HandleRequest(_ context.Context, req *clusterpb.RequestMessage) (*clusterpb.MemberHandleResponse, error) {
	handlerNode, found := w.node.handler.localHandlers[req.Route]
	if !found {
		return nil, fmt.Errorf("service not found in current node: %v", req.Route)
	}
	s, err := w.node.findOrCreateSession(req.SessionId, req.GateAddr)
	if err != nil {
		return nil, err
	}
	msg := &message.Message{
		Type:  message.Request,
		ID:    req.Id,
		Route: req.Route,
		Data:  req.Data,
	}
	w.node.handler.localProcess(handlerNode, req.Id, s, msg)
	return &clusterpb.MemberHandleResponse{}, nil
}

func (w worker) HandleNotify(_ context.Context, req *clusterpb.NotifyMessage) (*clusterpb.MemberHandleResponse, error) {
	handler, found := w.node.handler.localHandlers[req.Route]
	if !found {
		return nil, fmt.Errorf("service not found in current node: %v", req.Route)
	}
	s, err := w.node.findOrCreateSession(req.SessionId, req.GateAddr)
	if err != nil {
		return nil, err
	}
	msg := &message.Message{
		Type:  message.Notify,
		Route: req.Route,
		Data:  req.Data,
	}
	w.node.handler.localProcess(handler, 0, s, msg)
	return &clusterpb.MemberHandleResponse{}, nil
}

// SessionClosed 作为业务节点时, 处理 Gateway Session 已关闭的事件
func (w worker) SessionClosed(_ context.Context, req *clusterpb.SessionClosedRequest) (*clusterpb.SessionClosedResponse, error) {
	s, found := w.node.delSession(req.SessionId)
	if found {
		scheduler.Execute(func() { session.Event.FireSessionClosed(s) })
	}
	return &clusterpb.SessionClosedResponse{}, nil
}
