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
	"net"
	"net/http"
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
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
	"google.golang.org/grpc"
)

// Options 引擎选项
type Options struct {
	//注册路由
	AutoScan bool //自动扫描

	//握手
	CheckOrigin        func(*http.Request) bool             //跨域检测, WebSocket 升级阶段
	HandshakeValidator func(*session.Session, []byte) error //握手阶段回调

	//工作
	Pipeline   pipeline.Pipeline     //所有输入前置函数和输出前置函数
	Components *component.Components //业务组件, 类似 Controller

	//集群
	IsMaster           bool                       //是否中心节点
	TcpAddr            string                     //非 websocket 模式需要配置, 一般是 :Port
	AdvertiseAddr      string                     //RPC 服务对外地址, 一般是 IP:Port; 子节点, 要配置这个值, 以便向 Master 注册子自身的 ServiceAddr
	ServiceAddr        string                     //RPC 服务监听地址, 一般是 IP:Port; 主子节点都要配置这个值
	RemoteServiceRoute CustomerRemoteServiceRoute //自定义节点路由规则
	UnregisterCallback UnregisterCallback         //主节点可以配置回调
	Label              string                     // 节点标签, 用于标识节点, 例如 "master", "slave-1", "slave-2" 等
}

// DefaultOptions 默认选项
func DefaultOptions() *Options {
	return &Options{
		//注册路由
		AutoScan: true,
		//握手
		CheckOrigin:        func(_ *http.Request) bool { return true },
		HandshakeValidator: nil,
		//工作
		Components: &component.Components{},
		//集群
		IsMaster: false, //默认不是主节点
	}
}

// SingleMode 是否单节点模式
func (o *Options) SingleMode() bool {
	//不是主节点 且 不是子节点
	return !o.IsMaster && o.AdvertiseAddr == ""
}

// Node 表示 nano 集群中的一个节点，该节点将包含一组服务。
// 所有服务都将注册到集群，消息将通过 grpc 转发到节点提供相应的服务。
type Node struct {
	engine   npi.Engine          //引擎
	pool     sync.Pool           //Context 池, 用于复用 Context 对象
	upgrader *websocket.Upgrader //升级器
	*Options                     //选项

	cluster   *cluster
	handler   *LocalHandler
	server    *grpc.Server
	rpcClient *rpcClient

	mu       sync.RWMutex
	sessions map[int64]*session.Session

	once          sync.Once
	keepaliveExit chan struct{}
}

// NewNode 创建新的节点
func NewNode(engine npi.Engine, opts *Options) *Node {
	return &Node{
		engine: engine,
		pool: sync.Pool{New: func() any {
			return &npi.Context{}
		}},
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     opts.CheckOrigin,
		},
		Options: opts,
	}
}

// Startup 启动节点
func (n *Node) Startup() error {
	n.sessions = map[int64]*session.Session{}
	n.cluster = newCluster(n)
	n.handler = newHandler(n)
	components := n.Components.List()
	autoScan := n.AutoScan
	if autoScan {
		for _, c := range components {
			//扫描
			err := n.handler.scan(c.Comp, c.Opts)
			if err != nil {
				return err
			}
		}
	}
	//创建缓存数据
	cache()

	//初始化节点
	if err := n.initNode(); err != nil {
		return err
	}

	//初始化组件列表
	for _, c := range components {
		c.Comp.Init()
	}
	for _, c := range components {
		c.Comp.AfterInit()
	}

	//tcp 协议直接开启监听, 如果是 websocket 模式, 则不需要开启监听, 通过 WSHandler() 依附于gin
	if n.TcpAddr != "" {
		go func() {
			n.listenAndServe()
		}()
	}

	return nil
}

// Handler 返回当前节点的本地处理器
func (n *Node) Handler() *LocalHandler {
	return n.handler
}

// initNode 初始化节点的 grpc 服务
func (n *Node) initNode() error {
	// 单节点模式
	if n.SingleMode() {
		return nil
	}

	listener, err := net.Listen("tcp", n.ServiceAddr)
	if err != nil {
		return err
	}

	//初始化 gRPC 服务器并注册服务
	n.server = grpc.NewServer()
	n.rpcClient = newRPCClient()
	n.registerServices() //注册服务

	//开始 gRPC 监听
	go func() {
		err := n.server.Serve(listener)
		if err != nil {
			log.Fatal("Start current node failed.", err)
		}
	}()

	//初始化主节点
	if n.IsMaster {
		return n.initMaster()
	}

	//初始化成员节点
	return n.initMember()
}

// initMaster 初始化主节点
func (n *Node) initMaster() error {
	member := &Member{
		isMaster: true,
		memberInfo: &clusterpb.MemberInfo{
			Label:       n.Label,
			ServiceAddr: n.ServiceAddr,
			Services:    n.handler.LocalService(),
		},
	}
	n.cluster.members = append(n.cluster.members, member)
	n.cluster.setRpcClient(n.rpcClient)
	return nil
}

// initMember 初始化成员节点
func (n *Node) initMember() error {
	pool, err := n.rpcClient.getConnPool(n.AdvertiseAddr)
	if err != nil {
		return err
	}
	client := clusterpb.NewMasterClient(pool.Get())
	request := &clusterpb.RegisterRequest{
		MemberInfo: &clusterpb.MemberInfo{
			Label:       n.Label,
			ServiceAddr: n.ServiceAddr,
			Services:    n.handler.LocalService(),
		},
	}

	//将成员地址和成员服务注册到主节点, 主节点返回其他成员的服务, 同时主节点也会将当前节点的服务通知其他成员节点
	for {
		resp, err := client.Register(context.Background(), request)
		if err == nil {
			n.handler.initRemoteService(resp.Members)
			n.cluster.initMembers(resp.Members)
			break
		}
		log.Info("Register current node to cluster failed, and will retry in %v.", env.RetryInterval.String(), err)
		time.Sleep(env.RetryInterval)
	}

	//成员节点, 异步心跳
	n.once.Do(n.keepalive)
	return nil
}

// registerServices 注册 grpc 服务
func (n *Node) registerServices() {
	clusterpb.RegisterMemberServer(n.server, n)
	if n.IsMaster {
		clusterpb.RegisterMasterServer(n.server, n.cluster)
	}
}

// Shutdown 停止应用程序注册的所有组件，倒序调用组件的关闭方法，从 master 节点取消注册
func (n *Node) Shutdown() {
	// reverse call `BeforeShutdown` hooks
	components := n.Components.List()
	length := len(components)
	for i := length - 1; i >= 0; i-- {
		components[i].Comp.BeforeShutdown()
	}

	// reverse call `Shutdown` hooks
	for i := length - 1; i >= 0; i-- {
		components[i].Comp.Shutdown()
	}
	// close sendHeartbeat
	if n.keepaliveExit != nil {
		close(n.keepaliveExit)
	}
	if !n.IsMaster && n.AdvertiseAddr != "" {
		pool, err := n.rpcClient.getConnPool(n.AdvertiseAddr)
		if err != nil {
			log.Info("Retrieve master address error.", err)
			goto EXIT
		}
		client := clusterpb.NewMasterClient(pool.Get())
		request := &clusterpb.UnregisterRequest{
			ServiceAddr: n.ServiceAddr,
		}
		_, err = client.Unregister(context.Background(), request)
		if err != nil {
			log.Info("Unregister current node failed", err)
			goto EXIT
		}
	}

EXIT:
	if n.server != nil {
		n.server.GracefulStop()
	}
}

// ServeHttp 处理 HTTP 请求, 主要用于 WebSocket 升级
func (n *Node) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := n.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Info("Upgrade failure, URI=%s", r.RequestURI, err)
		return
	}
	n.handler.handleWS(conn)
}

// listenAndServe 启动 tcp/ip 监听; 对于同一个连接(读是单独协程, 写是单独协程), 所有连接共用一个业务协程
func (n *Node) listenAndServe() {
	listener, err := net.Listen("tcp", n.TcpAddr)
	if err != nil {
		log.Fatal("Start server listen %s error.", n.TcpAddr, err)
	}

	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Info("Accept connection error.", err)
			continue
		}

		go n.handler.handle(conn)
	}
}

func (n *Node) storeSession(s *session.Session) {
	n.saveSession(s.ID(), s)
}

func (n *Node) findOrCreateSession(sid int64, gateAddr string) (*session.Session, error) {
	s, found := n.findSession(sid)
	if !found {
		conns, err := n.rpcClient.getConnPool(gateAddr)
		if err != nil {
			return nil, err
		}
		ac := newAcceptor(sid, clusterpb.NewMemberClient(conns.Get()), n.handler.remoteProcess, gateAddr)
		s = session.New(ac)
		ac.session = s
		n.saveSession(sid, s)
	}
	return s, nil
}

func (n *Node) HandleRequest(_ context.Context, req *clusterpb.RequestMessage) (*clusterpb.MemberHandleResponse, error) {
	handlerNode, found := n.handler.localHandlers[req.Route]
	if !found {
		return nil, fmt.Errorf("service not found in current node: %v", req.Route)
	}
	s, err := n.findOrCreateSession(req.SessionId, req.GateAddr)
	if err != nil {
		return nil, err
	}
	msg := &message.Message{
		Type:  message.Request,
		ID:    req.Id,
		Route: req.Route,
		Data:  req.Data,
	}
	n.handler.localProcess(handlerNode, req.Id, s, msg)
	return &clusterpb.MemberHandleResponse{}, nil
}

func (n *Node) HandleNotify(_ context.Context, req *clusterpb.NotifyMessage) (*clusterpb.MemberHandleResponse, error) {
	handler, found := n.handler.localHandlers[req.Route]
	if !found {
		return nil, fmt.Errorf("service not found in current node: %v", req.Route)
	}
	s, err := n.findOrCreateSession(req.SessionId, req.GateAddr)
	if err != nil {
		return nil, err
	}
	msg := &message.Message{
		Type:  message.Notify,
		Route: req.Route,
		Data:  req.Data,
	}
	n.handler.localProcess(handler, 0, s, msg)
	return &clusterpb.MemberHandleResponse{}, nil
}

func (n *Node) HandlePush(_ context.Context, req *clusterpb.PushMessage) (*clusterpb.MemberHandleResponse, error) {
	s, found := n.findSession(req.SessionId)
	if !found {
		return &clusterpb.MemberHandleResponse{}, fmt.Errorf("session not found: %v", req.SessionId)
	}
	return &clusterpb.MemberHandleResponse{}, s.Push(req.Route, req.Data)
}

func (n *Node) HandleResponse(_ context.Context, req *clusterpb.ResponseMessage) (*clusterpb.MemberHandleResponse, error) {
	s, found := n.findSession(req.SessionId)
	if !found {
		return &clusterpb.MemberHandleResponse{}, fmt.Errorf("session not found: %v", req.SessionId)
	}
	return &clusterpb.MemberHandleResponse{}, s.ResponseMID(req.Id, req.Data)
}

func (n *Node) NewMember(_ context.Context, req *clusterpb.NewMemberRequest) (*clusterpb.NewMemberResponse, error) {
	n.handler.addRemoteService(req.MemberInfo)
	n.cluster.addMember(req.MemberInfo)
	return &clusterpb.NewMemberResponse{}, nil
}

func (n *Node) DelMember(_ context.Context, req *clusterpb.DelMemberRequest) (*clusterpb.DelMemberResponse, error) {
	log.Info("DelMember member", req.String())
	n.handler.delMember(req.ServiceAddr)
	n.cluster.delMember(req.ServiceAddr)
	return &clusterpb.DelMemberResponse{}, nil
}

// SessionClosed 作为业务节点时, 处理 Gateway Session 已关闭的事件
func (n *Node) SessionClosed(_ context.Context, req *clusterpb.SessionClosedRequest) (*clusterpb.SessionClosedResponse, error) {
	s, found := n.delSession(req.SessionId)
	if found {
		scheduler.PushTask(func() { session.Lifetime.Close(s) })
	}
	return &clusterpb.SessionClosedResponse{}, nil
}

// CloseSession 作为 Gateway 时, 处理业务节点关闭 Session 的请求
func (n *Node) CloseSession(_ context.Context, req *clusterpb.CloseSessionRequest) (*clusterpb.CloseSessionResponse, error) {
	s, found := n.delSession(req.SessionId)
	if found {
		s.Close()
	}
	return &clusterpb.CloseSessionResponse{}, nil
}

// findSession 查找 Session, 如果不存在则返回 nil, false
func (n *Node) findSession(sid int64) (*session.Session, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	s, found := n.sessions[sid]
	return s, found
}

// delSession 删除 Session, 如果删除前存在则返回 Session 和 true, 否则返回 nil 和 false
func (n *Node) delSession(sid int64) (*session.Session, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	s, found := n.sessions[sid]
	delete(n.sessions, sid)
	return s, found
}

// saveSession 保存 Session, 如果存在则覆盖
func (n *Node) saveSession(sid int64, s *session.Session) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sessions[sid] = s
}

// keepalive 作为 Gateway 或 业务节点, 启动定时发送心跳协程
func (n *Node) keepalive() {
	if n.keepaliveExit == nil {
		n.keepaliveExit = make(chan struct{})
	}
	if n.AdvertiseAddr == "" || n.IsMaster {
		return
	}
	go func() {
		ticker := time.NewTicker(env.HeartbeatInterval)
		for {
			select {
			case <-ticker.C:
				n.heartbeat()
			case <-n.keepaliveExit:
				log.Info("Exit member node heartbeat ")
				ticker.Stop()
				return
			}
		}
	}()
}

// heartbeat 作为 Gateway 或 业务节点, 向 Master 发送一次心跳请求
func (n *Node) heartbeat() {
	pool, err := n.rpcClient.getConnPool(n.AdvertiseAddr)
	if err != nil {
		log.Info("rpcClient master conn", err)
		return
	}
	masterCli := clusterpb.NewMasterClient(pool.Get())
	request := clusterpb.HeartbeatRequest{
		MemberInfo: &clusterpb.MemberInfo{
			Label:       n.Label,
			ServiceAddr: n.ServiceAddr,
			Services:    n.handler.LocalService(),
		},
	}
	if _, err = masterCli.Heartbeat(context.Background(), &request); err != nil {
		log.Info("Member send heartbeat error.", err)
	}
}
