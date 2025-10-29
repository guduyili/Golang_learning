package gee

import (
	"net/http"
	"strings"
)

type router struct {
	// handlers 保存路由表，键为 "方法-路径"，值为对应的处理函数
	handlers map[string]HandlerFunc
	// roots 保存对应的 Trie 根节点
	roots map[string]*node
}

func newRouter() *router {
	// 初始化路由表
	return &router{
		handlers: make(map[string]HandlerFunc),
		roots:    make(map[string]*node),
	}
}

// 拆分'/'分隔的路由路径为各个部分,唯独*通配符后不再继续拆分
func parsePattern(pattern string) []string {
	tmp := strings.Split(pattern, "/")

	parts := make([]string, 0)
	for _, item := range tmp {
		// 遇到空字符串跳过 避免出现 // 的情况
		if item != "" {
			parts = append(parts, item)
			// 遇到 * 通配符直接停止
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {

	parts := parsePattern(pattern)

	// 组合方法和路径生成唯一键，存储对应处理函数
	key := method + "-" + pattern
	// 获取对应方法的 Trie根节点，不存在则创建
	_, ok := r.roots[method]
	if !ok {
		r.roots[method] = &node{}
	}
	// 将路由pattern插入对应方法的Trie树中
	r.roots[method].insert(pattern, parts, 0)
	// 存储处理函数
	r.handlers[key] = handler

}

// handle 根据请求的 method 和 path 查找对应的处理函数
func (r *router) handle(c *Context) {
	// key := c.Method + "-" + c.Path
	// if handler, ok := r.handlers[key]; ok {
	// 	handler(c)
	// } else {
	// 	c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
	// }

	n, params := r.getRoute(c.Method, c.Path)
	if n != nil {
		key := c.Method + "-" + n.pattern
		// 保存路径参数并将路由处理函数加入中间件链末尾
		c.Params = params
		// r.handlers[key](c)
		c.handlers = append(c.handlers, r.handlers[key])
	} else {
		//未匹配任何路由， 追加一个返回404的处理函数
		c.handlers = append(c.handlers, func(c *Context) {
			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
		})
	}
	c.Next()
}

// getRoute 根据请求的 method 和 path 查找对应的节点和参数
func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	searchParts := parsePattern(path)
	params := make(map[string]string)
	root, ok := r.roots[method]
	if !ok {
		return nil, nil
	}

	n := root.search(searchParts, 0)
	if n != nil {
		parts := parsePattern(n.pattern)
		for index, part := range parts {
			// 处理':'参数
			if part[0] == ':' {
				// :name -> params["name"]
				params[part[1:]] = searchParts[index]
			}
			//处理'*'参数
			if part[0] == '*' && len(part) > 1 {
				// 处理'*'通配符，保存剩余路径
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}
	return nil, nil
}

// getRoutes 获取某个方法下的所有路由节点
func (r *router) getRoutes(method string) []*node {
	// 获取对应方法的 Trie 根节点
	root, ok := r.roots[method]
	if !ok {
		return nil
	}

	//使用travel 深度遍历收集所有节点
	nodes := make([]*node, 0)
	root.travel(&nodes)
	return nodes
}
