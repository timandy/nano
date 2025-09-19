package nano

import (
	"sync"
	"sync/atomic"

	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/session"
)

const (
	groupStatusWorking = 0
	groupStatusClosed  = 1
)

// SessionFilterFunc 表示用于在组播时会话的过滤器, 过滤器返回 true 的会话将收到消息
type SessionFilterFunc func(*session.Session) bool

// SessionWalkFunc 表示用于在遍历会话的函数, 返回 true 时继续遍历, 返回 false 时停止遍历
type SessionWalkFunc func(*session.Session) bool

// Group 表示一组会话，用于管理多个会话
type Group struct {
	name     string                     // 组名
	status   atomic.Int32               // 组的状态
	sessions map[int64]*session.Session // 组内的会话
	mu       sync.RWMutex               // 组内的会话锁
}

// NewGroup 构造函数
func NewGroup(name string) *Group {
	g := &Group{
		name:     name,
		sessions: make(map[int64]*session.Session),
	}
	g.status.Store(groupStatusWorking)
	return g
}

// Name 返回组名
func (g *Group) Name() string {
	return g.name
}

// Add 往组中添加会话
func (g *Group) Add(s *session.Session) error {
	if g.IsClosed() {
		return ErrClosedGroup
	}

	if env.Debug {
		log.Info("Add session to group %s, ID=%d, UID=%d", g.name, s.ID(), s.UID())
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	id := s.ID()
	_, ok := g.sessions[id]
	if ok {
		return ErrSessionDuplication
	}

	g.sessions[id] = s
	return nil
}

// Remove 从组中移除会话
func (g *Group) Remove(s *session.Session) error {
	if g.IsClosed() {
		return ErrClosedGroup
	}

	if env.Debug {
		log.Info("Remove session from group %s, UID=%d", g.name, s.UID())
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.sessions, s.ID())
	return nil
}

// Contains 检查组中是否包含指定会话
func (g *Group) Contains(s *session.Session) bool {
	return g.ContainsSID(s.ID())
}

// Clear 删除组中的所有会话
func (g *Group) Clear() error {
	if g.IsClosed() {
		return ErrClosedGroup
	}

	if env.Debug {
		log.Info("Clear all session from group %s", g.name)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.sessions = make(map[int64]*session.Session)
	return nil
}

// Count 获取组中会话的数量
func (g *Group) Count() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.sessions)
}

// SIDs 获取组中所有会话的 SID 列表
func (g *Group) SIDs() []int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var sids []int64
	for _, s := range g.sessions {
		sids = append(sids, s.ID())
	}
	return sids
}

// ContainsSID 检查组中是否包含指定 SID 的会话
func (g *Group) ContainsSID(sid int64) bool {
	s := g.FindBySID(sid)
	return s != nil
}

// FindBySID 获取指定 SID 的会话
func (g *Group) FindBySID(sid int64) *session.Session {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.sessions[sid]
}

// UIDs 获取组中所有会话的 UID 列表
func (g *Group) UIDs() []int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var uids []int64
	for _, s := range g.sessions {
		uids = append(uids, s.UID())
	}
	return uids
}

// ContainsUID 检查组中是否包含指定 UID 的会话
func (g *Group) ContainsUID(uid int64) bool {
	s := g.FindByUID(uid)
	return s != nil
}

// FindByUID 获取指定 UID 的会话
func (g *Group) FindByUID(uid int64) *session.Session {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, s := range g.sessions {
		if s.UID() == uid {
			return s
		}
	}
	return nil
}

// Find 查找第一个满足指定条件的会话
func (g *Group) Find(fn SessionFilterFunc) *session.Session {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, s := range g.sessions {
		if fn(s) {
			return s
		}
	}
	return nil
}

// Filter 返回满足过滤条件的会话列表
func (g *Group) Filter(fn SessionFilterFunc) []*session.Session {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var sessions []*session.Session
	for _, s := range g.sessions {
		if fn(s) {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

// Walk 遍历会话, fn 返回 true 时继续遍历, 返回 false 时停止遍历
func (g *Group) Walk(fn SessionWalkFunc) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, s := range g.sessions {
		if !fn(s) {
			return
		}
	}
}

// Multicast 将消息发送给满足过滤条件的会话
func (g *Group) Multicast(route string, v any, fn SessionFilterFunc) error {
	if g.IsClosed() {
		return ErrClosedGroup
	}

	data, err := env.Marshal(v)
	if err != nil {
		return err
	}

	if env.Debug {
		log.Info("Multicast %s, Data=%v", route, v)
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, s := range g.sessions {
		if !fn(s) {
			continue
		}
		if err = s.Push(route, data); err != nil {
			log.Error("Session push message error, ID=%d, UID=%d.", s.ID(), s.UID(), err)
		}
	}

	return err
}

// Broadcast 将消息发送给组中的所有会话
func (g *Group) Broadcast(route string, v any) error {
	if g.IsClosed() {
		return ErrClosedGroup
	}

	data, err := env.Marshal(v)
	if err != nil {
		return err
	}

	if env.Debug {
		log.Info("Broadcast %s, Data=%v", route, v)
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, s := range g.sessions {
		if err = s.Push(route, data); err != nil {
			log.Error("Session push message error, ID=%d, UID=%d.", s.ID(), s.UID(), err)
		}
	}

	return err
}

// MultiKick 将消息发送给满足过滤条件的会话并关闭连接, 被踢出的会话将从组中移除
func (g *Group) MultiKick(v any, fn SessionFilterFunc) error {
	if g.IsClosed() {
		return ErrClosedGroup
	}

	data, err := env.Marshal(v)
	if err != nil {
		return err
	}

	if env.Debug {
		log.Info("MultiKick, Data=%v", v)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	for _, s := range g.sessions {
		if !fn(s) {
			continue
		}
		if err = s.Kick(data); err != nil {
			log.Error("Session kick error, ID=%d, UID=%d.", s.ID(), s.UID(), err)
		}
		// 移除
		delete(g.sessions, s.ID())
	}

	return err
}

// BroadKick 将消息发送给组中的所有会话并关闭连接, 清空组内所有会话
func (g *Group) BroadKick(v any) error {
	if g.IsClosed() {
		return ErrClosedGroup
	}

	data, err := env.Marshal(v)
	if err != nil {
		return err
	}

	if env.Debug {
		log.Info("BroadKick, Data=%v", v)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	for _, s := range g.sessions {
		if err = s.Kick(data); err != nil {
			log.Error("Session kick error, ID=%d, UID=%d.", s.ID(), s.UID(), err)
		}
	}

	// 清空
	g.sessions = make(map[int64]*session.Session)

	return err
}

// IsClosed 组是否关闭
func (g *Group) IsClosed() bool {
	return g.status.Load() == groupStatusClosed
}

// Close 关闭并清空组内所有会话
func (g *Group) Close() error {
	if !g.status.CompareAndSwap(groupStatusWorking, groupStatusClosed) {
		return ErrCloseClosedGroup
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// 关闭所有会话
	for _, s := range g.sessions {
		s.Close()
	}

	// 清空
	g.sessions = make(map[int64]*session.Session)
	return nil
}
