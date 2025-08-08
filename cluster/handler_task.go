package cluster

import (
	"github.com/lonng/nano/npi"
	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/session"
)

// 任务对象
type localHandlerTask struct {
	localHandler *LocalHandler
	handlerNode  *npi.HandlerNode
	session      *session.Session
	msg          *message.Message
	service      string
	lastMid      uint64
}

// run 运行任务
func (task *localHandlerTask) run() {
	h := task.localHandler
	handlerNode := task.handlerNode
	s := task.session
	msg := task.msg
	service := task.service
	lastMid := task.lastMid

	// 并行锁
	defer s.LockUnlock()()

	// 标记, 写响应的时候使用
	switch v := s.NetworkEntity().(type) {
	case *agent:
		v.lastMid = lastMid
	case *acceptor:
		v.lastMid = lastMid
	}
	// 获取 Context
	pool := &h.node.pool
	c := pool.Get().(*npi.Context)
	defer pool.Put(c)
	c.Reset()
	// 初始化
	c.Mid = lastMid
	c.RoutePath = msg.Route
	c.Service = service
	c.Msg = msg
	c.Session = s
	// 有路由
	if handlerNode.Len() > 0 {
		c.HandlerNode = handlerNode
		c.Next()
		return
	}
	// 无路由
	c.HandlerNode = h.allNoRoutes
	c.Next()
}
