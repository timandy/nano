// Copyright (c) nano Authors. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package cluster

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/scheduler"
)

var _ clusterpb.MasterServer = (*clusterService)(nil)

// cluster 成员节点管理器.
// 当前节点作为 master 时, 该结构可作为 rpc server.
// 作为 member 时, 该结构可作为存储集群内节点列表的容器.
type clusterService struct {
	// If cluster is not large enough, use slice is OK
	node *Node

	mu      sync.RWMutex
	members []*Member
}

// 构造函数
func newClusterService(node *Node) *clusterService {
	c := &clusterService{node: node}
	if node.opts.NodeType.IsMaster() {
		c.startHeartbeatCheckTimer()
	}
	return c
}

// Register implements the MasterServer gRPC service
func (c *clusterService) Register(_ context.Context, req *clusterpb.RegisterRequest) (*clusterpb.RegisterResponse, error) {
	if req.MemberInfo == nil {
		return nil, ErrInvalidRegisterReq
	}
	// 先删除一次, 防止重复注册
	c.delMember(req.MemberInfo.ServiceAddr)

	// Notify registered node to update remote services
	resp := &clusterpb.RegisterResponse{}
	newMember := &clusterpb.NewMemberRequest{MemberInfo: req.MemberInfo}
	for _, m := range c.members {
		resp.Members = append(resp.Members, m.memberInfo)
		if m.isMaster {
			continue
		}
		pool, err := c.node.rpcClient.getConnPool(m.memberInfo.ServiceAddr)
		if err != nil {
			return nil, err
		}
		client := clusterpb.NewMemberClient(pool.Get())
		_, err = client.NewMember(context.Background(), newMember)
		if err != nil {
			return nil, err
		}
	}

	log.Info("New peer register to cluster", req.MemberInfo.ServiceAddr)

	// Register services to current node
	c.node.handler.addRemoteService(req.MemberInfo)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.members = append(c.members, &Member{isMaster: false, memberInfo: req.MemberInfo, lastHeartbeatAt: time.Now()})
	return resp, nil
}

// Unregister implements the MasterServer gRPC service
func (c *clusterService) Unregister(_ context.Context, req *clusterpb.UnregisterRequest) (*clusterpb.UnregisterResponse, error) {
	if req.ServiceAddr == "" {
		return nil, ErrInvalidRegisterReq
	}

	var index = -1
	resp := &clusterpb.UnregisterResponse{}
	for i, m := range c.members {
		if m.memberInfo.ServiceAddr == req.ServiceAddr {
			index = i
			break
		}
	}
	if index < 0 {
		return nil, fmt.Errorf("address %s has not registered", req.ServiceAddr)
	}

	// Notify registered node to update remote services
	delMember := &clusterpb.DelMemberRequest{ServiceAddr: req.ServiceAddr}
	for i, m := range c.members {
		if i == index {
			// this node is down.
			continue
		}

		if m.MemberInfo().ServiceAddr == c.node.opts.ServiceAddr {
			continue
		}
		pool, err := c.node.rpcClient.getConnPool(m.memberInfo.ServiceAddr)
		if err != nil {
			return nil, err
		}
		client := clusterpb.NewMemberClient(pool.Get())
		_, err = client.DelMember(context.Background(), delMember)
		if err != nil {
			return nil, err
		}
	}

	log.Info("Exists peer unregister to cluster", req.ServiceAddr)

	if c.node.opts.UnregisterCallback != nil {
		c.node.opts.UnregisterCallback(*c.members[index])
	}

	// Register services to current node
	c.node.handler.delMember(req.ServiceAddr)
	c.mu.Lock()
	defer c.mu.Unlock()
	if index >= len(c.members)-1 {
		c.members = c.members[:index]
	} else {
		c.members = append(c.members[:index], c.members[index+1:]...)
	}

	return resp, nil
}

func (c *clusterService) Heartbeat(_ context.Context, req *clusterpb.HeartbeatRequest) (*clusterpb.HeartbeatResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	isHit := false
	for i, m := range c.members {
		if m.MemberInfo().GetServiceAddr() == req.GetMemberInfo().GetServiceAddr() {
			c.members[i].lastHeartbeatAt = time.Now()
			isHit = true
		}
	}
	if !isHit {
		// master local not binding this node, other members do not need to be notified, because this node registered.
		// maybe the master process reload
		m := &Member{
			isMaster:        false,
			memberInfo:      req.GetMemberInfo(),
			lastHeartbeatAt: time.Now(),
		}
		c.members = append(c.members, m)
		c.node.handler.addRemoteService(req.MemberInfo)
		log.Info("Heartbeat peer register to cluster", req.MemberInfo.ServiceAddr)
	}
	return &clusterpb.HeartbeatResponse{}, nil
}

func (c *clusterService) initMembers(members []*clusterpb.MemberInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, info := range members {
		c.members = append(c.members, &Member{
			memberInfo: info,
		})
	}
}

func (c *clusterService) addMember(info *clusterpb.MemberInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var found bool
	for _, member := range c.members {
		if member.memberInfo.ServiceAddr == info.ServiceAddr {
			member.memberInfo = info
			found = true
			break
		}
	}
	if !found {
		c.members = append(c.members, &Member{
			memberInfo: info,
		})
	}
}

func (c *clusterService) delMember(addr string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var index = -1
	for i, member := range c.members {
		if member.memberInfo.ServiceAddr == addr {
			index = i
			break
		}
	}
	if index != -1 {
		if index >= len(c.members)-1 {
			c.members = c.members[:index]
		} else {
			c.members = append(c.members[:index], c.members[index+1:]...)
		}
	}
}

func (c *clusterService) startHeartbeatCheckTimer() {
	if !c.node.opts.NodeType.IsMaster() {
		return
	}

	go func() {
		ticker := scheduler.Heartbeat.NewTicker(env.HeartbeatInterval)
		for {
			select {
			case <-ticker.C:
				c.checkHeartbeat()
			}
		}
	}()
}

func (c *clusterService) checkHeartbeat() {
	unregisterMembers := make([]*Member, 0)
	// check heartbeat time
	for _, m := range c.members {
		if time.Now().Sub(m.lastHeartbeatAt) > 4*env.HeartbeatInterval && !m.isMaster {
			unregisterMembers = append(unregisterMembers, m)
		}
	}

	for _, m := range unregisterMembers {
		req := &clusterpb.UnregisterRequest{
			ServiceAddr: m.MemberInfo().ServiceAddr,
		}
		if _, err := c.Unregister(context.Background(), req); err != nil {
			log.Error("Heartbeat unregister error.", err)
		}
	}
}

func (c *clusterService) remoteAddrs() []string {
	var addrs []string
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, m := range c.members {
		addrs = append(addrs, m.memberInfo.ServiceAddr)
	}
	return addrs
}
