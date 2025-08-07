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
	"net"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/internal/env"
	nhttp "github.com/lonng/nano/internal/http"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/npi"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
	"google.golang.org/grpc"
)

// Node 表示 nano 集群中的一个节点，该节点将包含一组服务。
// 所有服务都将注册到集群，消息将通过 grpc 转发到节点提供相应的服务。
type Node struct {
	engine   npi.Engine          //引擎
	pool     sync.Pool           //Context 池, 用于复用 Context 对象
	upgrader *websocket.Upgrader //升级器
	opts     *Options            //选项

	cluster   *clusterService
	handler   *LocalHandler
	rpcServer *grpc.Server
	rpcClient *rpcClient

	//会话
	mu       sync.RWMutex
	sessions map[int64]*session.Session

	//优雅停机 tcp
	inShutdown    atomic.Bool // true when server is in shutdown
	listeners     map[*net.Listener]struct{}
	listenersMu   sync.Mutex
	listenerGroup sync.WaitGroup

	//优雅停机 http
	httpServers     map[*nhttp.Server]struct{}
	httpServersMu   sync.Mutex
	httpServerGroup sync.WaitGroup

	//心跳
	once          sync.Once
	keepaliveExit chan struct{}
}

// NewNode 创建新的节点
func NewNode(engine npi.Engine, opts *Options) *Node {
	//创建实例
	n := &Node{
		engine: engine,
		pool: sync.Pool{New: func() any {
			return &npi.Context{}
		}},
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     opts.CheckOrigin,
		},
		opts:     opts,
		sessions: map[int64]*session.Session{},
	}

	//创建处理器
	n.cluster = newClusterService(n) //master 和 member 都需要初始化该实例, 用来管理远程节点
	n.handler = newHandler(n)

	//注册服务, 扫描方法
	components := n.opts.Components.List()
	scan := n.opts.AutoScan
	for _, c := range components {
		err := n.handler.scan(c.Comp, c.Opts, scan)
		if err != nil {
			log.Fatal("Scan component %s failed.", reflect.TypeOf(c.Comp).String(), err)
		}
	}

	//创建缓存数据
	cache()

	//初始化组件列表
	for _, c := range components {
		c.Comp.Init()
	}
	for _, c := range components {
		c.Comp.AfterInit()
	}

	//启动 gRPC 服务
	n.startGrpc()

	return n
}

// startGrpc 初始化节点的 grpc 服务
func (n *Node) startGrpc() {
	// 单节点模式
	nodeType := n.opts.NodeType
	if nodeType.IsSingle() {
		return
	}

	// 开启 gRPC 端口
	listener, err := net.Listen("tcp", n.opts.ServiceAddr)
	if err != nil {
		log.Fatal("Start listen %v failed.", n.opts.ServiceAddr, err)
	}

	// 初始化 gRPC 服务器并注册服务
	n.rpcServer = grpc.NewServer()
	n.rpcClient = newRPCClient()
	n.registerServices() //注册服务

	// 开始 gRPC 服务
	go func() {
		err = n.rpcServer.Serve(listener)
		if err != nil {
			log.Fatal("Start current node failed.", err)
		}
	}()

	//初始化主节点
	if nodeType.IsMaster() {
		n.initMaster()
		return
	}

	//初始化成员节点
	n.initMember()
}

// initMaster 初始化主节点
func (n *Node) initMaster() {
	member := &Member{
		isMaster: true,
		memberInfo: &clusterpb.MemberInfo{
			Label:       n.opts.Label,
			ServiceAddr: n.opts.ServiceAddr,
			Services:    n.handler.LocalService(),
		},
	}
	n.cluster.members = append(n.cluster.members, member)
}

// initMember 初始化成员节点
func (n *Node) initMember() {
	pool, err := n.rpcClient.getConnPool(n.opts.AdvertiseAddr)
	if err != nil {
		log.Fatal("Get conn pool of %s failed.", n.opts.AdvertiseAddr, err)
	}
	client := clusterpb.NewMasterClient(pool.Get())
	request := &clusterpb.RegisterRequest{
		MemberInfo: &clusterpb.MemberInfo{
			Label:       n.opts.Label,
			ServiceAddr: n.opts.ServiceAddr,
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
		log.Error("Register current node to cluster failed, and will retry in %v.", env.RetryInterval.String(), err)
		time.Sleep(env.RetryInterval)
	}

	//成员节点, 异步心跳
	n.once.Do(n.keepalive)
}

// registerServices 注册 grpc 服务
func (n *Node) registerServices() {
	nodeType := n.opts.NodeType
	if nodeType.IsMaster() {
		clusterpb.RegisterMasterServer(n.rpcServer, n.cluster)
	}
	if nodeType.IsMember() {
		clusterpb.RegisterMemberServer(n.rpcServer, newMemberService(n))
	}
	if nodeType.IsGate() {
		clusterpb.RegisterGateServer(n.rpcServer, newGateService(n))
	}
	if nodeType.IsWorker() {
		clusterpb.RegisterWorkerServer(n.rpcServer, newWorkerService(n))
	}
}

// Handler 返回当前节点的本地处理器
func (n *Node) Handler() *LocalHandler {
	return n.handler
}

// ServeHttp 处理 HTTP 请求, 主要用于 WebSocket 升级
func (n *Node) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := n.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Upgrade failure, URI=%s", r.RequestURI, err)
		return
	}
	n.handler.handleWS(conn)
}

// findOrCreateSession 查找或创建 Session
func (n *Node) findOrCreateSession(sid int64, gateAddr string) (*session.Session, error) {
	s, found := n.findSession(sid)
	if !found {
		conns, err := n.rpcClient.getConnPool(gateAddr)
		if err != nil {
			return nil, err
		}
		ac := newAcceptor(sid, n, clusterpb.NewGateClient(conns.Get()), n.handler.remoteProcess, gateAddr)
		s = session.New(ac)
		ac.session = s
		n.saveSession(sid, s)
	}
	return s, nil
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

// keepalive 作为 Member 节点, 启动定时发送心跳协程
func (n *Node) keepalive() {
	if n.keepaliveExit == nil {
		n.keepaliveExit = make(chan struct{})
	}
	if n.opts.AdvertiseAddr == "" || n.opts.NodeType.IsMaster() {
		return
	}
	go n.startHeartbeatTimer()
}

// startHeartbeatTimer 启动心跳定时器, 每隔一段时间向 Master 发送一次心跳请求
func (n *Node) startHeartbeatTimer() {
	ticker := scheduler.Heartbeat.NewTicker(env.HeartbeatInterval)
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
}

// heartbeat 作为 Member 节点, 向 Master 发送一次心跳请求
func (n *Node) heartbeat() {
	pool, err := n.rpcClient.getConnPool(n.opts.AdvertiseAddr)
	if err != nil {
		log.Error("Get master conn pool error.", err)
		return
	}
	masterCli := clusterpb.NewMasterClient(pool.Get())
	request := clusterpb.HeartbeatRequest{
		MemberInfo: &clusterpb.MemberInfo{
			Label:       n.opts.Label,
			ServiceAddr: n.opts.ServiceAddr,
			Services:    n.handler.LocalService(),
		},
	}
	if _, err = masterCli.Heartbeat(context.Background(), &request); err != nil {
		log.Error("Member send heartbeat error.", err)
	}
}
