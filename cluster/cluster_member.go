package cluster

import (
	"context"

	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
)

var _ clusterpb.MemberServer = memberService{}
var _ clusterpb.MemberServer = (*memberService)(nil)

type memberService struct {
	node *Node
}

func newMemberService(node *Node) memberService {
	return memberService{node: node}
}

func (m memberService) NewMember(_ context.Context, req *clusterpb.NewMemberRequest) (*clusterpb.NewMemberResponse, error) {
	if env.Debug {
		log.Info("NewMember member %v", req.String())
	}
	m.node.handler.addRemoteService(req.MemberInfo)
	m.node.cluster.addMember(req.MemberInfo)
	return &clusterpb.NewMemberResponse{}, nil
}

func (m memberService) DelMember(_ context.Context, req *clusterpb.DelMemberRequest) (*clusterpb.DelMemberResponse, error) {
	if env.Debug {
		log.Info("DelMember member %v", req.String())
	}
	m.node.handler.delMember(req.ServiceAddr)
	m.node.cluster.delMember(req.ServiceAddr)
	return &clusterpb.DelMemberResponse{}, nil
}
