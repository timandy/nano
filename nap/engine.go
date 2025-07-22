package nap

type Engine interface {
	IRoutes
	NoRoute(handlers ...HandlerFunc)
	AddRoute(route string, handlers ...HandlerFunc)
	Routes() RoutesInfo
	Trees() HandlerTrees
	AllNoRoutes() HandlersChain
}
