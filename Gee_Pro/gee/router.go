package gee

import (
	"net/http"
)

type router struct {
	// handlers 保存路由表，键为 "方法-路径"，值为对应的处理函数
	handlers map[string]HandlerFunc
}

func newRouter() *router {
	// 初始化路由表
	return &router{handlers: make(map[string]HandlerFunc)}
}

func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	// 组合方法和路径生成唯一键，存储对应处理函数
	key := method + "-" + pattern
	r.handlers[key] = handler
}

// handle 根据请求的 method 和 path 查找对应的处理函数
func (r *router) handle(c *Context) {
	key := c.Method + "-" + c.Path
	if handler, ok := r.handlers[key]; ok {
		handler(c)
	} else {
		c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
	}
}
