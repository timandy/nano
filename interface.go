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
	"sync/atomic"
	"syscall"

	"github.com/lonng/nano/cluster"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/scheduler"
)

// VERSION returns current nano version
var VERSION = "0.5.0"

// Engine 引擎
type Engine struct {
	cluster.Options
	running int32
	node    *cluster.Node
}

// New 创建引擎实例
func New(opts ...Option) *Engine {
	options := cluster.DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}
	return &Engine{
		Options: *options,
		running: 0,
	}
}

// Startup 启动引擎
func (e *Engine) Startup() error {
	if !atomic.CompareAndSwapInt32(&e.running, 0, 1) {
		return errors.New("nano has running")
	}

	opts := &e.Options
	// Use listen address as client address in non-cluster mode
	if opts.SingleMode() {
		log.Println("Nano runs in singleton mode")
	}

	e.node = cluster.NewNode(opts)
	err := e.node.Startup()
	if err != nil {
		return fmt.Errorf("nano node Startup failed: %v", err)
	}
	if e.node.ServiceAddr != "" {
		log.Println(fmt.Sprintf("Nano server started grpc at %s", e.node.ServiceAddr))
	}
	go scheduler.Sched()
	return nil
}

// Shutdown 发送信号并关闭 nano
func (e *Engine) Shutdown() {
	env.Close()
	if e.node != nil {
		e.node.Shutdown()
	}
	scheduler.Close()
	atomic.StoreInt32(&e.running, 0)
}

// WsHandler 返回处理 WebSocket 连接的函数, 只有启动后才能获取
func (e *Engine) WsHandler() http.Handler {
	return e.node.WsHandler()
}

// Listen 启动 WebSocket 服务
func (e *Engine) Listen(addr string, path string) error {
	err := e.Startup()
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle(path, e.WsHandler())
	return http.ListenAndServe(addr, mux)
}

// Run 启动 grpc 服务 和 TCP 服务(指定了 TcpAddr), 并等待退出信号
func (e *Engine) Run() error {
	err := e.Startup()
	if err != nil {
		return err
	}
	e.Wait()
	return nil
}

// Wait 等待退出信号
func (e *Engine) Wait() {
	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)
	select {
	case <-env.DieChan:
		log.Println("Nano server will shutdown in a few seconds")
	case s := <-sg:
		log.Println("Nano server got signal", s)
	}
	log.Println("Nano server is stopping...")
	e.Shutdown()
}
