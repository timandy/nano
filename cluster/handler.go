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
	"math/rand"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/npi"
	"github.com/lonng/nano/pipeline"
	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/protocal/packet"
	"github.com/lonng/nano/scheduler/schedulerapi"
	"github.com/lonng/nano/session"
)

type rpcHandler func(session *session.Session, msg *message.Message, noCopy bool)

// CustomerRemoteServiceRoute customer remote service route
type CustomerRemoteServiceRoute func(service string, session *session.Session, members []*clusterpb.MemberInfo) *clusterpb.MemberInfo

// UnregisterCallback 取消注册的回调
type UnregisterCallback func(Member)

type LocalHandler struct {
	localServices map[string]*component.Service // all registered service
	localHandlers npi.HandlerTrees              // all handler methods
	allNoRoutes   *npi.HandlerNode              // no routes handler methods

	mu             sync.RWMutex
	remoteServices map[string][]*clusterpb.MemberInfo

	node     *Node
	pipeline pipeline.Pipeline
	opts     *Options
}

func newHandler(node *Node) *LocalHandler {
	engine := node.engine
	return &LocalHandler{
		localServices:  make(map[string]*component.Service),
		localHandlers:  getTrees(engine),
		allNoRoutes:    getAllNoRoutes(engine),
		remoteServices: map[string][]*clusterpb.MemberInfo{},
		node:           node,
		pipeline:       node.opts.Pipeline,
		opts:           node.opts,
	}
}

func getTrees(engine npi.Engine) npi.HandlerTrees {
	if engine == nil {
		return npi.HandlerTrees{}
	}
	trees := engine.Trees()
	if trees == nil {
		return npi.HandlerTrees{}
	}
	return trees
}

func getAllNoRoutes(engine npi.Engine) *npi.HandlerNode {
	if engine == nil {
		return npi.NewHandlerNode()
	}
	return npi.NewHandlerNode(engine.AllNoRoutes()...)
}

func (h *LocalHandler) scan(comp component.Component, opts []component.Option, scan bool) error {
	s := component.NewService(comp, opts)

	// 注册服务
	if _, ok := h.localServices[s.Name]; ok {
		return fmt.Errorf("handler: service already defined: %s", s.Name)
	}
	h.localServices[s.Name] = s

	// 禁用了自动扫描, 只注册服务, 不扫描方法
	if !scan {
		return nil
	}

	// 扫描方法
	if err := s.ExtractHandler(); err != nil {
		return err
	}

	// 注册到路由树
	for name, handler := range s.Handlers {
		route := fmt.Sprintf("%s.%s", s.Name, name)
		log.Info("Register local handler", route)
		h.localHandlers.Append(route, npi.NewHandlerNodeWithName(name, handler.Call))
	}
	return nil
}

func (h *LocalHandler) initRemoteService(members []*clusterpb.MemberInfo) {
	for _, m := range members {
		h.addRemoteService(m)
	}
}

func (h *LocalHandler) addRemoteService(member *clusterpb.MemberInfo) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, s := range member.Services {
		log.Info("Register remote service", s)
		h.remoteServices[s] = append(h.remoteServices[s], member)
	}
}

func (h *LocalHandler) delMember(addr string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for name, members := range h.remoteServices {
		for i, maddr := range members {
			if addr == maddr.ServiceAddr {
				if i >= len(members)-1 {
					members = members[:i]
				} else {
					members = append(members[:i], members[i+1:]...)
				}
			}
		}
		if len(members) == 0 {
			delete(h.remoteServices, name)
		} else {
			h.remoteServices[name] = members
		}
	}
}

func (h *LocalHandler) LocalService() []string {
	var result []string
	for service := range h.localServices {
		result = append(result, service)
	}
	sort.Strings(result)
	return result
}

func (h *LocalHandler) RemoteService() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []string
	for service := range h.remoteServices {
		result = append(result, service)
	}
	sort.Strings(result)
	return result
}

// handleWS 对于同一个连接(读是单独协程, 写是单独协程), 所有连接共用一个业务协程
func (h *LocalHandler) handleWS(conn *websocket.Conn) {
	c, err := newWSConn(conn)
	if err != nil {
		log.Error("Read from websocket connection error.", err)
		return
	}
	h.handle(c) //使用 http 的协程, 以便共享 ThreadLocal
}

// handle 循环读取数据包, 一个连接开启一个单独的写线程
func (h *LocalHandler) handle(conn net.Conn) {
	// create a client agent and startup write gorontine
	agent := newAgent(conn, h.pipeline, h.remoteProcess)
	h.node.storeSession(agent.session)

	// startup write goroutine
	go agent.write()

	if env.Debug {
		log.Info("New session established: %s", agent.String())
	}

	// guarantee agent related resource be destroyed
	defer func() {
		request := &clusterpb.SessionClosedRequest{
			SessionId: agent.session.ID(),
		}

		members := h.node.cluster.remoteAddrs()
		for _, remote := range members {
			log.Info("Notify remote server", remote)
			pool, err := h.node.rpcClient.getConnPool(remote)
			if err != nil {
				log.Error("Cannot retrieve connection pool for %s.", remote, err)
				continue
			}
			client := clusterpb.NewWorkerClient(pool.Get())
			_, err = client.SessionClosed(context.Background(), request)
			if err != nil {
				log.Error("Cannot closed session in remote address %v.", remote, err)
				continue
			}
			if env.Debug {
				log.Info("Notify remote server success", remote)
			}
		}

		_ = agent.Close()
		if env.Debug {
			log.Info("Session read goroutine exit, SessionID=%d, UID=%d", agent.session.ID(), agent.session.UID())
		}
	}()

	// read loop
	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Error("Read message error, session will be closed immediately.", err)
			return
		}

		// TODO(warning): decoder use slice for performance, packet data should be copy before next Decode
		packets, err := agent.decoder.Decode(buf[:n])
		if err != nil {
			log.Error("Decode packet error.", err)

			// 处理已经解码的包并返回
			for _, p := range packets {
				if err := h.processPacket(agent, p); err != nil {
					log.Error("Process packet error.", err)
					return
				}
			}
			return
		}

		// 处理所有包
		for _, p := range packets {
			if err := h.processPacket(agent, p); err != nil {
				log.Error("Process packet error.", err)
				return
			}
		}
	}
}

// processPacket 处理接收到的包, 转换成消息
func (h *LocalHandler) processPacket(agent *agent, p *packet.Packet) error {
	switch p.Type {
	case packet.Handshake:
		validator := h.opts.HandshakeValidator
		if validator != nil {
			if err := validator(agent.session, p.Data); err != nil {
				return err
			}
		}

		if _, err := agent.conn.Write(getHsd()); err != nil {
			return err
		}

		agent.setStatus(statusHandshake)
		if env.Debug {
			log.Info("Session handshake Id=%d, Remote=%s", agent.session.ID(), agent.conn.RemoteAddr())
		}

	case packet.HandshakeAck:
		agent.setStatus(statusWorking)
		if env.Debug {
			log.Info("Receive handshake ACK Id=%d, Remote=%s", agent.session.ID(), agent.conn.RemoteAddr())
		}

	case packet.Data:
		if agent.status() < statusWorking {
			return fmt.Errorf("receive data on socket which not yet ACK, session will be closed immediately, remote=%s", agent.conn.RemoteAddr().String())
		}

		msg, err := message.Decode(p.Data)
		if err != nil {
			return err
		}
		h.processMessage(agent, msg)

	case packet.Heartbeat:
		// expected
	}

	agent.lastAt = time.Now().Unix()
	return nil
}

// processMessage 处理消息, 分发到本地或远程
func (h *LocalHandler) processMessage(agent *agent, msg *message.Message) {
	var lastMid uint64
	switch msg.Type {
	case message.Request:
		lastMid = msg.ID
	case message.Notify:
		lastMid = 0
	default:
		log.Info("Invalid message type: " + msg.Type.String())
		return
	}
	handlerNode, found := h.localHandlers[msg.Route]
	if !found {
		h.remoteProcess(agent.session, msg, false)
	} else {
		h.localProcess(handlerNode, lastMid, agent.session, msg)
	}
}

// remoteProcess 处理远程消息
func (h *LocalHandler) remoteProcess(session *session.Session, msg *message.Message, noCopy bool) {
	index := strings.LastIndex(msg.Route, ".")
	if index < 0 {
		log.Info("nano/handler: invalid route %s", msg.Route)
		return
	}

	service := msg.Route[:index]
	members := h.findMembers(service)
	if len(members) == 0 {
		log.Info("nano/handler: %s not found(forgot registered?)", msg.Route)
		return
	}

	// Select a remote service address
	// 1. if exist customer remote service route ,use it, otherwise use default strategy
	// 2. Use the service address directly if the router contains binding item
	// 3. Select a remote service address randomly and bind to router
	var remoteAddr string
	if h.node.opts.RemoteServiceRoute != nil {
		if addr, found := session.Router().Find(service); found {
			remoteAddr = addr
		} else {
			member := h.node.opts.RemoteServiceRoute(service, session, members)
			if member == nil {
				log.Info("customize remoteServiceRoute handler: %s is not found", msg.Route)
				return
			}
			remoteAddr = member.ServiceAddr
			session.Router().Bind(service, remoteAddr)
		}
	} else {
		if addr, found := session.Router().Find(service); found {
			remoteAddr = addr
		} else {
			remoteAddr = members[rand.Intn(len(members))].ServiceAddr
			session.Router().Bind(service, remoteAddr)
		}
	}
	pool, err := h.node.rpcClient.getConnPool(remoteAddr)
	if err != nil {
		log.Error("Get client conn pool error.", err)
		return
	}
	var data = msg.Data
	if !noCopy && len(msg.Data) > 0 {
		data = make([]byte, len(msg.Data))
		copy(data, msg.Data)
	}

	// Retrieve gate address and session id
	gateAddr := h.node.opts.ServiceAddr
	sessionId := session.ID()
	switch v := session.NetworkEntity().(type) {
	case *acceptor:
		gateAddr = v.gateAddr
		sessionId = v.sid
	}

	client := clusterpb.NewWorkerClient(pool.Get())
	switch msg.Type {
	case message.Request:
		request := &clusterpb.RequestMessage{
			GateAddr:  gateAddr,
			SessionId: sessionId,
			Id:        msg.ID,
			Route:     msg.Route,
			Data:      data,
		}
		_, err = client.HandleRequest(context.Background(), request)
	case message.Notify:
		request := &clusterpb.NotifyMessage{
			GateAddr:  gateAddr,
			SessionId: sessionId,
			Route:     msg.Route,
			Data:      data,
		}
		_, err = client.HandleNotify(context.Background(), request)
	}
	if err != nil {
		log.Error("Process remote message (%d:%s) error.", msg.ID, msg.Route, err)
	}
}

// localProcess 本地处理
func (h *LocalHandler) localProcess(handlerNode *npi.HandlerNode, lastMid uint64, session *session.Session, msg *message.Message) {
	if pipe := h.pipeline; pipe != nil {
		err := pipe.Inbound().Process(session, msg)
		if err != nil {
			log.Error("Pipeline process failed.", err)
			return
		}
	}
	index := strings.LastIndex(msg.Route, ".")
	if index < 0 {
		log.Info("nano/handler: invalid route %s", msg.Route)
		return
	}
	service := msg.Route[:index]

	if env.Debug {
		log.Info("UID=%d, Message={%s}, Data=%v", session.UID(), msg.String(), msg.Data)
	}

	task := func() {
		//标记, 写响应的时候使用
		switch v := session.NetworkEntity().(type) {
		case *agent:
			v.lastMid = lastMid
		case *acceptor:
			v.lastMid = lastMid
		}
		//获取 Context
		pool := &h.node.pool
		c := pool.Get().(*npi.Context)
		defer pool.Put(c)
		c.Reset()
		//初始化
		c.Mid = lastMid
		c.RoutePath = msg.Route
		c.Service = service
		c.Msg = msg
		c.Session = session
		//有路由
		if handlerNode.Len() > 0 {
			c.HandlerNode = handlerNode
			c.Next()
			return
		}
		//无路由
		c.HandlerNode = h.allNoRoutes
		c.Next()
	}

	// component 级别的执行器
	executorFactory := func() schedulerapi.Executor {
		if s, found := h.localServices[service]; found {
			return s.Executor
		}
		return nil
	}

	// 按优先级调度任务
	session.Execute(task, executorFactory)
}

// findMembers 远程处理时, 查找服务对应的成员
func (h *LocalHandler) findMembers(service string) []*clusterpb.MemberInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.remoteServices[service]
}
