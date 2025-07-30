package message

import (
	"strings"

	"github.com/lonng/nano/internal/log"
)

var (
	routes = make(map[string]uint16) // route map to code
	codes  = make(map[uint16]string) // code map to route
)

// SetRouteDict 设置路由压缩编码字典
// TODO(warning): 请不要再运行时设置, 危险!!!!!!
func SetRouteDict(dict map[string]uint16) {
	for route, code := range dict {
		r := strings.TrimSpace(route)

		// duplication check
		if _, ok := routes[r]; ok {
			log.Error("duplicated route(route: %s, code: %d)", r, code)
		}

		if _, ok := codes[code]; ok {
			log.Error("duplicated route(route: %s, code: %d)", r, code)
		}

		// update map, using last value when key duplicated
		routes[r] = code
		codes[code] = r
	}
}

// GetRouteDict 获取路由压缩编码字典
func GetRouteDict() (map[string]uint16, bool) {
	if len(routes) <= 0 {
		return nil, false
	}
	return routes, true
}
