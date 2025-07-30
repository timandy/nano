package cluster

import (
	"errors"
	"net"
	"net/http"

	"github.com/lonng/nano/internal/env"
	nhttp "github.com/lonng/nano/internal/http"
	"github.com/lonng/nano/internal/log"
)

var ErrServerClosed = errors.New("nano: Server closed")

// ListenAndServe 启动 tcp/ip 监听; 对于同一个连接(读是单独协程, 写是单独协程), 所有连接共用一个业务协程
func (n *Node) ListenAndServe(addr string) error {
	// 打开端口
	log.Info("==========【Nano is starting %v】==========", addr)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Info("==========【Nano failed to start, %v】==========", err)
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer ln.Close()

	// 跟踪 listener
	if !n.trackListener(&ln, true) {
		return ErrServerClosed
	}
	defer n.trackListener(&ln, false)

	// 启动服务
	log.Info("==========【Nano already started :%v】==========", ln.Addr().(*net.TCPAddr).Port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			if n.shuttingDown() {
				return ErrServerClosed
			}
			select {
			case <-env.DieChan:
				return ErrServerClosed // 正常退出
			default:
				log.Error("Nano: Accept connection error.", err)
				continue
			}
		}
		go n.handler.handle(conn)
	}
}

// ListenAndServeWs 启动 http/ws 监听; 对于同一个连接(读是单独协程, 写是单独协程), 所有连接共用一个业务协程
func (n *Node) ListenAndServeWs(addr string, path string) error {
	// 构建 http server
	mux := http.NewServeMux()
	mux.Handle(path, n)
	server := &nhttp.Server{Addr: addr, Handler: mux}

	// 打开端口
	log.Info("==========【Nano is starting %v】==========", addr)
	ln, err := server.Listen()
	if err != nil {
		log.Info("==========【Nano failed to start, %v】==========", err)
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer ln.Close()

	// 跟踪 http server
	if !n.trackHttpServer(server, true) {
		return ErrServerClosed
	}
	defer n.trackHttpServer(server, false)

	// 启动服务
	log.Info("==========【Nano already started :%v】==========", ln.Addr().(*net.TCPAddr).Port)
	return server.Serve(ln)
}
