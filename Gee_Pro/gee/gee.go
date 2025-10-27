package gee

import (
	"log"
	"net/http"
)

type HandlerFunc func(c *Context)

// Engine实现了ServerHTTP接口，
// 并通过RouterGroup支持路由分组和中间件
type (
	RouterGroup struct {
		// prefix 当前组的公共前缀
		prefix string
		// middlewares 当前组的中间件
		middlewares []HandlerFunc
		//parent 指向父分组 支持嵌套
		parent *RouterGroup
		// 通过engine间接的访问各种接口
		engine *Engine
	}

	Engine struct {
		*RouterGroup
		//存放路由树
		router *router
		//groups记录所有分组
		groups []*RouterGroup
	}
)

// New 创建 Engine 实例，初始化路由器与默认根分组
func New() *Engine {
	//初始化路由表，确保外部拿到实例即可直接注册路由
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

// Group 基于父分组创建新的子分组，继承前缀并使用统一Engine
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		engine: engine,
		parent: group,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

// key 由请求方法和静态路由地址构成
func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)

	group.engine.router.addRoute(method, pattern, handler)
}

// GET方法用于注册GET请求的路由
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

// POST 方法用于注册 POST 请求的路由
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

// Run 用于启动一个 HTTP 服务器
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w, req)
	engine.router.handle(c)
}
