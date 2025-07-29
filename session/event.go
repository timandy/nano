package session

var Event = &event{}

// SessionCallback 会话回调
type SessionCallback func(*Session)

// MessageCallback 会话消息回调
type MessageCallback func(*Session, string, any)

// MessageErrorCallback 会话消息错误回调
type MessageErrorCallback func(*Session, string, any, error)

type event struct {
	// 会话创建事件
	onSessionCreated []SessionCallback
	// 会话关闭事件
	onSessionClosed []SessionCallback
	// 消息推送成功事件, 只表示写入队列成功, 实际不一定到达客户端
	onMessagePushed []MessageCallback
	// 消息推送失败事件
	onMessagePushFailed []MessageErrorCallback
}

// SessionCreated 设置会话创建事件的回调
func (lt *event) SessionCreated(callback SessionCallback) {
	lt.onSessionCreated = append(lt.onSessionCreated, callback)
}

// FireSessionCreated 触发会话创建事件
func (lt *event) FireSessionCreated(s *Session) {
	if len(lt.onSessionCreated) == 0 {
		return
	}

	for _, fn := range lt.onSessionCreated {
		fn(s)
	}
}

// SessionClosed 设置会话关闭事件的回调
func (lt *event) SessionClosed(callback SessionCallback) {
	lt.onSessionClosed = append(lt.onSessionClosed, callback)
}

// FireSessionClosed 触发会话关闭事件
func (lt *event) FireSessionClosed(s *Session) {
	if len(lt.onSessionClosed) == 0 {
		return
	}

	for _, fn := range lt.onSessionClosed {
		fn(s)
	}
}

// MessagePushed 设置消息推送事件的回调
func (lt *event) MessagePushed(callback MessageCallback) {
	lt.onMessagePushed = append(lt.onMessagePushed, callback)
}

// FireMessagePushed 触发消息推送事件
func (lt *event) FireMessagePushed(s *Session, routePath string, v any) {
	if len(lt.onMessagePushed) == 0 {
		return
	}

	for _, fn := range lt.onMessagePushed {
		fn(s, routePath, v)
	}
}

// MessagePushFailed 设置消息推送失败事件的回调
func (lt *event) MessagePushFailed(callback MessageErrorCallback) {
	lt.onMessagePushFailed = append(lt.onMessagePushFailed, callback)
}

// FireMessagePushFailed 触发消息推送失败事件
func (lt *event) FireMessagePushFailed(s *Session, routePath string, v any, err error) {
	if len(lt.onMessagePushFailed) == 0 {
		return
	}

	for _, fn := range lt.onMessagePushFailed {
		fn(s, routePath, v, err)
	}
}
