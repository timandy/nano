package cluster

import (
	"context"
	"fmt"

	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/session"
)

var _ clusterpb.WorkerServer = workerService{}
var _ clusterpb.WorkerServer = (*workerService)(nil)

type workerService struct {
	node *Node
}

func newWorkerService(node *Node) workerService {
	return workerService{node: node}
}

func (w workerService) HandleRequest(_ context.Context, req *clusterpb.RequestMessage) (*clusterpb.MemberHandleResponse, error) {
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
	w.node.handler.localProcess(s, msg, handlerNode)
	return &clusterpb.MemberHandleResponse{}, nil
}

func (w workerService) HandleNotify(_ context.Context, req *clusterpb.NotifyMessage) (*clusterpb.MemberHandleResponse, error) {
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
	w.node.handler.localProcess(s, msg, handler)
	return &clusterpb.MemberHandleResponse{}, nil
}

// SessionClosed 集群模式中, 作为 Worker 时, 处理 Gate 发来的 Session 已关闭的事件(被动关闭)
func (w workerService) SessionClosed(_ context.Context, req *clusterpb.SessionClosedRequest) (*clusterpb.SessionClosedResponse, error) {
	// Worker 主动关闭时, 通知 Gate, 然后 Gate 会再次通知 Worker, 但此时 session 已经被删除 found==false, 不会触发事件(主动删除的时候已经触发了)
	s, found := w.node.delSession(req.SessionId)
	if found {
		s.Execute(func() { session.Event.FireSessionClosed(s) }) //异步执行关闭事件
	}
	// 注意不要调用 s.Close() 会造成再次通知 Gate
	return &clusterpb.SessionClosedResponse{}, nil
}
