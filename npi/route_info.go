package npi

// RouteInfo represents a request route's specification which contains method and path and its handler.
type RouteInfo struct {
	Route       string      //路由
	Handler     string      //函数名
	HandlerFunc HandlerFunc //函数
}

// RoutesInfo defines a RouteInfo slice.
type RoutesInfo []RouteInfo
