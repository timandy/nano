package npi

import "github.com/lonng/nano/internal/utils/reflection"

// HandlerNode holds return values of (*Node).getValue method
type HandlerNode struct {
	name     string
	handlers HandlersChain
}

func NewHandlerNode(handlers ...HandlerFunc) *HandlerNode {
	name := ""
	if len(handlers) > 0 {
		name = reflection.NameOfFunction(handlers[len(handlers)-1])
	}
	return NewHandlerNodeWithName(name, handlers...)
}

func NewHandlerNodeWithName(name string, handlers ...HandlerFunc) *HandlerNode {
	return &HandlerNode{
		name:     name,
		handlers: handlers,
	}
}

func (h *HandlerNode) Name() string {
	return h.name
}

func (h *HandlerNode) Combine(node *HandlerNode) {
	h.handlers = append(h.handlers, node.handlers...)
	h.name = node.name
}

func (h *HandlerNode) Handlers() HandlersChain {
	return h.handlers
}

func (h *HandlerNode) Len() int {
	return len(h.handlers)
}
