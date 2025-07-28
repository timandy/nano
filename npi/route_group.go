package npi

import (
	"github.com/lonng/nano/internal/utils/assert"
)

type IRoutes interface {
	Group(handlers ...HandlerFunc) IRoutes

	Use(middleware ...HandlerFunc) IRoutes

	Handle(route string, handlers ...HandlerFunc) IRoutes
}

var _ IRoutes = (*RouterGroup)(nil)

type RouterGroup struct {
	Handlers HandlersChain
	engine   Engine
	root     bool
}

func NewRootGroup(engine Engine) RouterGroup {
	return RouterGroup{engine: engine, root: true}
}

func NewRouterGroup(engine Engine, handlers ...HandlerFunc) *RouterGroup {
	return &RouterGroup{
		Handlers: handlers,
		engine:   engine,
		root:     false,
	}
}

func (group *RouterGroup) Group(handlers ...HandlerFunc) IRoutes {
	return &RouterGroup{
		Handlers: group.CombineHandlers(handlers),
		engine:   group.engine,
		root:     false,
	}
}

func (group *RouterGroup) Use(handler ...HandlerFunc) IRoutes {
	group.Handlers = append(group.Handlers, handler...)
	return group.returnObj()
}

func (group *RouterGroup) Handle(route string, handler ...HandlerFunc) IRoutes {
	handlers := group.CombineHandlers(handler)
	group.engine.AddRoute(route, handlers...)
	return group.returnObj()
}

func (group *RouterGroup) CombineHandlers(handlers HandlersChain) HandlersChain {
	finalSize := len(group.Handlers) + len(handlers)
	assert.Assert(finalSize < int(abortIndex), "too many handlers")
	mergedHandlers := make(HandlersChain, finalSize)
	copy(mergedHandlers, group.Handlers)
	copy(mergedHandlers[len(group.Handlers):], handlers)
	return mergedHandlers
}

func (group *RouterGroup) returnObj() IRoutes {
	if group.root {
		return group.engine
	}
	return group
}
