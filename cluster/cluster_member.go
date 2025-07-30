package cluster

import (
	"context"

	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
)

var _ clusterpb.MemberServer = member{}
var _ clusterpb.MemberServer = (*member)(nil)

type member struct {
	node *Node
}

func newMember(node *Node) member {
	return member{node: node}
}

func (m member) NewMember(_ context.Context, req *clusterpb.NewMemberRequest) (*clusterpb.NewMemberResponse, error) {
	if env.Debug {
		log.Info("NewMember member", req.String())
	}
	m.node.handler.addRemoteService(req.MemberInfo)
	m.node.cluster.addMember(req.MemberInfo)
	return &clusterpb.NewMemberResponse{}, nil
}

func (m member) DelMember(_ context.Context, req *clusterpb.DelMemberRequest) (*clusterpb.DelMemberResponse, error) {
	if env.Debug {
		log.Info("DelMember member", req.String())
	}
	m.node.handler.delMember(req.ServiceAddr)
	m.node.cluster.delMember(req.ServiceAddr)
	return &clusterpb.DelMemberResponse{}, nil
}
