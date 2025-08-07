package cluster

import (
	"context"
	"fmt"

	"github.com/lonng/nano/cluster/clusterpb"
)

var _ clusterpb.GateServer = gateService{}
var _ clusterpb.GateServer = (*gateService)(nil)

type gateService struct {
	node *Node
}

func newGateService(node *Node) gateService {
	return gateService{node: node}
}

func (g gateService) HandlePush(_ context.Context, req *clusterpb.PushMessage) (*clusterpb.GateHandleResponse, error) {
	s, found := g.node.findSession(req.SessionId)
	if !found {
		return &clusterpb.GateHandleResponse{}, fmt.Errorf("session not found: %v", req.SessionId)
	}
	return &clusterpb.GateHandleResponse{}, s.Push(req.Route, req.Data)
}

func (g gateService) HandleResponse(_ context.Context, req *clusterpb.ResponseMessage) (*clusterpb.GateHandleResponse, error) {
	s, found := g.node.findSession(req.SessionId)
	if !found {
		return &clusterpb.GateHandleResponse{}, fmt.Errorf("session not found: %v", req.SessionId)
	}
	return &clusterpb.GateHandleResponse{}, s.ResponseMID(req.Id, req.Data)
}

// CloseSession 作为 Gate 时, 处理业务节点关闭 Session 的请求
func (g gateService) CloseSession(_ context.Context, req *clusterpb.CloseSessionRequest) (*clusterpb.CloseSessionResponse, error) {
	s, found := g.node.findSession(req.SessionId)
	if found {
		s.Close() // agent 的 read 协程退出时, 会将 session 从 node 中删除并触发事件
	}
	return &clusterpb.CloseSessionResponse{}, nil
}
