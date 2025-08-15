package cluster

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sort"
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
	// create a client agent and startup write goroutine
	agt := newAgent(conn, h.pipeline, h.remoteProcess)
	s := agt.session
	h.node.saveSession(s.ID(), s)
	defer func() {
		h.node.delSession(s.ID())                                //防止 session 泄露
		s.Execute(func() { session.Event.FireSessionClosed(s) }) //异步执行关闭事件
	}()

	// startup write goroutine
	go agt.write()

	if env.Debug {
		log.Info("New session established: %s", agt.String())
	}

	// guarantee agent related resource be destroyed
	defer func() {
		request := &clusterpb.SessionClosedRequest{SessionId: s.ID()}

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

		_ = agt.Close()
		if env.Debug {
			log.Info("Session read goroutine exit, SessionID=%d, UID=%d", s.ID(), s.UID())
		}
	}()

	// read loop
	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if agt.status() == statusClosed {
				return
			}
			log.Error("Read message error, session will be closed immediately.", err)
			return
		}

		// TODO(warning): decoder use slice for performance, packet data should be copy before next Decode
		packets, err := agt.decoder.Decode(buf[:n])
		if err != nil {
			log.Error("Decode packet error.", err)

			// 处理已经解码的包并返回
			for _, p := range packets {
				if err = h.processPacket(agt, p); err != nil {
					log.Error("Process packet error.", err)
					return
				}
			}
			return
		}

		// 处理所有包
		for _, p := range packets {
			if err = h.processPacket(agt, p); err != nil {
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
	default:
		return errors.New("invalid packet type")
	}

	agent.lastAt.Store(time.Now().Unix())
	return nil
}

// processMessage 处理消息, 分发到本地或远程
func (h *LocalHandler) processMessage(agent *agent, msg *message.Message) {
	if handlerNode, found := h.localHandlers[msg.Route]; found {
		h.localProcess(agent.session, msg, handlerNode)
		return
	}
	h.remoteProcess(agent.session, msg, false)
}

// remoteProcess 处理远程消息
func (h *LocalHandler) remoteProcess(session *session.Session, msg *message.Message, noCopy bool) {
	service := msg.Service()
	members := h.findMembers(service)
	if len(members) == 0 {
		// 没有服务节点, 转本地处理 404
		h.localProcess(session, msg, nil)
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
				if env.Debug {
					log.Info("customize remoteServiceRoute handler: %s is not found", msg.Route)
				}
				// 有服务节点, 但自定义路由没匹配到, 转本地处理 404
				h.localProcess(session, msg, nil)
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
func (h *LocalHandler) localProcess(session *session.Session, msg *message.Message, handlerNode *npi.HandlerNode) {
	// 计算响应 ID
	lastMid, err := msg.ResponseID()
	if err != nil {
		log.Info(err.Error() + ":" + msg.Type.String())
		return
	}

	// 计算服务名
	service := msg.Service()

	// 执行管道任务
	if pipe := h.pipeline; pipe != nil {
		err = pipe.Inbound().Process(session, msg)
		if err != nil {
			log.Error("Pipeline process failed.", err)
			return
		}
	}

	// 记录日志
	if env.Debug {
		log.Info("UID=%d, Message={%s}, Data=%v", session.UID(), msg.String(), msg.Data)
	}

	// session 任务对象
	task := &localHandlerTask{
		localHandler: h,
		handlerNode:  handlerNode,
		session:      session,
		msg:          msg,
		service:      service,
		lastMid:      lastMid,
	}

	// component 级别的执行器
	executorFactory := func() schedulerapi.Executor {
		if s, found := h.localServices[service]; found {
			return s.Executor
		}
		return nil
	}

	// 按优先级调度任务
	session.Execute(task.run, executorFactory)
}

// findMembers 远程处理时, 查找服务对应的成员
func (h *LocalHandler) findMembers(service string) []*clusterpb.MemberInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.remoteServices[service]
}
