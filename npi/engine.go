package npi

type Engine interface {
	IRoutes
	NoRoute(handlers ...HandlerFunc)
	AddRoute(path string, handlers ...HandlerFunc)
	Routes() RoutesInfo
	Trees() HandlerTrees
	AllNoRoutes() HandlersChain
}
