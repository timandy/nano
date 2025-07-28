package npi

import (
	"github.com/lonng/nano/internal/utils/assert"
)

type IRoutes interface {
	BasePath() string

	Group(relativePath string, handlers ...HandlerFunc) *RouterGroup

	Use(middleware ...HandlerFunc) IRoutes

	Handle(route string, handlers ...HandlerFunc) IRoutes
}

var _ IRoutes = (*RouterGroup)(nil)

type RouterGroup struct {
	Handlers HandlersChain
	basePath string
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

func (group *RouterGroup) BasePath() string {
	return group.basePath
}

func (group *RouterGroup) Group(relativePath string, handlers ...HandlerFunc) *RouterGroup {
	return &RouterGroup{
		Handlers: group.CombineHandlers(handlers),
		basePath: group.calculateAbsolutePath(relativePath),
		engine:   group.engine,
		root:     false,
	}
}

func (group *RouterGroup) Use(handler ...HandlerFunc) IRoutes {
	group.Handlers = append(group.Handlers, handler...)
	return group.returnObj()
}

func (group *RouterGroup) Handle(routePath string, handler ...HandlerFunc) IRoutes {
	absolutePath := group.calculateAbsolutePath(routePath)
	handlers := group.CombineHandlers(handler)
	group.engine.AddRoute(absolutePath, handlers...)
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

func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	return JoinPaths(group.basePath, relativePath)
}

func (group *RouterGroup) returnObj() IRoutes {
	if group.root {
		return group.engine
	}
	return group
}
