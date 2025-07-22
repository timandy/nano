package nap

import (
	"math"
	"reflect"

	"github.com/lonng/nano/binding"
	"github.com/lonng/nano/internal/message"
	"github.com/lonng/nano/internal/utils/reflection"
	"github.com/lonng/nano/session"
)

// ContextType Context 的指针类型
var ContextType = reflect.TypeOf((*Context)(nil))

// abortIndex represents a typical value used in abort functions.
const abortIndex int8 = math.MaxInt8 >> 1

type Context struct {
	Mid         uint64           //消息id
	Service     string           //服务名, 例如路由 Room.Join 的 Service 为 Room
	Msg         *message.Message //消息
	Session     *session.Session //会话
	Errors      errorMsgs        //错误列表
	HandlerNode *HandlerNode     //处理函数
	index       int8             //当前处理函数索引
}

// Reset 从池获取后立即重置
func (c *Context) Reset() {
	c.Mid = 0
	c.Service = ""
	c.Msg = nil
	c.Session = nil
	c.Errors = c.Errors[:0]
	c.HandlerNode = nil
	c.index = -1
}

// HandlerName 函数名
func (c *Context) HandlerName() string {
	return c.HandlerNode.Name()
}

// HandlerNames 函数名列表
func (c *Context) HandlerNames() []string {
	handlers := c.HandlerNode.Handlers()
	hn := make([]string, 0, len(handlers))
	for _, val := range handlers {
		if val == nil {
			continue
		}
		hn = append(hn, reflection.NameOfFunction(val))
	}
	return hn
}

// Handler 返回主要的处理函数
func (c *Context) Handler() HandlerFunc {
	return c.HandlerNode.Handlers().Last()
}

// Next 执行下一个处理函数, 洋葱模型
func (c *Context) Next() {
	c.index++
	handlers := c.HandlerNode.Handlers()
	for c.index < int8(len(handlers)) {
		if handlers[c.index] != nil {
			handlers[c.index](c)
		}
		c.index++
	}
}

// Error 将错误附加到当前上下文
func (c *Context) Error(err error) {
	if err == nil {
		panic("err is nil")
	}
	c.Errors = append(c.Errors, err)
}

// IsAborted 是否已经终止
func (c *Context) IsAborted() bool {
	return c.index >= abortIndex
}

// Abort 终止调用
func (c *Context) Abort() {
	c.index = abortIndex
}

// AbortWithResponse 返回响应并终止
func (c *Context) AbortWithResponse(v any) error {
	err := c.Session.ResponseMID(c.Mid, v)
	c.Abort()
	return err
}

// AbortWithError 返回错误并终止
func (c *Context) AbortWithError(err error) {
	c.Abort()
	c.Error(err)
}

// Response 返回响应
func (c *Context) Response(v any) error {
	return c.Session.ResponseMID(c.Mid, v)
}

// ShouldBind 解析并验证数据
func (c *Context) ShouldBind(obj any) error {
	return binding.ShouldBind(c.Msg, obj)
}
