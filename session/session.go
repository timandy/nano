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

package session

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/scheduler/schedulerapi"
	"github.com/lonng/nano/session/service"
)

// NetworkEntity represent low-level network instance
type NetworkEntity interface {
	Push(route string, v any) error
	RPC(route string, v any) error
	LastMid() uint64
	Response(v any) error
	ResponseMid(mid uint64, v any) error
	Close() error
	RemoteAddr() net.Addr
}

var (
	//ErrIllegalUID represents a invalid uid
	ErrIllegalUID = errors.New("illegal uid")
)

// Session 会话表示一个客户端会话，可以在低级保持连接期间存储临时数据，当低级连接断开时，所有数据都会被释放。
// 与客户端相关的会话实例将作为第一个参数传递给构造函数。
type Session struct {
	id         int64                 // session global unique id
	uid        int64                 // binding user id
	lastTime   int64                 // last heartbeat time
	entity     NetworkEntity         // low-level network entity
	data       map[string]any        // session data store
	dataMu     sync.RWMutex          // protect data
	executor   schedulerapi.Executor // executor for session tasks
	executorMu sync.RWMutex          // read write executor lock
	router     *Router               // routing table
}

// New 返回新的会话实例 NetworkEntity 是低级网络实例
func New(entity NetworkEntity) *Session {
	s := &Session{
		id:       service.Connections.SessionID(),
		entity:   entity,
		data:     make(map[string]any),
		lastTime: time.Now().Unix(),
		router:   newRouter(),
	}
	Event.FireSessionCreated(s)
	return s
}

// NetworkEntity 返回低级网络代理对象
func (s *Session) NetworkEntity() NetworkEntity {
	return s.entity
}

// Router returns the service router
func (s *Session) Router() *Router {
	return s.router
}

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

// ID returns the session id
func (s *Session) ID() int64 {
	return s.id
}

// UID returns uid that bind to current session
func (s *Session) UID() int64 {
	return atomic.LoadInt64(&s.uid)
}

// LastMid returns the last message id
func (s *Session) LastMid() uint64 {
	return s.entity.LastMid()
}

// Bind bind UID to current session
func (s *Session) Bind(uid int64) error {
	if uid < 1 {
		return ErrIllegalUID
	}

	atomic.StoreInt64(&s.uid, uid)
	return nil
}

// RemoteAddr returns the remote network address.
func (s *Session) RemoteAddr() net.Addr {
	return s.entity.RemoteAddr()
}

// Remove delete data associated with the key from session storage
func (s *Session) Remove(key string) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	delete(s.data, key)
}

// Set associates value with the key in session storage
func (s *Session) Set(key string, value any) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	s.data[key] = value
}

// HasKey decides whether a key has associated value
func (s *Session) HasKey(key string) bool {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	_, has := s.data[key]
	return has
}

// Int returns the value associated with the key as a int.
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

// Int8 returns the value associated with the key as a int8.
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

// Int16 returns the value associated with the key as a int16.
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

// Int32 returns the value associated with the key as a int32.
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

// Int64 returns the value associated with the key as a int64.
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

// Uint returns the value associated with the key as a uint.
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

// Uint8 returns the value associated with the key as a uint8.
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

// Uint16 returns the value associated with the key as a uint16.
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

// Uint32 returns the value associated with the key as a uint32.
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

// Uint64 returns the value associated with the key as a uint64.
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

// Float32 returns the value associated with the key as a float32.
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

// Float64 returns the value associated with the key as a float64.
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

// String returns the value associated with the key as a string.
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

// Value returns the value associated with the key as a any.
func (s *Session) Value(key string) any {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	return s.data[key]
}

// State 返回所有会话所有数据
func (s *Session) State() map[string]any {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	return s.data
}

// Restore 重新连接后的会话数据
func (s *Session) Restore(data map[string]any) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	s.data = data
}

// Clear 释放与当前会话相关的所有数据
func (s *Session) Clear() {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	atomic.StoreInt64(&s.uid, 0)
	s.data = map[string]any{}
}

// Close 终止当前会话，会话相关数据将不会被释放，所有相关数据都应在会话关闭回调中显式清除
func (s *Session) Close() {
	_ = s.entity.Close()
}

// BindExecutor 绑定执行器
func (s *Session) BindExecutor(executor schedulerapi.Executor) {
	if executor == nil {
		return
	}

	s.executorMu.Lock()
	defer s.executorMu.Unlock()

	s.executor = executor
}

// UnbindExecutor 解除执行器
func (s *Session) UnbindExecutor() {
	s.executorMu.Lock()
	defer s.executorMu.Unlock()

	s.executor = nil
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
		if executor == nil {
			continue
		}
		if !executor.Execute(task) {
			continue
		}
	}
	// 全局级别执行器
	scheduler.Execute(task)
}

// getExecutor 获取当前会话的执行器
func (s *Session) getExecutor() schedulerapi.Executor {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	return s.executor
}
