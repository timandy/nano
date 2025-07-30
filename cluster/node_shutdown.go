package cluster

import (
	"context"
	"net"

	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/internal/env"
	nhttp "github.com/lonng/nano/internal/http"
	"github.com/lonng/nano/internal/log"
)

// Shutdown 停止应用程序注册的所有组件，倒序调用组件的关闭方法，从 master 节点取消注册
func (n *Node) Shutdown() {
	log.Info("==========【Nano is shutting down...】==========")
	defer func() {
		//关闭 gRPC server
		if n.rpcServer != nil {
			n.rpcServer.GracefulStop()
		}
		//等待所有 HTTP 服务器关闭完成
		n.httpServerGroup.Wait()
		//等待监听器关闭完成
		n.listenerGroup.Wait()
		log.Info("==========【Nano already shutdown】==========")
	}()
	// 全局关闭信号
	env.Close()
	// 设置当前节点为关闭状态
	n.inShutdown.Store(true)
	// 关闭所有监听器
	_ = n.closeListenersLocked()
	// 关闭所有 HTTP 服务器
	_ = n.closeHttpServersLocked()
	// 关闭心跳信号
	if n.keepaliveExit != nil {
		close(n.keepaliveExit)
	}

	// 组件卸载回调
	components := n.opts.Components.List()
	length := len(components)
	for i := length - 1; i >= 0; i-- {
		components[i].Comp.BeforeShutdown()
	}

	// 组件卸载回调
	for i := length - 1; i >= 0; i-- {
		components[i].Comp.Shutdown()
	}

	// 向 master 发送取消注册消息
	if n.opts.NodeType.IsMaster() || n.opts.AdvertiseAddr == "" {
		return
	}
	pool, err := n.rpcClient.getConnPool(n.opts.AdvertiseAddr)
	if err != nil {
		log.Error("Retrieve master address error.", err)
		return
	}
	client := clusterpb.NewMasterClient(pool.Get())
	request := &clusterpb.UnregisterRequest{
		ServiceAddr: n.opts.ServiceAddr,
	}
	_, err = client.Unregister(context.Background(), request)
	if err != nil {
		log.Error("Unregister current node failed", err)
	}
}

// shuttingDown 检查当前节点是否正在关闭
func (n *Node) shuttingDown() bool {
	return n.inShutdown.Load()
}

// trackListener 记录当前节点的监听器, 用于在关闭时关闭所有监听器
func (n *Node) trackListener(ln *net.Listener, add bool) bool {
	n.listenersMu.Lock()
	defer n.listenersMu.Unlock()

	if n.listeners == nil {
		n.listeners = make(map[*net.Listener]struct{})
	}
	if add {
		if n.shuttingDown() {
			return false
		}
		n.listeners[ln] = struct{}{}
		n.listenerGroup.Add(1)
	} else {
		delete(n.listeners, ln)
		n.listenerGroup.Done()
	}
	return true
}

// closeListenersLocked 关闭所有监听器
func (n *Node) closeListenersLocked() error {
	n.listenersMu.Lock()
	defer n.listenersMu.Unlock()

	var err error
	for ln := range n.listeners {
		if cerr := (*ln).Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

// trackHttpServer 记录当前节点的 HTTP 服务器, 用于在关闭时关闭所有 HTTP 服务器
func (n *Node) trackHttpServer(server *nhttp.Server, add bool) bool {
	n.httpServersMu.Lock()
	defer n.httpServersMu.Unlock()

	if n.httpServers == nil {
		n.httpServers = make(map[*nhttp.Server]struct{})
	}
	if add {
		if n.shuttingDown() {
			return false
		}
		n.httpServers[server] = struct{}{}
		n.httpServerGroup.Add(1)
	} else {
		delete(n.httpServers, server)
		n.httpServerGroup.Done()
	}
	return true
}

// closeHttpServersLocked 关闭所有 HTTP 服务器, 并等待所有 HTTP 服务器关闭完成
func (n *Node) closeHttpServersLocked() error {
	n.httpServersMu.Lock()
	defer n.httpServersMu.Unlock()

	var err error
	for server := range n.httpServers {
		if cerr := server.Shutdown(context.Background()); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}
