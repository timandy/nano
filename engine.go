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

package nano

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/lonng/nano/cluster"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/internal/utils/assert"
	"github.com/lonng/nano/nap"
	"github.com/lonng/nano/scheduler"
)

// VERSION returns current nano version
var VERSION = "0.5.0"

var _ nap.Engine = (*Engine)(nil)

// Engine 引擎
type Engine struct {
	nap.RouterGroup

	trees      nap.HandlerTrees
	allNoRoute nap.HandlersChain
	noRoute    nap.HandlersChain

	running int32
	node    *cluster.Node
	opts    *cluster.Options
}

// New 创建引擎实例
func New(opts ...Option) *Engine {
	options := cluster.DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}
	return &Engine{
		RouterGroup: nap.NewRootGroup(),
		trees:       nap.HandlerTrees{},
		allNoRoute:  nil,
		noRoute:     nil,
		running:     0,
		opts:        options,
	}
}

// Options 获取选项
func (engine *Engine) Options() *cluster.Options {
	return engine.opts
}

// NoRoute 为 NoRoute 添加处理程序。默认情况下，它返回一个 404 代码。
func (engine *Engine) NoRoute(handlers ...nap.HandlerFunc) {
	engine.noRoute = handlers
	engine.rebuild404Handlers()
}

// rebuild404Handlers 重新组织 404 处理程序
func (engine *Engine) rebuild404Handlers() {
	engine.allNoRoute = engine.CombineHandlers(engine.noRoute)
}

// AddRoute 添加路由
func (engine *Engine) AddRoute(route string, handlers ...nap.HandlerFunc) {
	assert.Assert(strings.Contains(route, "."), "route should contains '.'")
	assert.Assert(len(handlers) > 0, "there must be at least one handler")

	engine.trees.Append(route, nap.NewHandlerNode(handlers...))
}

// Routes 获取注册的路由信息
func (engine *Engine) Routes() nap.RoutesInfo {
	var routes nap.RoutesInfo
	for route, node := range engine.trees {
		routes = append(routes, nap.RouteInfo{
			Route:       route,
			Handler:     node.Name(),
			HandlerFunc: node.Handlers().Last(),
		})
	}
	return routes
}

// Trees 获取全部路由处理程序
func (engine *Engine) Trees() nap.HandlerTrees {
	return engine.trees
}

// AllNoRoutes 获取无路由的处理程序
func (engine *Engine) AllNoRoutes() nap.HandlersChain {
	return engine.allNoRoute
}

// Startup 启动引擎
func (engine *Engine) Startup() error {
	if !atomic.CompareAndSwapInt32(&engine.running, 0, 1) {
		return errors.New("nano has running")
	}

	opts := engine.opts
	// Use listen address as client address in non-cluster mode
	if opts.SingleMode() {
		log.Println("Nano runs in singleton mode")
	}

	engine.node = cluster.NewNode(engine, opts)
	err := engine.node.Startup()
	if err != nil {
		return fmt.Errorf("nano node Startup failed: %v", err)
	}
	if opts.ServiceAddr != "" {
		log.Println(fmt.Sprintf("Nano server started grpc at %s", opts.ServiceAddr))
	}
	go scheduler.Sched()
	return nil
}

// Shutdown 发送信号并关闭 nano
func (engine *Engine) Shutdown() {
	env.Close()
	if engine.node != nil {
		engine.node.Shutdown()
	}
	scheduler.Close()
	atomic.StoreInt32(&engine.running, 0)
}

// Node 获取引擎的节点, 启动后才有只
func (engine *Engine) Node() *cluster.Node {
	return engine.node
}

// WsHandler 返回处理 WebSocket 连接的函数, 只有启动后才能获取
func (engine *Engine) WsHandler() http.Handler {
	return engine.node.WsHandler()
}

// Listen 启动 WebSocket 服务
func (engine *Engine) Listen(addr string, path string) error {
	err := engine.Startup()
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle(path, engine.WsHandler())
	return http.ListenAndServe(addr, mux)
}

// Run 启动 grpc 服务 和 TCP 服务(指定了 TcpAddr), 并等待退出信号
func (engine *Engine) Run() error {
	err := engine.Startup()
	if err != nil {
		return err
	}
	engine.Wait()
	return nil
}

// Wait 等待退出信号
func (engine *Engine) Wait() {
	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)
	select {
	case <-env.DieChan:
		log.Println("Nano server will shutdown in a few seconds")
	case s := <-sg:
		log.Println("Nano server got signal", s)
	}
	log.Println("Nano server is stopping...")
	engine.Shutdown()
}
