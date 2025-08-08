//go:build benchmark
// +build benchmark

package io

import (
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/protocal/serialize/protobuf"
	"github.com/lonng/nano/scheduler/defscheduler"
	"github.com/lonng/nano/scheduler/schedulerapi"
	"github.com/lonng/nano/session"
	"github.com/lonng/nano/test/benchmark/io"
	"github.com/lonng/nano/test/benchmark/testdata"
)

const (
	addr          = "127.0.0.1:13250" // local address
	conc          = 2000              // concurrent client count
	routine_count = 500               // number of executors
)

type TestHandler struct {
	component.Base
	metrics int32
	group   *nano.Group
}

// 多协程演示
func (h *TestHandler) Init() {
	var executors []schedulerapi.Executor
	for i := 0; i < routine_count; i++ {
		scheduler := defscheduler.NewScheduler("executor"+strconv.Itoa(routine_count), time.Second)
		scheduler.Start()
		executors = append(executors, scheduler)
	}

	// 实际情况可按照一个房间一个协程
	session.Event.SessionCreated(func(s *session.Session) {
		idx := s.ID() % routine_count
		s.BindExecutor(executors[idx])
	})
}

func (h *TestHandler) AfterInit() {
	ticker := time.NewTicker(time.Second)

	// metrics output ticker
	go func() {
		for range ticker.C {
			println("QPS", atomic.LoadInt32(&h.metrics))
			atomic.StoreInt32(&h.metrics, 0)
		}
	}()
}

func NewTestHandler() *TestHandler {
	return &TestHandler{
		group: nano.NewGroup("handler"),
	}
}

func (h *TestHandler) Ping(s *session.Session, data *testdata.Ping) error {
	atomic.AddInt32(&h.metrics, 1)
	time.Sleep(30 * time.Millisecond) //添加耗时操作
	return s.Push("pong", &testdata.Pong{Content: data.Content})
}

func server() {
	components := &component.Components{}
	components.Register(NewTestHandler())

	e := nano.New(
		nano.WithSerializer(protobuf.NewSerializer()),
		nano.WithComponents(components),
	)
	_ = e.RunTcp(addr)
}

func client() {
	c := io.NewConnector()

	chReady := make(chan struct{})
	c.OnConnected(func() {
		chReady <- struct{}{}
	})

	if err := c.Start(addr); err != nil {
		panic(err)
	}

	c.On("pong", func(data any) {})

	<-chReady
	for /*i := 0; i < 1; i++*/ {
		c.Notify("TestHandler.Ping", &testdata.Ping{})
		time.Sleep(10 * time.Millisecond)
	}
}

func TestIO(t *testing.T) {
	go server()

	// wait server startup
	time.Sleep(1 * time.Second)
	for i := 0; i < conc; i++ {
		go client()
	}

	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)

	<-sg

	t.Log("exit")
}
