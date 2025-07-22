package nap

type HandlerTrees map[string]*HandlerNode

func (t HandlerTrees) Append(route string, handlerNode *HandlerNode) {
	if node, found := t[route]; found {
		node.Combine(handlerNode)
		return
	}
	t[route] = handlerNode
}
