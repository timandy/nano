package session

var Lifetime = &lifetime{}

// LifetimeHandler 表示回调
type LifetimeHandler func(*Session)

type lifetime struct {
	// callbacks that emitted on session closed
	onCreated []LifetimeHandler
	// callbacks that emitted on session closed
	onClosed []LifetimeHandler
}

// SessionCreated 设置会话创建事件的回调
func (lt *lifetime) SessionCreated(handler LifetimeHandler) {
	lt.onCreated = append(lt.onCreated, handler)
}

// FireCreated 触发会话创建事件
func (lt *lifetime) FireCreated(s *Session) {
	if len(lt.onCreated) == 0 {
		return
	}

	for _, h := range lt.onCreated {
		h(s)
	}
}

// SessionClosed 设置会话关闭事件的回调
func (lt *lifetime) SessionClosed(handler LifetimeHandler) {
	lt.onClosed = append(lt.onClosed, handler)
}

// FireClosed 触发会话关闭事件
func (lt *lifetime) FireClosed(s *Session) {
	if len(lt.onClosed) == 0 {
		return
	}

	for _, h := range lt.onClosed {
		h(s)
	}
}
