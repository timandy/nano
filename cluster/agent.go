package cluster

import (
	"errors"
	"fmt"
	"net"
	"sync"
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
	session       *session.Session    // session
	conn          net.Conn            // low-level conn fd
	lastMid       uint64              // last message id
	chDie         chan struct{}       // wait for close
	chSend        chan pendingMessage // push message queue
	lastAt        atomic.Int64        // last heartbeat unix time stamp
	decoder       *packet.Decoder     // binary decoder
	pipeline      pipeline.Pipeline   //
	rpcHandler    rpcHandler          //
	writeReady    atomic.Bool         // write 协程是否已经启动
	connCloseOnce sync.Once           // 确保 conn 只关闭一次
	chanCloseOnce sync.Once           // 确保 chDie 和 chSend 只关闭一次
	state         atomic.Int32        // current agent state
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
		chSend:     make(chan pendingMessage, agentWriteBacklog),
		decoder:    packet.NewDecoder(),
		pipeline:   pipeline,
		rpcHandler: rpcHandler,
	}
	a.lastAt.Store(time.Now().Unix())
	a.state.Store(statusStart)

	// binding session
	s := session.New(a)
	a.session = s
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

// Close 设置关闭状态, 发送关闭信号; 如果 write 协程未就绪, 直接关闭底层连接; 否则由 write 协程 flush 完数据后关闭
func (a *agent) Close() error {
	if a.status() == statusClosed {
		return ErrCloseClosedSession
	}
	a.setStatus(statusClosed)

	if env.Debug {
		log.Info("Session closing, ID=%d, UID=%d, IP=%s", a.session.ID(), a.session.UID(), a.conn.RemoteAddr())
	}

	// 关闭 chan, 发出停止信号
	a.closeChanOnce()

	// 如果 write 协程已经启动, 则不需要关闭底层连接, 因为 write 协程 flush 完会自动处理
	if a.writeReady.Load() {
		return nil
	}

	// 如果 write 协程还未启动, 则直接关闭底层连接
	return a.closeConnOnce()
}

// String 返回描述信息
func (a *agent) String() string {
	return fmt.Sprintf("Remote=%s, LastTime=%d", a.conn.RemoteAddr().String(), a.lastAt.Load())
}

// closeChanOnce 确保 chan 只关闭一次
func (a *agent) closeChanOnce() {
	a.chanCloseOnce.Do(func() {
		close(a.chDie)
		close(a.chSend)
	})
}

// closeConnOnce 确保 conn 只关闭一次
func (a *agent) closeConnOnce() error {
	var err error
	a.connCloseOnce.Do(func() {
		err = a.conn.Close()
	})
	return err
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
		close(chWrite)
		// 关闭 chan, 必须关闭 chan 后才能执行 flush, 否则阻塞
		a.closeChanOnce()
		// 非强制退出, 则将所有待发送的消息写入底层连接
		if !forceQuit {
			a.flush(chWrite)
		}
		// 更改 agent 状态, 必须先更改状态再关闭底层连接
		_ = a.Close()
		// 关闭底层连接, 此时 conn.Read() 将返回错误, 因上一步已经把状态关闭, 所以读协程会跳过日志退出
		_ = a.closeConnOnce()
		if env.Debug {
			log.Info("Session write goroutine exit, SessionID=%d, UID=%d", a.session.ID(), a.session.UID())
		}
	}()

	// 标记 write 协程已经就绪
	a.writeReady.Store(true)

	for {
		select {
		// 心跳检测
		case <-ticker.C:
			deadline := time.Now().Add(-2 * heartbeat).Unix()
			lastAt := a.lastAt.Load()
			if lastAt < deadline {
				log.Info("Session heartbeat timeout, LastTime=%d, Deadline=%d", lastAt, deadline)
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
