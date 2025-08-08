package session

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"

	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/scheduler/schedulerapi"
	"github.com/lonng/nano/session/service"
)

var (
	//ErrIllegalUID 无效的 uid
	ErrIllegalUID = errors.New("illegal uid")
)

// Session 会话表示一个客户端会话，可以在低级保持连接期间存储临时数据，当低级连接断开时，所有数据都会被释放。
// 与客户端相关的会话实例将作为第一个参数传递给构造函数。
type Session struct {
	id         int64                 // 会话 ID(集群模式下, worker 节点中 Node.sessions 中 key 不是该 id)
	uid        atomic.Int64          // 绑定的用户 ID
	entity     NetworkEntity         // 底层网络对象
	router     *Router               // 路由表(service->addr)
	data       map[string]any        // 会话数据
	dataMu     sync.RWMutex          // 会话数据锁
	executor   schedulerapi.Executor // 执行器
	executorMu sync.RWMutex          // 读写 executor 字段锁
	runMu      sync.Mutex            // 并行锁, 防止同会话不同请求调度到各自 executor 并发执行, 造成数据错乱(例如 lastMid 设置)
}

// New 返回新的会话实例 NetworkEntity 是低级网络实例
func New(entity NetworkEntity) *Session {
	s := &Session{
		id:     service.Connections.SessionID(),
		entity: entity,
		router: newRouter(),
		data:   make(map[string]any),
	}
	Event.FireSessionCreated(s)
	return s
}

//==================== 属性 ====================

// ID 返回会话 ID
func (s *Session) ID() int64 {
	return s.id
}

// Bind 绑定用户 ID 到当前会话
func (s *Session) Bind(uid int64) error {
	if uid <= 0 {
		return ErrIllegalUID
	}
	s.uid.Store(uid)
	return nil
}

// UID 返回当前会话绑定的用户 ID
func (s *Session) UID() int64 {
	return s.uid.Load()
}

// LastMid 返回最后一个消息 ID
func (s *Session) LastMid() uint64 {
	return s.entity.LastMid()
}

// NetworkEntity 返回低级网络代理对象
func (s *Session) NetworkEntity() NetworkEntity {
	return s.entity
}

// Router 返回当前会话的路由表
func (s *Session) Router() *Router {
	return s.router
}

// RemoteAddr returns the remote network address.
func (s *Session) RemoteAddr() net.Addr {
	return s.entity.RemoteAddr()
}

//==================== 调度 ====================

// BindExecutor 绑定执行器
func (s *Session) BindExecutor(executor schedulerapi.Executor) {
	if executor == nil {
		return
	}

	s.executorMu.Lock()
	defer s.executorMu.Unlock()

	s.executor = executor
}

// Execute 执行任务
func (s *Session) Execute(task func(), executorFactory ...func() schedulerapi.Executor) {
	// session 级别的执行器
	executor := s.getExecutor()
	if executor != nil && executor.Execute(task) {
		return
	}
	// 外部传入的执行器, 一般是 component 级别的执行器
	for _, fac := range executorFactory {
		if fac == nil {
			continue
		}
		executor = fac()
		if executor != nil && executor.Execute(task) {
			return
		}
	}
	// 全局级别执行器
	scheduler.Execute(task)
}

// LockUnlock 锁定会话, 并返回一个解锁函数, defer s.LockUnlock()()
func (s *Session) LockUnlock() func() {
	s.runMu.Lock()
	return func() {
		s.runMu.Unlock()
	}
}

// getExecutor 获取当前会话的执行器
func (s *Session) getExecutor() schedulerapi.Executor {
	s.executorMu.RLock()
	defer s.executorMu.RUnlock()

	return s.executor
}

// clearExecutor 清除执行器绑定
func (s *Session) clearExecutor() {
	s.executorMu.Lock()
	defer s.executorMu.Unlock()

	s.executor = nil
}

//==================== 通信 ====================

// RPC 将消息发送到远程服务器
func (s *Session) RPC(route string, v any) error {
	return s.entity.RPC(route, v)
}

// Push 推送消息给客户端
func (s *Session) Push(route string, v any) (err error) {
	route, v = Event.FireMessagePushing(s, route, v)
	defer func() {
		if err != nil {
			Event.FireMessagePushFailed(s, route, v, err)
			return
		}
		Event.FireMessagePushed(s, route, v)
	}()
	err = s.entity.Push(route, v)
	return
}

// Response 响应消息给客户端
func (s *Session) Response(v any) error {
	return s.entity.Response(v)
}

// ResponseMID 响应消息给客户端，mid 是请求消息 ID
func (s *Session) ResponseMID(mid uint64, v any) error {
	return s.entity.ResponseMid(mid, v)
}

//==================== 数据 ====================

// Remove 删除与 key 关联的值
func (s *Session) Remove(key string) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	delete(s.data, key)
}

// Set 设置与 key 关联的值
func (s *Session) Set(key string, value any) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	s.data[key] = value
}

// HasKey 获取是否设置了与 key 关联的值
func (s *Session) HasKey(key string) bool {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	_, has := s.data[key]
	return has
}

// Value 返回与 key 关联的值
func (s *Session) Value(key string) any {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	return s.data[key]
}

// Int 返回与 key 关联的值，类型为 int
func (s *Session) Int(key string) int {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(int)
	if !ok {
		return 0
	}
	return value
}

// Int8 返回与 key 关联的值，类型为 int8
func (s *Session) Int8(key string) int8 {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(int8)
	if !ok {
		return 0
	}
	return value
}

// Int16 返回与 key 关联的值，类型为 int16
func (s *Session) Int16(key string) int16 {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(int16)
	if !ok {
		return 0
	}
	return value
}

// Int32 返回与 key 关联的值，类型为 int32
func (s *Session) Int32(key string) int32 {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(int32)
	if !ok {
		return 0
	}
	return value
}

// Int64 返回与 key 关联的值，类型为 int64
func (s *Session) Int64(key string) int64 {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(int64)
	if !ok {
		return 0
	}
	return value
}

// Uint 返回与 key 关联的值，类型为 uint
func (s *Session) Uint(key string) uint {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(uint)
	if !ok {
		return 0
	}
	return value
}

// Uint8 返回与 key 关联的值，类型为 uint8
func (s *Session) Uint8(key string) uint8 {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(uint8)
	if !ok {
		return 0
	}
	return value
}

// Uint16 返回与 key 关联的值，类型为 uint16
func (s *Session) Uint16(key string) uint16 {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(uint16)
	if !ok {
		return 0
	}
	return value
}

// Uint32 返回与 key 关联的值，类型为 uint32
func (s *Session) Uint32(key string) uint32 {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(uint32)
	if !ok {
		return 0
	}
	return value
}

// Uint64 返回与 key 关联的值，类型为 uint64
func (s *Session) Uint64(key string) uint64 {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(uint64)
	if !ok {
		return 0
	}
	return value
}

// Float32 返回与 key 关联的值，类型为 float32
func (s *Session) Float32(key string) float32 {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(float32)
	if !ok {
		return 0
	}
	return value
}

// Float64 返回与 key 关联的值，类型为 float64
func (s *Session) Float64(key string) float64 {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(float64)
	if !ok {
		return 0
	}
	return value
}

// String 返回与 key 关联的值，类型为 string
func (s *Session) String(key string) string {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return ""
	}

	value, ok := v.(string)
	if !ok {
		return ""
	}
	return value
}

// State 返回会话关联的所有键值对
func (s *Session) State() map[string]any {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	return s.data
}

// Restore 设置会话关联的所有键值对
func (s *Session) Restore(data map[string]any) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	s.data = data
}

// clearData 清除所有绑定的键值对
func (s *Session) clearData() {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	s.data = map[string]any{}
}

//==================== 终止 ====================

// Clear 释放与当前会话相关的所有数据(uid, 键值对, 执行器)
func (s *Session) Clear() {
	s.uid.Store(0)
	s.clearData()
	s.clearExecutor()
}

// Close 终止当前会话，会话相关数据将不会被释放，所有相关数据都应在会话关闭回调中显式清除
func (s *Session) Close() {
	_ = s.entity.Close()
}
