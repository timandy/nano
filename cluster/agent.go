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
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/pipeline"
	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/protocal/packet"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
)

const (
	agentWriteBacklog = 16
)

var (
	// ErrBrokenPipe represents the low-level connection has broken.
	ErrBrokenPipe = errors.New("broken low-level pipe")
	// ErrBufferExceed indicates that the current session buffer is full and can not receive more data.
	ErrBufferExceed = errors.New("session send buffer exceed")
)

var _ session.NetworkEntity = (*agent)(nil)

// agent 与客户端直接通信的网络对象(单点模式中的节点, 或集群模式的网关)
type agent struct {
	// regular agent member
	session    *session.Session    // session
	conn       net.Conn            // low-level conn fd
	lastMid    uint64              // last message id
	state      atomic.Int32        // current agent state
	chDie      chan struct{}       // wait for close
	chSend     chan pendingMessage // push message queue
	lastAt     int64               // last heartbeat unix time stamp
	decoder    *packet.Decoder     // binary decoder
	pipeline   pipeline.Pipeline   //
	rpcHandler rpcHandler          //
	srv        reflect.Value       // cached session reflect.Value
}

// pendingMessage 待发送的消息
type pendingMessage struct {
	typ     message.Type // message type
	route   string       // message route(push)
	mid     uint64       // response message id(response)
	payload any          // payload
}

// newAgent 构造函数
func newAgent(conn net.Conn, pipeline pipeline.Pipeline, rpcHandler rpcHandler) *agent {
	a := &agent{
		conn:       conn,
		chDie:      make(chan struct{}),
		lastAt:     time.Now().Unix(),
		chSend:     make(chan pendingMessage, agentWriteBacklog),
		decoder:    packet.NewDecoder(),
		pipeline:   pipeline,
		rpcHandler: rpcHandler,
	}
	a.state.Store(statusStart)

	// binding session
	s := session.New(a)
	a.session = s
	a.srv = reflect.ValueOf(s)
	return a
}

// RemoteAddr 客户端地址, 一般是上游负载均衡的地址
func (a *agent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

// LastMid 上次消息 ID
func (a *agent) LastMid() uint64 {
	return a.lastMid
}

// RPC 调用集群内的服务
func (a *agent) RPC(route string, v any) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}

	// TODO: buffer
	data, err := env.Marshal(v)
	if err != nil {
		return err
	}
	msg := &message.Message{
		Type:  message.Notify,
		Route: route,
		Data:  data,
	}
	a.rpcHandler(a.session, msg, true)
	return nil
}

// Push 推送数据给客户端
func (a *agent) Push(route string, v any) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}

	if len(a.chSend) >= agentWriteBacklog {
		return ErrBufferExceed
	}

	if env.Debug {
		switch d := v.(type) {
		case []byte:
			log.Info("Type=Push, ID=%d, UID=%d, Route=%s, Data=%dbytes", a.session.ID(), a.session.UID(), route, len(d))
		default:
			log.Info("Type=Push, ID=%d, UID=%d, Route=%s, Data=%v", a.session.ID(), a.session.UID(), route, v)
		}
	}

	return a.send(pendingMessage{typ: message.Push, route: route, payload: v})
}

// Response 返回响应数据给客户端
func (a *agent) Response(v any) error {
	return a.ResponseMid(a.lastMid, v)
}

// ResponseMid 返回响应数据给客户端
func (a *agent) ResponseMid(mid uint64, v any) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}

	if mid <= 0 {
		return ErrSessionOnNotify
	}

	if len(a.chSend) >= agentWriteBacklog {
		return ErrBufferExceed
	}

	if env.Debug {
		switch d := v.(type) {
		case []byte:
			log.Info("Type=Response, ID=%d, UID=%d, MID=%d, Data=%dbytes", a.session.ID(), a.session.UID(), mid, len(d))
		default:
			log.Info("Type=Response, ID=%d, UID=%d, MID=%d, Data=%v", a.session.ID(), a.session.UID(), mid, v)
		}
	}

	return a.send(pendingMessage{typ: message.Response, mid: mid, payload: v})
}

// Close 关闭低级连接, 任何被阻塞读取或写入作都将被取消阻塞并返回错误
func (a *agent) Close() error {
	if a.status() == statusClosed {
		return ErrCloseClosedSession
	}
	a.setStatus(statusClosed)

	if env.Debug {
		log.Info("Session closed, ID=%d, UID=%d, IP=%s", a.session.ID(), a.session.UID(), a.conn.RemoteAddr())
	}

	// 防止关闭已经是关闭状态的 chan
	select {
	case <-a.chDie:
		// expect
	default:
		close(a.chDie)
	}

	return a.conn.Close()
}

// String 返回描述信息
func (a *agent) String() string {
	return fmt.Sprintf("Remote=%s, LastTime=%d", a.conn.RemoteAddr().String(), atomic.LoadInt64(&a.lastAt))
}

// status 获取当前状态
func (a *agent) status() int32 {
	return a.state.Load()
}

// setStatus 设置状态
func (a *agent) setStatus(state int32) {
	a.state.Store(state)
}

// write 连接的 write 协程的任务
func (a *agent) write() {
	heartbeat := env.HeartbeatInterval
	ticker := scheduler.Heartbeat.NewTicker(heartbeat)
	chWrite := make(chan []byte, agentWriteBacklog)
	forceQuit := false
	// clean func
	defer func() {
		ticker.Stop()
		close(a.chSend)
		close(chWrite)
		//非强制退出, 则将所有待发送的消息写入底层连接
		if !forceQuit {
			a.flush(chWrite)
		}
		_ = a.Close()
		if env.Debug {
			log.Info("Session write goroutine exit, SessionID=%d, UID=%d", a.session.ID(), a.session.UID())
		}
	}()

	for {
		select {
		// 心跳检测
		case <-ticker.C:
			deadline := time.Now().Add(-2 * heartbeat).Unix()
			if atomic.LoadInt64(&a.lastAt) < deadline {
				log.Info("Session heartbeat timeout, LastTime=%d, Deadline=%d", atomic.LoadInt64(&a.lastAt), deadline)
				return
			}
			chWrite <- getHbd()

		// 封包任务
		case msg := <-a.chSend:
			data := a.packMsg(msg)
			if len(data) == 0 {
				break
			}
			chWrite <- data

		// 发送任务
		case data := <-chWrite:
			// close agent while low-level conn broken
			err := a.writeData(data)
			if err != nil {
				return
			}

		// 会话关闭
		case <-a.chDie: // agent closed signal
			return

		// 引擎退出
		case <-env.DieChan: // application quit
			forceQuit = true
			return
		}
	}
}

// send 将消息放入待发送队列
func (a *agent) send(m pendingMessage) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = ErrBrokenPipe
		}
	}()
	a.chSend <- m
	return
}

// packMsg 将消息封装成可直接写入底层连接的数据包
func (a *agent) packMsg(data pendingMessage) []byte {
	// 序列化
	payload, err := env.Marshal(data.payload)
	if err != nil {
		switch data.typ {
		case message.Push:
			log.Error("Push: %s error.", data.route, err)
		case message.Response:
			log.Error("Response message(id: %d) error.", data.mid, err)
		default:
			// expect
		}
		return nil
	}

	// 构建 Message
	m := &message.Message{
		Type:  data.typ,
		Data:  payload,
		Route: data.route,
		ID:    data.mid,
	}

	// 执行 pipeline
	if pipe := a.pipeline; pipe != nil {
		err = pipe.Outbound().Process(a.session, m)
		if err != nil {
			log.Error("broken pipeline", err)
			return nil
		}
	}

	// 编码 Message
	em, err := m.Encode()
	if err != nil {
		log.Error("Encode msg error.", err)
		return nil
	}

	// 封包
	p, err := packet.Encode(packet.Data, em)
	if err != nil {
		log.Error("Encode packet error.", err)
		return nil
	}
	return p
}

// flush 关闭连接之前, 将所有待发送的消息写入底层连接
func (a *agent) flush(chWrite <-chan []byte) {
	// 处理写入任务
	for data := range chWrite {
		err := a.writeData(data)
		if err != nil {
			return // 底层连接断开, 退出写入
		}
	}
	// 处理封包和写入任务
	for msg := range a.chSend {
		data := a.packMsg(msg)
		if len(data) == 0 {
			continue // 忽略封包失败错误
		}
		err := a.writeData(data)
		if err != nil {
			return // 底层连接断开, 退出写入
		}
	}
}

// writeData 将封好的包写入底层连接
func (a *agent) writeData(data []byte) error {
	if _, err := a.conn.Write(data); err != nil {
		log.Error("Write data to low-level conn error.", err)
		return err
	}
	return nil
}
