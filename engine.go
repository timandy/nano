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
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/lonng/nano/cluster"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/internal/utils/assert"
	"github.com/lonng/nano/npi"
	"github.com/lonng/nano/scheduler"
)

// VERSION returns current nano version
var VERSION = "0.5.0"

var _ npi.Engine = (*Engine)(nil)

// Engine 引擎
type Engine struct {
	npi.RouterGroup

	trees      npi.HandlerTrees
	allNoRoute npi.HandlersChain
	noRoute    npi.HandlersChain

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
	engine := &Engine{
		trees:      npi.HandlerTrees{},
		allNoRoute: nil,
		noRoute:    nil,
		running:    0,
		opts:       options,
	}
	engine.RouterGroup = npi.NewRootGroup(engine)
	return engine
}

// ServeHTTP conforms to the http.Handler interface.
func (engine *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	engine.node.ServeHTTP(w, r)
}

// NoRoute 为 NoRoute 添加处理程序。默认情况下，它返回一个 404 代码。
func (engine *Engine) NoRoute(handlers ...npi.HandlerFunc) {
	engine.noRoute = handlers
	engine.rebuild404Handlers()
}

// rebuild404Handlers 重新组织 404 处理程序
func (engine *Engine) rebuild404Handlers() {
	engine.allNoRoute = engine.CombineHandlers(engine.noRoute)
}

// AddRoute 添加路由
func (engine *Engine) AddRoute(path string, handlers ...npi.HandlerFunc) {
	assert.Assert(strings.Contains(path, "."), "path should contains '.'")
	assert.Assert(len(handlers) > 0, "there must be at least one handler")

	engine.trees.Append(path, npi.NewHandlerNode(handlers...))
}

// Routes 获取注册的路由信息
func (engine *Engine) Routes() npi.RoutesInfo {
	var routes npi.RoutesInfo
	for path, node := range engine.trees {
		routes = append(routes, npi.RouteInfo{
			Path:        path,
			Handler:     node.Name(),
			HandlerFunc: node.Handlers().Last(),
		})
	}
	return routes
}

// Trees 获取全部路由处理程序
func (engine *Engine) Trees() npi.HandlerTrees {
	return engine.trees
}

// AllNoRoutes 获取无路由的处理程序
func (engine *Engine) AllNoRoutes() npi.HandlersChain {
	return engine.allNoRoute
}

// Register 注册组件
func (engine *Engine) Register(component component.Component, options ...component.Option) {
	engine.opts.Components.Register(component, options...)
}

// Startup 启动引擎
func (engine *Engine) Startup() error {
	if !atomic.CompareAndSwapInt32(&engine.running, 0, 1) {
		return errors.New("nano has running")
	}
	scheduler.Start()
	engine.node = cluster.NewNode(engine, engine.opts)
	return nil
}

// Shutdown 发送信号并关闭 nano
func (engine *Engine) Shutdown() {
	if engine.node != nil {
		engine.node.Shutdown()
	}
	scheduler.Close()
	atomic.StoreInt32(&engine.running, 0)
}

// RunTcp 启动 TCP 服务
func (engine *Engine) RunTcp(addr string) error {
	err := engine.Startup()
	if err != nil {
		return err
	}
	return engine.node.ListenAndServe(addr)
}

// RunWs 启动 WebSocket 服务
func (engine *Engine) RunWs(addr string, path string) error {
	err := engine.Startup()
	if err != nil {
		return err
	}
	return engine.node.ListenAndServeWs(addr, path)
}

// Run 启动 grpc 服务, 并等待退出信号
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
	case <-sg:
	}
	engine.Shutdown()
}
