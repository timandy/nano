package cluster

import (
	"context"
	"fmt"

	"github.com/lonng/nano/cluster/clusterpb"
)

var _ clusterpb.GateServer = gate{}
var _ clusterpb.GateServer = (*gate)(nil)

type gate struct {
	node *Node
}

func newGate(node *Node) gate {
	return gate{node: node}
}

func (g gate) HandlePush(_ context.Context, req *clusterpb.PushMessage) (*clusterpb.GateHandleResponse, error) {
	s, found := g.node.findSession(req.SessionId)
	if !found {
		return &clusterpb.GateHandleResponse{}, fmt.Errorf("session not found: %v", req.SessionId)
	}
	return &clusterpb.GateHandleResponse{}, s.Push(req.Route, req.Data)
}

func (g gate) HandleResponse(_ context.Context, req *clusterpb.ResponseMessage) (*clusterpb.GateHandleResponse, error) {
	s, found := g.node.findSession(req.SessionId)
	if !found {
		return &clusterpb.GateHandleResponse{}, fmt.Errorf("session not found: %v", req.SessionId)
	}
	return &clusterpb.GateHandleResponse{}, s.ResponseMID(req.Id, req.Data)
}

// CloseSession 作为 Gateway 时, 处理业务节点关闭 Session 的请求
func (g gate) CloseSession(_ context.Context, req *clusterpb.CloseSessionRequest) (*clusterpb.CloseSessionResponse, error) {
	s, found := g.node.delSession(req.SessionId)
	if found {
		s.Close()
	}
	return &clusterpb.CloseSessionResponse{}, nil
}
