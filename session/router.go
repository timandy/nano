package session

import "sync"

// Router is used to select remote service address
type Router struct {
	routes sync.Map
}

func newRouter() *Router {
	return &Router{}
}

// Bind bound an address to remote service
func (r *Router) Bind(service, address string) {
	r.routes.Store(service, address)
}

// Delete route delete bind for
// 1. some son-service restart, sadly it's not auto bind new son-service
// 2. some game have match to some service, delete bind and match customer service
func (r *Router) Delete(service string) {
	r.routes.Delete(service)
}

// Find finds the address corresponding a remote service
func (r *Router) Find(service string) (string, bool) {
	v, found := r.routes.Load(service)
	if !found {
		return "", false
	}
	return v.(string), true
}
