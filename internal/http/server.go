package http

import (
	"context"
	"net"
	"net/http"
	"reflect"
	"sync/atomic"
	"unsafe"

	"github.com/lonng/nano/internal/utils/reflection"
)

var inShutdownOffset uintptr

func init() {
	inShutdownField, found := reflect.TypeOf(http.Server{}).FieldByName("inShutdown")
	if !found {
		panic("The 'inShutdown' field of http.Server is not found.")
	}
	inShutdownOffset = inShutdownField.Offset
}

type Server http.Server

func (srv *Server) shuttingDown() bool {
	inShutdown := (*atomic.Bool)(reflection.Add(unsafe.Pointer(srv), inShutdownOffset))
	return inShutdown.Load()
}

func (srv *Server) Listen() (net.Listener, error) {
	if srv.shuttingDown() {
		return nil, http.ErrServerClosed
	}
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	return net.Listen("tcp", addr)
}

func (srv *Server) Serve(l net.Listener) error {
	return (*http.Server)(srv).Serve(l)
}

func (srv *Server) ListenAndServe() error {
	return (*http.Server)(srv).ListenAndServe()
}

func (srv *Server) Shutdown(ctx context.Context) error {
	return (*http.Server)(srv).Shutdown(ctx)
}

func (srv *Server) Close() error {
	return (*http.Server)(srv).Close()
}
