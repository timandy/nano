package npi

type HandlerTrees map[string]*HandlerNode

func (t HandlerTrees) Append(path string, handlerNode *HandlerNode) {
	if node, found := t[path]; found {
		node.Combine(handlerNode)
		return
	}
	t[path] = handlerNode
}
