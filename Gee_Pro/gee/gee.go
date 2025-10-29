package gee

import (
	"log"
	"net/http"
	"path"
	"strings"
	"text/template"
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

		//html缓存已解析的模板集合：funcMap存储自定义模板函数
		// funcMap 存储自定模板函数
		htmlTemplates *template.Template
		funcMap       template.FuncMap
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

// Use用于为路由添加中间件
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var middlewares []HandlerFunc
	// 遍历所有分组，找出请求路径匹配的分组
	for _, group := range engine.groups {
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}

	c := newContext(w, req)
	c.handlers = middlewares
	c.engine = engine
	engine.router.handle(c)
}

// createStaticHandler 创建处理静态文件请求的处理函数
func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := group.prefix + relativePath
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filepath")
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		fileServer.ServeHTTP(c.Writer, c.Req)
	}
}

// Static 用于注册静态文件服务的路由
func (group *RouterGroup) Static(relativePath string, root string) {
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	group.GET(urlPattern, handler)
}

// SetFuncMap 提供模板渲染时可调用的自定义函数集合
func (engine *Engine) SetFuncMap(funcMap map[string]interface{}) {
	engine.funcMap = funcMap
}

// LoadHTMLGlob 用于加载并解析指定模式匹配的模板文件
func (engine *Engine) LoadHTMLGlob(pattern string) {
	// 通过模式解析模板，将funcMap注入后缓存到Engine中
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}
