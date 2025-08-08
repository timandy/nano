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

// SessionFilter 表示用于在组播时会话的过滤器, 过滤器返回 true 的会话将收到消息
type SessionFilter func(*session.Session) bool

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

// Add 往组中添加会话
func (g *Group) Add(session *session.Session) error {
	if g.isClosed() {
		return ErrClosedGroup
	}

	if env.Debug {
		log.Info("Add session to group %s, ID=%d, UID=%d", g.name, session.ID(), session.UID())
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	id := session.ID()
	_, ok := g.sessions[session.ID()]
	if ok {
		return ErrSessionDuplication
	}

	g.sessions[id] = session
	return nil
}

// Remove 从组中移除会话
func (g *Group) Remove(s *session.Session) error {
	if g.isClosed() {
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

// Clear 删除组中的所有会话
func (g *Group) Clear() error {
	if g.isClosed() {
		return ErrClosedGroup
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

// Contains 检查组中是否包含指定 UID 的会话
func (g *Group) Contains(uid int64) bool {
	_, err := g.Member(uid)
	return err == nil
}

// Members 获取组中所有会话的 UID 列表
func (g *Group) Members() []int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var members []int64
	for _, s := range g.sessions {
		members = append(members, s.UID())
	}

	return members
}

// Member 获取指定 UID 的会话
func (g *Group) Member(uid int64) (*session.Session, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, s := range g.sessions {
		if s.UID() == uid {
			return s, nil
		}
	}

	return nil, ErrMemberNotFound
}

// FindMember 查找满足指定条件的会话
func (g *Group) FindMember(filter func(ses *session.Session) bool) (*session.Session, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, s := range g.sessions {
		if filter(s) {
			return s, nil
		}
	}

	return nil, ErrMemberNotFound
}

// Multicast 将消息发送给满足过滤条件的会话
func (g *Group) Multicast(route string, v any, filter SessionFilter) error {
	if g.isClosed() {
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
		if !filter(s) {
			continue
		}
		if err = s.Push(route, data); err != nil {
			log.Error("Push message error.", err)
		}
	}

	return nil
}

// Broadcast 将消息发送给组中的所有会话
func (g *Group) Broadcast(route string, v any) error {
	if g.isClosed() {
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

// Close 关闭组，释放所有资源
func (g *Group) Close() error {
	if !g.status.CompareAndSwap(groupStatusWorking, groupStatusClosed) {
		return ErrCloseClosedGroup
	}

	// release all reference
	g.sessions = make(map[int64]*session.Session)
	return nil
}

// isClosed 组是否关闭
func (g *Group) isClosed() bool {
	return g.status.Load() == groupStatusClosed
}
