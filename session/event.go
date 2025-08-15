package session

// Event 应该在引擎启动前注册回调函数, 启动后就不要再注册了
var Event = &event{}

// SessionCallback 会话回调
type SessionCallback func(s *Session)

// MessagePreCallback 会话消息处理前回调, 可以修改消息内容和路由
type MessagePreCallback func(s *Session, routePath string, msg any) (newRoutePath string, newMsg any)

// MessagePostCallback 会话消息处理成功回调
type MessagePostCallback func(s *Session, routePath string, msg any)

// MessagePostErrorCallback 会话消息处理错误回调
type MessagePostErrorCallback func(s *Session, routePath string, msg any, err error)

type event struct {
	// 会话创建事件
	onSessionCreated []SessionCallback
	// 会话关闭事件
	onSessionClosed []SessionCallback
	// 消息推送前事件, 可以修改消息内容和路由
	onMessagePushing []MessagePreCallback
	// 消息推送成功事件, 只表示写入队列成功, 实际不一定到达客户端
	onMessagePushed []MessagePostCallback
	// 消息推送失败事件, 只表示写入队列失败, 肯定不会到达客户端
	onMessagePushFailed []MessagePostErrorCallback
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

	// 执行完回调, 清除会话关联的数据
	s.Clear()
}

// MessagePushing 设置消息推送前的回调, 可以修改消息内容和路由
func (lt *event) MessagePushing(callback MessagePreCallback) {
	lt.onMessagePushing = append(lt.onMessagePushing, callback)
}

// FireMessagePushing 触发消息推送前事件, 可以修改消息内容和路由
func (lt *event) FireMessagePushing(s *Session, routePath string, msg any) (string, any) {
	if len(lt.onMessagePushing) == 0 {
		return routePath, msg
	}

	for _, fn := range lt.onMessagePushing {
		routePath, msg = fn(s, routePath, msg)
	}
	return routePath, msg
}

// MessagePushed 设置消息推送成功事件的回调
func (lt *event) MessagePushed(callback MessagePostCallback) {
	lt.onMessagePushed = append(lt.onMessagePushed, callback)
}

// FireMessagePushed 触发消息推送成功事件
func (lt *event) FireMessagePushed(s *Session, routePath string, msg any) {
	if len(lt.onMessagePushed) == 0 {
		return
	}

	for _, fn := range lt.onMessagePushed {
		fn(s, routePath, msg)
	}
}

// MessagePushFailed 设置消息推送失败事件的回调
func (lt *event) MessagePushFailed(callback MessagePostErrorCallback) {
	lt.onMessagePushFailed = append(lt.onMessagePushFailed, callback)
}

// FireMessagePushFailed 触发消息推送失败事件
func (lt *event) FireMessagePushFailed(s *Session, routePath string, msg any, err error) {
	if len(lt.onMessagePushFailed) == 0 {
		return
	}

	for _, fn := range lt.onMessagePushFailed {
		fn(s, routePath, msg, err)
	}
}
